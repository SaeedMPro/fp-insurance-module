// Package workflow implements the multi-stage claim lifecycle described in the
// proposal: submit -> expert review -> decision (approve / reject-with-reason /
// return-for-documents) -> automatic payment calculation -> close. Every
// transition is validated against an explicit state table and written to the
// audit trail (before/after status, actor, reason) in the same transaction.
package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"insurance-module/internal/audit"
	"insurance-module/internal/models"
	"insurance-module/internal/ruleengine"
)

var (
	ErrInvalidTransition = errors.New("workflow: transition not allowed from the claim's current status")
	ErrForbidden         = errors.New("workflow: actor is not permitted to perform this action")
	ErrReasonRequired    = errors.New("workflow: a reason is required for this action")
)

// transitions enumerates every legal (from -> to) status change. Anything not
// listed here is rejected, so the workflow can never skip a step or run backwards
// except through the explicit ReturnedForDocs -> Submitted resubmission path.
var transitions = map[models.ClaimStatus][]models.ClaimStatus{
	models.ClaimDraft:           {models.ClaimSubmitted},
	models.ClaimSubmitted:       {models.ClaimUnderReview},
	models.ClaimUnderReview:     {models.ClaimApproved, models.ClaimRejected, models.ClaimReturnedForDocs},
	models.ClaimReturnedForDocs: {models.ClaimSubmitted},
	models.ClaimApproved:        {models.ClaimPaid},
	models.ClaimRejected:        {models.ClaimClosed},
	models.ClaimPaid:            {models.ClaimClosed},
}

type Engine struct {
	db    *gorm.DB
	rules *ruleengine.Engine
	audit *audit.Service
}

func NewEngine(db *gorm.DB, rules *ruleengine.Engine, auditSvc *audit.Service) *Engine {
	return &Engine{db: db, rules: rules, audit: auditSvc}
}

// Actor identifies who is performing a transition, for both the RBAC checks
// baked into each method below and the audit trail.
type Actor struct {
	UserID   uuid.UUID
	Username string
	Role     models.Role
}

func (e *Engine) getClaim(ctx context.Context, tx *gorm.DB, claimID uuid.UUID) (*models.Claim, error) {
	var claim models.Claim
	if err := tx.WithContext(ctx).First(&claim, "id = ?", claimID).Error; err != nil {
		return nil, err
	}
	return &claim, nil
}

