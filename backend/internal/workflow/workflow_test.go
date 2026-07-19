package workflow

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"insurance-module/internal/audit"
	"insurance-module/internal/models"
	"insurance-module/internal/ruleengine"
)

// testDB opens a connection to a real Postgres instance (started via
// deploy/docker-compose.yml) and runs the whole test inside one outer
// transaction that is always rolled back, so the seeded reference data
// and other tests are never polluted by these fixtures.
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

// seedFixtures creates a plan/service-type-agnostic employee + user against the
// already-migrated+seeded "استاندارد" plan / "outpatient_visit" service type.
func seedFixtures(t *testing.T, tx *gorm.DB) (employee models.Employee, reviewer models.User) {
	t.Helper()

	var plan models.CoveragePlan
	require.NoError(t, tx.Where("name = ?", "استاندارد").First(&plan).Error)

	employee = models.Employee{
		PersonnelNo:      "WF-" + time.Now().Format("150405.000000"),
		FullName:         "Test Employee",
		EmploymentStatus: models.EmploymentActive,
		HireDate:         time.Now().AddDate(-2, 0, 0),
		PlanID:           &plan.ID,
	}
	require.NoError(t, tx.Create(&employee).Error)

	empUser := models.User{
		Username:     "employee_" + employee.PersonnelNo,
		PasswordHash: "x",
		FullName:     employee.FullName,
		Role:         models.RoleEmployee,
		EmployeeID:   &employee.ID,
		IsActive:     true,
	}
	require.NoError(t, tx.Create(&empUser).Error)

	reviewer = models.User{
		Username:     "reviewer_" + employee.PersonnelNo,
		PasswordHash: "x",
		FullName:     "Test Reviewer",
		Role:         models.RoleReviewer,
		IsActive:     true,
	}
	require.NoError(t, tx.Create(&reviewer).Error)

	return employee, reviewer
}

func newClaim(t *testing.T, tx *gorm.DB, employee models.Employee, createdBy uuid.UUID, amount float64) models.Claim {
	t.Helper()
	var st models.ServiceType
	require.NoError(t, tx.Where("code = ?", "outpatient_visit").First(&st).Error)

	claim := models.Claim{
		EmployeeID:      employee.ID,
		BeneficiaryType: models.BeneficiarySelf,
		ServiceTypeID:   st.ID,
		PlanID:          *employee.PlanID,
		RequestedAmount: amount,
		ReceiptDate:     time.Now(),
		Status:          models.ClaimDraft,
		CreatedBy:       createdBy,
	}
	require.NoError(t, tx.Create(&claim).Error)
	return claim
}

func TestWorkflow_HappyPath_ApprovePaidClose(t *testing.T) {
	tx := testDB(t)
	ctx := context.Background()

	employee, reviewerUser := seedFixtures(t, tx)
	rules := ruleengine.NewEngine(tx)
	auditSvc := audit.NewService(tx)
	eng := NewEngine(tx, rules, auditSvc)

	var empUser models.User
	require.NoError(t, tx.Where("employee_id = ?", employee.ID).First(&empUser).Error)

	claim := newClaim(t, tx, employee, empUser.ID, 400000)

	employeeActor := Actor{UserID: empUser.ID, Username: empUser.Username, Role: models.RoleEmployee}
	reviewerActor := Actor{UserID: reviewerUser.ID, Username: reviewerUser.Username, Role: models.RoleReviewer}

	claimAfter, err := eng.Submit(ctx, employeeActor, claim.ID)
	require.NoError(t, err)
	require.Equal(t, models.ClaimSubmitted, claimAfter.Status)

	claimAfter, err = eng.StartReview(ctx, reviewerActor, claim.ID)
	require.NoError(t, err)
	require.Equal(t, models.ClaimUnderReview, claimAfter.Status)

	claimAfter, err = eng.Approve(ctx, reviewerActor, claim.ID)
	require.NoError(t, err)
	require.Equal(t, models.ClaimApproved, claimAfter.Status)
	require.NotNil(t, claimAfter.PayableAmount)
	require.InDelta(t, 280000, *claimAfter.PayableAmount, 0.01) // 70% of 400,000

	claimAfter, err = eng.MarkPaid(ctx, reviewerActor, claim.ID)
	require.NoError(t, err)
	require.Equal(t, models.ClaimPaid, claimAfter.Status)

	var payment models.Payment
	require.NoError(t, tx.Where("claim_id = ?", claim.ID).First(&payment).Error)
	require.InDelta(t, 280000, payment.Amount, 0.01)

	claimAfter, err = eng.Close(ctx, reviewerActor, claim.ID)
	require.NoError(t, err)
	require.Equal(t, models.ClaimClosed, claimAfter.Status)

	trail, err := auditSvc.Trail(ctx, "claim", claim.ID.String())
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(trail), 5) // submit, start_review, approve, mark_paid, close
}

