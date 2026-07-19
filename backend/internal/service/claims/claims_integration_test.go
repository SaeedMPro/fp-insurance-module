package claims_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"insurance-module/internal/app/apptest"
	"insurance-module/internal/domain"
	"insurance-module/internal/service/claims"
	"insurance-module/internal/service/employees"
	"insurance-module/internal/service/users"
	"insurance-module/internal/storage/postgres"
	transporthttp "insurance-module/internal/transport/http"
)

// fixture creates (employee + linked employee-user + reviewer) through the
// real services inside the rolled-back test transaction.
type fixture struct {
	employee      domain.Employee
	employeeActor domain.Actor
	reviewerActor domain.Actor
}

func setup(t *testing.T, store *postgres.Store, svcs transporthttp.Services) fixture {
	t.Helper()
	ctx := context.Background()
	// Short unique suffix: national_id is VARCHAR(20), so "WF-NID-" + 12 digits fits.
	suffix := fmt.Sprintf("%012d", time.Now().UnixNano()%1e12)

	plan, err := store.GetPlanByName(ctx, "استاندارد")
	require.NoError(t, err)

	emp, err := svcs.Employees.Create(ctx, employees.CreateInput{
		PersonnelNo: "WF-" + suffix,
		FullName:    "Workflow Employee",
		NationalID:  "WF-NID-" + suffix,
		HireDate:    time.Now().AddDate(-2, 0, 0),
		PlanID:      &plan.ID,
	})
	require.NoError(t, err)

	empUser, err := svcs.Users.Create(ctx, users.CreateInput{
		Username: "wf-emp-" + suffix, Password: "x-test-password",
		FullName: emp.FullName, Role: domain.RoleEmployee, EmployeeID: &emp.ID,
	})
	require.NoError(t, err)

	reviewer, err := svcs.Users.Create(ctx, users.CreateInput{
		Username: "wf-rev-" + suffix, Password: "x-test-password",
		FullName: "Workflow Reviewer", Role: domain.RoleReviewer,
	})
	require.NoError(t, err)

	return fixture{
		employee:      emp,
		employeeActor: domain.Actor{UserID: empUser.ID, Username: empUser.Username, Role: empUser.Role},
		reviewerActor: domain.Actor{UserID: reviewer.ID, Username: reviewer.Username, Role: reviewer.Role},
	}
}

func newClaim(t *testing.T, store *postgres.Store, svcs transporthttp.Services, fx fixture, amount float64) domain.Claim {
	t.Helper()
	st, err := store.GetServiceTypeByCode(context.Background(), "outpatient_visit")
	require.NoError(t, err)
	claim, err := svcs.Claims.Create(context.Background(), fx.employeeActor, claims.CreateInput{
		BeneficiaryType: domain.BeneficiarySelf,
		ServiceTypeID:   st.ID,
		RequestedAmount: amount,
		ReceiptDate:     time.Now(),
	})
	require.NoError(t, err)
	return claim
}

func TestWorkflow_HappyPath_ApprovePaidClose(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()
	fx := setup(t, store, svcs)
	claim := newClaim(t, store, svcs, fx, 400000)

	after, err := svcs.Claims.Submit(ctx, fx.employeeActor, claim.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ClaimSubmitted, after.Status)
	require.NotNil(t, after.SubmittedAt)

	after, err = svcs.Claims.StartReview(ctx, fx.reviewerActor, claim.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ClaimUnderReview, after.Status)

	after, err = svcs.Claims.Approve(ctx, fx.reviewerActor, claim.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ClaimApproved, after.Status)
	require.NotNil(t, after.PayableAmount)
	require.InDelta(t, 280000, *after.PayableAmount, 0.01) // 70% of 400,000 (seeded standard rule)

	after, err = svcs.Claims.MarkPaid(ctx, fx.reviewerActor, claim.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ClaimPaid, after.Status)

	after, err = svcs.Claims.Close(ctx, fx.reviewerActor, claim.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ClaimClosed, after.Status)

	trail, err := svcs.Claims.History(ctx, fx.reviewerActor, claim.ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(trail), 5) // submit, start_review, approve, mark_paid, close
}

func TestWorkflow_RejectRequiresReasonThenCloses(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()
	fx := setup(t, store, svcs)
	claim := newClaim(t, store, svcs, fx, 100000)

	_, err := svcs.Claims.Submit(ctx, fx.employeeActor, claim.ID)
	require.NoError(t, err)
	_, err = svcs.Claims.StartReview(ctx, fx.reviewerActor, claim.ID)
	require.NoError(t, err)

	_, err = svcs.Claims.Reject(ctx, fx.reviewerActor, claim.ID, "")
	require.ErrorIs(t, err, claims.ErrReasonRequired)

	after, err := svcs.Claims.Reject(ctx, fx.reviewerActor, claim.ID, "missing receipt")
	require.NoError(t, err)
	require.Equal(t, domain.ClaimRejected, after.Status)
	require.Equal(t, "missing receipt", after.RejectionReason)

	after, err = svcs.Claims.Close(ctx, fx.reviewerActor, claim.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ClaimClosed, after.Status)
}

func TestWorkflow_ReturnForDocsAndResubmit(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()
	fx := setup(t, store, svcs)
	claim := newClaim(t, store, svcs, fx, 100000)

	_, err := svcs.Claims.Submit(ctx, fx.employeeActor, claim.ID)
	require.NoError(t, err)
	_, err = svcs.Claims.StartReview(ctx, fx.reviewerActor, claim.ID)
	require.NoError(t, err)

	after, err := svcs.Claims.ReturnForDocs(ctx, fx.reviewerActor, claim.ID, "please attach the pharmacy receipt")
	require.NoError(t, err)
	require.Equal(t, domain.ClaimReturnedForDocs, after.Status)

	after, err = svcs.Claims.Resubmit(ctx, fx.employeeActor, claim.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ClaimSubmitted, after.Status)
}

func TestWorkflow_InvalidTransitionRejected(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()
	fx := setup(t, store, svcs)
	claim := newClaim(t, store, svcs, fx, 100000)

	// Claim is still "draft" — approving directly must fail.
	_, err := svcs.Claims.Approve(ctx, fx.reviewerActor, claim.ID)
	require.ErrorIs(t, err, claims.ErrInvalidTransition)
}

func TestWorkflow_ForbiddenActorRejected(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()
	fx := setup(t, store, svcs)
	claim := newClaim(t, store, svcs, fx, 100000)

	_, err := svcs.Claims.Submit(ctx, fx.employeeActor, claim.ID)
	require.NoError(t, err)

	// An employee must not be able to start a review.
	_, err = svcs.Claims.StartReview(ctx, fx.employeeActor, claim.ID)
	require.ErrorIs(t, err, claims.ErrForbidden)
}

func TestClaims_EmployeeSeesOnlyOwnClaims(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()
	fx1 := setup(t, store, svcs)
	fx2 := setup(t, store, svcs)
	c1 := newClaim(t, store, svcs, fx1, 111000)
	_ = newClaim(t, store, svcs, fx2, 222000)

	// Employee 1 lists: only their claim.
	items, _, err := svcs.Claims.List(ctx, fx1.employeeActor, domain.ClaimFilter{Page: domain.NewPage(1, 50)})
	require.NoError(t, err)
	for _, c := range items {
		require.Equal(t, fx1.employeeActor.UserID, c.CreatedBy)
	}

	// Employee 2 cannot read employee 1's claim.
	_, err = svcs.Claims.Get(ctx, fx2.employeeActor, c1.ID)
	require.ErrorIs(t, err, domain.Forbiddenf("not permitted to access this claim"))
}
