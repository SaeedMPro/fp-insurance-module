package ruleengine

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"insurance-module/internal/models"
)

// testDB mirrors the helper in internal/workflow's tests: a real Postgres
// connection (deploy/docker-compose.yml), wrapped in a transaction that is
// always rolled back so fixtures never leak into the seeded reference data.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://insurance:insurance@localhost:5432/insurance?sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("skipping: no test database available (%v)", err)
	}
	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })
	return tx
}

func mustServiceType(t *testing.T, tx *gorm.DB, code string) models.ServiceType {
	t.Helper()
	var st models.ServiceType
	require.NoError(t, tx.Where("code = ?", code).First(&st).Error)
	return st
}

func mustPlan(t *testing.T, tx *gorm.DB, name string) models.CoveragePlan {
	t.Helper()
	var p models.CoveragePlan
	require.NoError(t, tx.Where("name = ?", name).First(&p).Error)
	return p
}

func TestCalculate_EmployeeInactive(t *testing.T) {
	tx := testDB(t)
	ctx := context.Background()
	plan := mustPlan(t, tx, "استاندارد")
	st := mustServiceType(t, tx, "outpatient_visit")

	emp := models.Employee{
		PersonnelNo: "IT-INACTIVE-1", FullName: "Inactive Person",
		EmploymentStatus: models.EmploymentTerminated, HireDate: time.Now().AddDate(-2, 0, 0), PlanID: &plan.ID,
	}
	require.NoError(t, tx.Create(&emp).Error)

	eng := NewEngine(tx)
	_, err := eng.Calculate(ctx, CalcInput{
		EmployeeID: emp.ID, ServiceTypeID: st.ID, PlanID: plan.ID,
		BeneficiaryType: models.BeneficiarySelf, RequestedAmount: 100000, ReceiptDate: time.Now(),
	})
	require.ErrorIs(t, err, ErrEmployeeInactive)
}

func TestCalculate_WaitingPeriodNotMet(t *testing.T) {
	tx := testDB(t)
	ctx := context.Background()
	plan := mustPlan(t, tx, "استاندارد")
	st := mustServiceType(t, tx, "dental") // seeded with a 90-day waiting period

	emp := models.Employee{
		PersonnelNo: "IT-WAITING-1", FullName: "New Hire",
		EmploymentStatus: models.EmploymentActive, HireDate: time.Now().AddDate(0, 0, -10), PlanID: &plan.ID,
	}
	require.NoError(t, tx.Create(&emp).Error)

	eng := NewEngine(tx)
	_, err := eng.Calculate(ctx, CalcInput{
		EmployeeID: emp.ID, ServiceTypeID: st.ID, PlanID: plan.ID,
		BeneficiaryType: models.BeneficiarySelf, RequestedAmount: 100000, ReceiptDate: time.Now(),
	})
	require.ErrorIs(t, err, ErrWaitingPeriod)
}

func TestCalculate_IneligibleRelation(t *testing.T) {
	tx := testDB(t)
	ctx := context.Background()
	plan := mustPlan(t, tx, "استاندارد")
	st := mustServiceType(t, tx, "dental") // seeded eligible_relations = {self,spouse,child}, no "parent"

	emp := models.Employee{
		PersonnelNo: "IT-RELATION-1", FullName: "Long Tenured",
		EmploymentStatus: models.EmploymentActive, HireDate: time.Now().AddDate(-5, 0, 0), PlanID: &plan.ID,
	}
	require.NoError(t, tx.Create(&emp).Error)
	dep := models.Dependent{EmployeeID: emp.ID, FullName: "Their Parent", Relation: models.RelationParent}
	require.NoError(t, tx.Create(&dep).Error)

	eng := NewEngine(tx)
	_, err := eng.Calculate(ctx, CalcInput{
		EmployeeID: emp.ID, ServiceTypeID: st.ID, PlanID: plan.ID,
		BeneficiaryType: models.BeneficiaryDependent, DependentID: &dep.ID,
		RequestedAmount: 100000, ReceiptDate: time.Now(),
	})
	require.ErrorIs(t, err, ErrNotEligible)
}

func TestCalculate_DependentMismatch(t *testing.T) {
	tx := testDB(t)
	ctx := context.Background()
	plan := mustPlan(t, tx, "استاندارد")
	st := mustServiceType(t, tx, "dental")

	emp1 := models.Employee{PersonnelNo: "IT-DEP-1", FullName: "Employee One", NationalID: "IT-DEP-NID-1", EmploymentStatus: models.EmploymentActive, HireDate: time.Now().AddDate(-5, 0, 0), PlanID: &plan.ID}
	require.NoError(t, tx.Create(&emp1).Error)
	emp2 := models.Employee{PersonnelNo: "IT-DEP-2", FullName: "Employee Two", NationalID: "IT-DEP-NID-2", EmploymentStatus: models.EmploymentActive, HireDate: time.Now().AddDate(-5, 0, 0), PlanID: &plan.ID}
	require.NoError(t, tx.Create(&emp2).Error)
	depOfEmp2 := models.Dependent{EmployeeID: emp2.ID, FullName: "Someone Else's Child", Relation: models.RelationChild}
	require.NoError(t, tx.Create(&depOfEmp2).Error)

	eng := NewEngine(tx)
	_, err := eng.Calculate(ctx, CalcInput{
		EmployeeID: emp1.ID, ServiceTypeID: st.ID, PlanID: plan.ID,
		BeneficiaryType: models.BeneficiaryDependent, DependentID: &depOfEmp2.ID,
		RequestedAmount: 100000, ReceiptDate: time.Now(),
	})
	require.ErrorIs(t, err, ErrDependentMismatch)
}

