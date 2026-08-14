package coverage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"insurance-module/internal/app/apptest"
	"insurance-module/internal/domain"
	"insurance-module/internal/service/coverage"
	"insurance-module/internal/service/employees"
	"insurance-module/internal/service/users"
	"insurance-module/internal/storage/postgres"
	transporthttp "insurance-module/internal/transport/http"
)

// fixtures builds an employee on the seeded "استاندارد" plan through the real
// services, inside the test transaction.
func newEmployee(t *testing.T, store *postgres.Store, svcs transporthttp.Services, personnelNo, nationalID string, hireDate time.Time) domain.Employee {
	t.Helper()
	plan, err := store.GetPlanByName(context.Background(), "استاندارد")
	require.NoError(t, err)
	emp, err := svcs.Employees.Create(context.Background(), employees.CreateInput{
		PersonnelNo: personnelNo,
		FullName:    "Test " + personnelNo,
		NationalID:  nationalID,
		HireDate:    hireDate,
		PlanID:      &plan.ID,
	})
	require.NoError(t, err)
	return emp
}

func serviceTypeID(t *testing.T, store *postgres.Store, code string) domain.ServiceType {
	t.Helper()
	st, err := store.GetServiceTypeByCode(context.Background(), code)
	require.NoError(t, err)
	return st
}

func TestCalculate_EmployeeInactive(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()

	emp := newEmployee(t, store, svcs, "IT-INACTIVE-1", "IT-NID-INACT", time.Now().AddDate(-2, 0, 0))
	terminated := domain.EmploymentTerminated
	_, err := svcs.Employees.Update(ctx, emp.ID, employees.UpdateInput{EmploymentStatus: &terminated})
	require.NoError(t, err)

	st := serviceTypeID(t, store, "outpatient_visit")
	_, err = svcs.Coverage.Calculate(ctx, coverage.CalcInput{
		EmployeeID: emp.ID, ServiceTypeID: st.ID, PlanID: *emp.PlanID,
		BeneficiaryType: domain.BeneficiarySelf, RequestedAmount: 100000, ReceiptDate: time.Now(),
	})
	require.ErrorIs(t, err, coverage.ErrEmployeeInactive)
}

func TestCalculate_WaitingPeriodNotMet(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()

	// dental is seeded with a 90-day waiting period; hire 10 days ago.
	emp := newEmployee(t, store, svcs, "IT-WAITING-1", "IT-NID-WAIT", time.Now().AddDate(0, 0, -10))
	st := serviceTypeID(t, store, "dental")

	_, err := svcs.Coverage.Calculate(ctx, coverage.CalcInput{
		EmployeeID: emp.ID, ServiceTypeID: st.ID, PlanID: *emp.PlanID,
		BeneficiaryType: domain.BeneficiarySelf, RequestedAmount: 100000, ReceiptDate: time.Now(),
	})
	require.ErrorIs(t, err, coverage.ErrWaitingPeriod)
}

func TestCalculate_IneligibleRelation(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()

	// dental's seeded relations are {self, spouse, child} — no parent.
	emp := newEmployee(t, store, svcs, "IT-RELATION-1", "IT-NID-REL", time.Now().AddDate(-5, 0, 0))
	dep, err := svcs.Employees.CreateDependent(ctx, emp.ID, employees.CreateDependentInput{
		FullName: "Their Parent", Relation: domain.RelationParent,
	})
	require.NoError(t, err)
	st := serviceTypeID(t, store, "dental")

	_, err = svcs.Coverage.Calculate(ctx, coverage.CalcInput{
		EmployeeID: emp.ID, ServiceTypeID: st.ID, PlanID: *emp.PlanID,
		BeneficiaryType: domain.BeneficiaryDependent, DependentID: &dep.ID,
		RequestedAmount: 100000, ReceiptDate: time.Now(),
	})
	require.ErrorIs(t, err, coverage.ErrNotEligible)
}