func canTransition(from, to models.ClaimStatus) bool {
	for _, allowed := range transitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// Submit moves a draft claim (created by, or on behalf of, the employee) into the
// review queue. Only the claim's owner may submit it.
func (e *Engine) Submit(ctx context.Context, actor Actor, claimID uuid.UUID) (*models.Claim, error) {
	return e.apply(ctx, actor, claimID, models.ClaimSubmitted, "submit", "", func(tx *gorm.DB, claim *models.Claim) error {
		if actor.Role != models.RoleAdmin && claim.CreatedBy != actor.UserID {
			return ErrForbidden
		}
		now := time.Now()
		claim.SubmittedAt = &now
		return nil
	})
}

// Resubmit moves a returned-for-documents claim back into the queue after the
// employee has completed the requested documents.
func (e *Engine) Resubmit(ctx context.Context, actor Actor, claimID uuid.UUID) (*models.Claim, error) {
	return e.apply(ctx, actor, claimID, models.ClaimSubmitted, "resubmit", "", func(tx *gorm.DB, claim *models.Claim) error {
		if actor.Role != models.RoleAdmin && claim.CreatedBy != actor.UserID {
			return ErrForbidden
		}
		return nil
	})
}

// StartReview is picked up by a reviewer (or admin) claiming a submitted request.
func (e *Engine) StartReview(ctx context.Context, actor Actor, claimID uuid.UUID) (*models.Claim, error) {
	return e.apply(ctx, actor, claimID, models.ClaimUnderReview, "start_review", "", func(tx *gorm.DB, claim *models.Claim) error {
		return requireReviewer(actor)
	})
}

// Approve decides in the claimant's favour. It immediately runs the rule engine
// to price the claim (config-driven percentage/cap/eligibility), so "approved"
// always carries a computed payable_amount — the proposal's automatic ceiling
// calculation happens right here, not as a separate manual step.
func (e *Engine) Approve(ctx context.Context, actor Actor, claimID uuid.UUID) (*models.Claim, error) {
	return e.apply(ctx, actor, claimID, models.ClaimApproved, "approve", "", func(tx *gorm.DB, claim *models.Claim) error {
		if err := requireReviewer(actor); err != nil {
			return err
		}
		result, err := e.rules.Calculate(ctx, ruleengine.CalcInput{
			EmployeeID:      claim.EmployeeID,
			ServiceTypeID:   claim.ServiceTypeID,
			PlanID:          claim.PlanID,
			BeneficiaryType: claim.BeneficiaryType,
			DependentID:     claim.DependentID,
			RequestedAmount: claim.RequestedAmount,
			ReceiptDate:     claim.ReceiptDate,
			ExcludeClaimID:  &claim.ID,
		})
		if err != nil {
			return fmt.Errorf("pricing claim: %w", err)
		}
		claim.CoveragePercentApplied = &result.CoveragePercentApplied
		claim.PayableAmount = &result.PayableAmount
		now := time.Now()
		claim.ReviewedBy = &actor.UserID
		claim.ReviewedAt = &now
		return nil
	})
}

// Reject records the reviewer's decision against the claim, with a mandatory reason.
func (e *Engine) Reject(ctx context.Context, actor Actor, claimID uuid.UUID, reason string) (*models.Claim, error) {
	if reason == "" {
		return nil, ErrReasonRequired
	}
	return e.apply(ctx, actor, claimID, models.ClaimRejected, "reject", reason, func(tx *gorm.DB, claim *models.Claim) error {
		if err := requireReviewer(actor); err != nil {
			return err
		}
		now := time.Now()
		claim.ReviewedBy = &actor.UserID
		claim.ReviewedAt = &now
		claim.RejectionReason = reason
		return nil
	})
}

// ReturnForDocs sends the claim back to the employee for missing/incomplete documents.
func (e *Engine) ReturnForDocs(ctx context.Context, actor Actor, claimID uuid.UUID, reason string) (*models.Claim, error) {
	if reason == "" {
		return nil, ErrReasonRequired
	}
	return e.apply(ctx, actor, claimID, models.ClaimReturnedForDocs, "return_for_docs", reason, func(tx *gorm.DB, claim *models.Claim) error {
		if err := requireReviewer(actor); err != nil {
			return err
		}
		claim.RejectionReason = reason
		return nil
	})
}

// MarkPaid simulates disbursement of the approved payable amount (the proposal
// explicitly puts a real payment gateway out of scope) and records a Payment row.
func (e *Engine) MarkPaid(ctx context.Context, actor Actor, claimID uuid.UUID) (*models.Claim, error) {
	return e.apply(ctx, actor, claimID, models.ClaimPaid, "mark_paid", "", func(tx *gorm.DB, claim *models.Claim) error {
		if err := requireReviewer(actor); err != nil {
			return err
		}
		if claim.PayableAmount == nil {
			return fmt.Errorf("workflow: claim has no payable amount to disburse")
		}
		now := time.Now()
		claim.PaidAt = &now
		payment := models.Payment{
			ClaimID:          claim.ID,
			Amount:           *claim.PayableAmount,
			PaymentReference: fmt.Sprintf("SIM-%s", uuid.NewString()[:8]),
			Status:           models.PaymentSimulated,
			PaidAt:           now,
		}
		return tx.Create(&payment).Error
	})
}

// Close terminates the claim's lifecycle (from either a rejected or a paid state).
func (e *Engine) Close(ctx context.Context, actor Actor, claimID uuid.UUID) (*models.Claim, error) {
	return e.apply(ctx, actor, claimID, models.ClaimClosed, "close", "", func(tx *gorm.DB, claim *models.Claim) error {
		if err := requireReviewer(actor); err != nil {
			return err
		}
		now := time.Now()
		claim.ClosedAt = &now
		return nil
	})
}

func requireReviewer(actor Actor) error {
	if actor.Role != models.RoleReviewer && actor.Role != models.RoleAdmin {
		return ErrForbidden
	}
	return nil
}

// apply is the shared transition runner: it loads the claim inside a transaction,
// checks the state table, lets the caller mutate the row and enforce its own RBAC
// rule, persists the change, and writes one audit log entry — all atomically.
func (e *Engine) apply(
	ctx context.Context,
	actor Actor,
	claimID uuid.UUID,
	to models.ClaimStatus,
	action string,
	reason string,
	mutate func(tx *gorm.DB, claim *models.Claim) error,
) (*models.Claim, error) {
	var result *models.Claim
	err := e.db.Transaction(func(tx *gorm.DB) error {
		claim, err := e.getClaim(ctx, tx, claimID)
		if err != nil {
			return err
		}
		if !canTransition(claim.Status, to) {
			return ErrInvalidTransition
		}
		before := map[string]interface{}{"status": claim.Status}

		if err := mutate(tx, claim); err != nil {
			return err
		}
		claim.Status = to
		claim.UpdatedAt = time.Now()

		if err := tx.Save(claim).Error; err != nil {
			return err
		}

		after := map[string]interface{}{"status": claim.Status}
		if reason != "" {
			after["reason"] = reason
		}
		if claim.PayableAmount != nil {
			after["payable_amount"] = *claim.PayableAmount
		}
		if err := e.audit.Log(ctx, tx, audit.Entry{
			EntityType:    "claim",
			EntityID:      claim.ID.String(),
			Action:        action,
			ActorUserID:   &actor.UserID,
			ActorUsername: actor.Username,
			Before:        before,
			After:         after,
		}); err != nil {
			return err
		}

		result = claim
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