func TestCalculate_NoActiveRuleForUnconfiguredPlanServiceCombo(t *testing.T) {
	tx := testDB(t)
	ctx := context.Background()
	plan := mustPlan(t, tx, "استاندارد")
	st := mustServiceType(t, tx, "outpatient_visit")

	emp := models.Employee{PersonnelNo: "IT-NORULE-1", FullName: "Someone", EmploymentStatus: models.EmploymentActive, HireDate: time.Now().AddDate(-5, 0, 0), PlanID: &plan.ID}
	require.NoError(t, tx.Create(&emp).Error)

	eng := NewEngine(tx)
	// A receipt date before any rule's effective_from window (contract starts 2025-03-21).
	_, err := eng.Calculate(ctx, CalcInput{
		EmployeeID: emp.ID, ServiceTypeID: st.ID, PlanID: plan.ID,
		BeneficiaryType: models.BeneficiarySelf, RequestedAmount: 100000,
		ReceiptDate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	require.ErrorIs(t, err, ErrNoActiveRule)
}

func TestCalculate_EndToEndPricingAndAnnualCapAccumulation(t *testing.T) {
	tx := testDB(t)
	ctx := context.Background()
	plan := mustPlan(t, tx, "استاندارد")
	st := mustServiceType(t, tx, "outpatient_visit") // 70%, per-claim cap 500,000, annual cap 5,000,000

	emp := models.Employee{PersonnelNo: "IT-ACCUM-1", FullName: "Accumulator", EmploymentStatus: models.EmploymentActive, HireDate: time.Now().AddDate(-5, 0, 0), PlanID: &plan.ID}
	require.NoError(t, tx.Create(&emp).Error)
	// Create our own user for the claims' created_by FK so the test is self-contained
	// and does not depend on demo-seed data existing.
	creator := models.User{Username: "it-accum-creator", PasswordHash: "x", FullName: "Creator", Role: models.RoleAdmin, IsActive: true}
	require.NoError(t, tx.Create(&creator).Error)

	eng := NewEngine(tx)

	// First claim: fully within caps.
	res1, err := eng.Calculate(ctx, CalcInput{
		EmployeeID: emp.ID, ServiceTypeID: st.ID, PlanID: plan.ID,
		BeneficiaryType: models.BeneficiarySelf, RequestedAmount: 400000, ReceiptDate: time.Now(),
	})
	require.NoError(t, err)
	require.InDelta(t, 280000, res1.PayableAmount, 0.01)

	// Persist it as "approved" so it counts toward annual-cap usage for the next calculation.
	claim1 := models.Claim{
		EmployeeID: emp.ID, BeneficiaryType: models.BeneficiarySelf, ServiceTypeID: st.ID, PlanID: plan.ID,
		RequestedAmount: 400000, ReceiptDate: time.Now(), Status: models.ClaimApproved,
		PayableAmount: &res1.PayableAmount, CreatedBy: creator.ID,
	}
	require.NoError(t, tx.Create(&claim1).Error)

	// Directly simulate heavy prior usage this year (4,900,000 of the 5,000,000 annual
	// cap already consumed) so the next calculation's remaining cap, not the per-claim
	// cap, is what binds.
	heavyUsage := 4600000.0
	claim2 := models.Claim{
		EmployeeID: emp.ID, BeneficiaryType: models.BeneficiarySelf, ServiceTypeID: st.ID, PlanID: plan.ID,
		RequestedAmount: 6571428.57, ReceiptDate: time.Now(), Status: models.ClaimPaid,
		PayableAmount: &heavyUsage, CreatedBy: creator.ID,
	}
	require.NoError(t, tx.Create(&claim2).Error)
	// Total used so far: 280,000 + 4,600,000 = 4,880,000 of the 5,000,000 annual cap,
	// leaving only 120,000 remaining — well under the 500,000 per-claim cap, so the
	// annual cap is what should bind on the next claim.

	res3, err := eng.Calculate(ctx, CalcInput{
		EmployeeID: emp.ID, ServiceTypeID: st.ID, PlanID: plan.ID,
		BeneficiaryType: models.BeneficiarySelf, RequestedAmount: 300000, ReceiptDate: time.Now(),
	})
	require.NoError(t, err)
	require.InDelta(t, 120000, res3.PayableAmount, 0.01)
	require.True(t, res3.CappedByAnnualCap)
	require.False(t, res3.CappedByPerClaim)
}