func TestCalculate_DependentMismatch(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()

	emp1 := newEmployee(t, store, svcs, "IT-DEP-1", "IT-NID-DEP1", time.Now().AddDate(-5, 0, 0))
	emp2 := newEmployee(t, store, svcs, "IT-DEP-2", "IT-NID-DEP2", time.Now().AddDate(-5, 0, 0))
	depOfEmp2, err := svcs.Employees.CreateDependent(ctx, emp2.ID, employees.CreateDependentInput{
		FullName: "Someone Else's Child", Relation: domain.RelationChild,
	})
	require.NoError(t, err)
	st := serviceTypeID(t, store, "dental")

	_, err = svcs.Coverage.Calculate(ctx, coverage.CalcInput{
		EmployeeID: emp1.ID, ServiceTypeID: st.ID, PlanID: *emp1.PlanID,
		BeneficiaryType: domain.BeneficiaryDependent, DependentID: &depOfEmp2.ID,
		RequestedAmount: 100000, ReceiptDate: time.Now(),
	})
	require.ErrorIs(t, err, coverage.ErrDependentMismatch)
}

func TestCalculate_NoActiveRuleBeforeContractStart(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()

	emp := newEmployee(t, store, svcs, "IT-NORULE-1", "IT-NID-NORULE", time.Now().AddDate(-5, 0, 0))
	st := serviceTypeID(t, store, "outpatient_visit")

	// Receipt date predates every rule's effective window (contract starts 2025-03-21).
	_, err := svcs.Coverage.Calculate(ctx, coverage.CalcInput{
		EmployeeID: emp.ID, ServiceTypeID: st.ID, PlanID: *emp.PlanID,
		BeneficiaryType: domain.BeneficiarySelf, RequestedAmount: 100000,
		ReceiptDate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	require.ErrorIs(t, err, coverage.ErrNoActiveRule)
}

func TestPublishRuleVersion_SameDayRepublishAndTiebreak(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()

	plan, err := store.GetPlanByName(ctx, "استاندارد")
	require.NoError(t, err)
	st := serviceTypeID(t, store, "optometry")
	admin := newAdminActor(t, svcs)

	today := time.Now().Truncate(24 * time.Hour)
	publish := func(pct float64) domain.CoverageRule {
		rule, err := svcs.Coverage.PublishRuleVersion(ctx, admin, coverage.PublishRuleInput{
			PlanID: plan.ID, ServiceTypeID: st.ID, CoveragePercent: domain.PercentFromFloat(pct),
			WaitingPeriodDays: 0, EligibleRelations: []domain.Relation{domain.RelationSelf},
			EffectiveFrom: today,
		})
		require.NoError(t, err)
		return rule
	}

	publish(61)
	// Second publish with the SAME effective date must not violate the CHECK
	// constraint, and the newest version must win the tiebreak.
	publish(63)

	active, err := store.ActiveRule(ctx, plan.ID, st.ID, today)
	require.NoError(t, err)
	require.Equal(t, domain.PercentFromFloat(63), active.CoveragePercent)

	// Exactly one open version remains.
	rules, err := svcs.Coverage.ListRules(ctx, domain.RuleFilter{PlanID: plan.ID.String(), ServiceTypeID: st.ID.String()})
	require.NoError(t, err)
	open := 0
	for _, r := range rules {
		if r.EffectiveTo == nil {
			open++
		}
	}
	require.Equal(t, 1, open)
}

// newAdminActor returns the seeded bootstrap admin (API Create cannot make admins).
func newAdminActor(t *testing.T, svcs transporthttp.Services) domain.Actor {
	t.Helper()
	u, created, err := svcs.Users.EnsureAdmin(context.Background(), users.CreateInput{
		Username: "admin",
		Password: "Admin123!",
		FullName: "مدیر سامانه",
	})
	require.NoError(t, err)
	require.False(t, created, "test DB should already have the seeded admin")
	return domain.Actor{UserID: u.ID, Username: u.Username, Role: u.Role}
}