func TestWorkflow_RejectPath(t *testing.T) {
	tx := testDB(t)
	ctx := context.Background()

	employee, reviewerUser := seedFixtures(t, tx)
	rules := ruleengine.NewEngine(tx)
	auditSvc := audit.NewService(tx)
	eng := NewEngine(tx, rules, auditSvc)

	var empUser models.User
	require.NoError(t, tx.Where("employee_id = ?", employee.ID).First(&empUser).Error)
	claim := newClaim(t, tx, employee, empUser.ID, 100000)

	employeeActor := Actor{UserID: empUser.ID, Username: empUser.Username, Role: models.RoleEmployee}
	reviewerActor := Actor{UserID: reviewerUser.ID, Username: reviewerUser.Username, Role: models.RoleReviewer}

	_, err := eng.Submit(ctx, employeeActor, claim.ID)
	require.NoError(t, err)
	_, err = eng.StartReview(ctx, reviewerActor, claim.ID)
	require.NoError(t, err)

	_, err = eng.Reject(ctx, reviewerActor, claim.ID, "")
	require.ErrorIs(t, err, ErrReasonRequired)

	claimAfter, err := eng.Reject(ctx, reviewerActor, claim.ID, "missing receipt")
	require.NoError(t, err)
	require.Equal(t, models.ClaimRejected, claimAfter.Status)

	claimAfter, err = eng.Close(ctx, reviewerActor, claim.ID)
	require.NoError(t, err)
	require.Equal(t, models.ClaimClosed, claimAfter.Status)
}

func TestWorkflow_ReturnForDocsAndResubmit(t *testing.T) {
	tx := testDB(t)
	ctx := context.Background()

	employee, reviewerUser := seedFixtures(t, tx)
	rules := ruleengine.NewEngine(tx)
	auditSvc := audit.NewService(tx)
	eng := NewEngine(tx, rules, auditSvc)

	var empUser models.User
	require.NoError(t, tx.Where("employee_id = ?", employee.ID).First(&empUser).Error)
	claim := newClaim(t, tx, employee, empUser.ID, 100000)

	employeeActor := Actor{UserID: empUser.ID, Username: empUser.Username, Role: models.RoleEmployee}
	reviewerActor := Actor{UserID: reviewerUser.ID, Username: reviewerUser.Username, Role: models.RoleReviewer}

	_, err := eng.Submit(ctx, employeeActor, claim.ID)
	require.NoError(t, err)
	_, err = eng.StartReview(ctx, reviewerActor, claim.ID)
	require.NoError(t, err)

	claimAfter, err := eng.ReturnForDocs(ctx, reviewerActor, claim.ID, "please attach the pharmacy receipt")
	require.NoError(t, err)
	require.Equal(t, models.ClaimReturnedForDocs, claimAfter.Status)

	claimAfter, err = eng.Resubmit(ctx, employeeActor, claim.ID)
	require.NoError(t, err)
	require.Equal(t, models.ClaimSubmitted, claimAfter.Status)
}

func TestWorkflow_InvalidTransitionRejected(t *testing.T) {
	tx := testDB(t)
	ctx := context.Background()

	employee, reviewerUser := seedFixtures(t, tx)
	rules := ruleengine.NewEngine(tx)
	auditSvc := audit.NewService(tx)
	eng := NewEngine(tx, rules, auditSvc)

	var empUser models.User
	require.NoError(t, tx.Where("employee_id = ?", employee.ID).First(&empUser).Error)
	claim := newClaim(t, tx, employee, empUser.ID, 100000)

	reviewerActor := Actor{UserID: reviewerUser.ID, Username: reviewerUser.Username, Role: models.RoleReviewer}

	// Claim is still "draft" — approving directly must fail.
	_, err := eng.Approve(ctx, reviewerActor, claim.ID)
	require.ErrorIs(t, err, ErrInvalidTransition)
}

func TestWorkflow_ForbiddenActorRejected(t *testing.T) {
	tx := testDB(t)
	ctx := context.Background()

	employee, _ := seedFixtures(t, tx)
	rules := ruleengine.NewEngine(tx)
	auditSvc := audit.NewService(tx)
	eng := NewEngine(tx, rules, auditSvc)

	var empUser models.User
	require.NoError(t, tx.Where("employee_id = ?", employee.ID).First(&empUser).Error)
	claim := newClaim(t, tx, employee, empUser.ID, 100000)

	employeeActor := Actor{UserID: empUser.ID, Username: empUser.Username, Role: models.RoleEmployee}
	_, err := eng.Submit(ctx, employeeActor, claim.ID)
	require.NoError(t, err)

	// An employee (not a reviewer/admin) must not be able to start a review.
	_, err = eng.StartReview(ctx, employeeActor, claim.ID)
	require.ErrorIs(t, err, ErrForbidden)
}
