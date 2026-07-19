package claims

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
	"insurance-module/internal/service/coverage"
)

// Submit moves a draft claim into the review queue. Only the claim's owner
// (or an admin) may submit it.
func (s *Service) Submit(ctx context.Context, actor domain.Actor, claimID uuid.UUID) (domain.Claim, error) {
	return s.apply(ctx, actor, claimID, domain.ClaimSubmitted, "submit", "", func(r Repo, claim *domain.Claim) error {
		if actor.Role != domain.RoleAdmin && claim.CreatedBy != actor.UserID {
			return ErrForbidden
		}
		now := s.clock.Now()
		claim.SubmittedAt = &now
		return nil
	})
}

// Resubmit returns a returned-for-documents claim to the queue.
func (s *Service) Resubmit(ctx context.Context, actor domain.Actor, claimID uuid.UUID) (domain.Claim, error) {
	return s.apply(ctx, actor, claimID, domain.ClaimSubmitted, "resubmit", "", func(r Repo, claim *domain.Claim) error {
		if actor.Role != domain.RoleAdmin && claim.CreatedBy != actor.UserID {
			return ErrForbidden
		}
		return nil
	})
}

// StartReview lets a reviewer (or admin) claim a submitted request.
func (s *Service) StartReview(ctx context.Context, actor domain.Actor, claimID uuid.UUID) (domain.Claim, error) {
	return s.apply(ctx, actor, claimID, domain.ClaimUnderReview, "start_review", "", func(r Repo, claim *domain.Claim) error {
		return requireReviewer(actor)
	})
}

// Approve decides in the claimant's favour and immediately prices the claim
// via the coverage service — "approved" always carries a computed payable
// amount; the automatic ceiling calculation is not a separate manual step.
func (s *Service) Approve(ctx context.Context, actor domain.Actor, claimID uuid.UUID) (domain.Claim, error) {
	return s.apply(ctx, actor, claimID, domain.ClaimApproved, "approve", "", func(r Repo, claim *domain.Claim) error {
		if err := requireReviewer(actor); err != nil {
			return err
		}
		result, err := s.pricer.Calculate(ctx, coverage.CalcInput{
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
		now := s.clock.Now()
		claim.ReviewedBy = &actor.UserID
		claim.ReviewedAt = &now
		return nil
	})
}

// Reject records the decision against the claim, with a mandatory reason.
func (s *Service) Reject(ctx context.Context, actor domain.Actor, claimID uuid.UUID, reason string) (domain.Claim, error) {
	if reason == "" {
		return domain.Claim{}, ErrReasonRequired
	}
	return s.apply(ctx, actor, claimID, domain.ClaimRejected, "reject", reason, func(r Repo, claim *domain.Claim) error {
		if err := requireReviewer(actor); err != nil {
			return err
		}
		now := s.clock.Now()
		claim.ReviewedBy = &actor.UserID
		claim.ReviewedAt = &now
		claim.RejectionReason = reason
		return nil
	})
}

// ReturnForDocs sends the claim back for missing/incomplete documents.
func (s *Service) ReturnForDocs(ctx context.Context, actor domain.Actor, claimID uuid.UUID, reason string) (domain.Claim, error) {
	if reason == "" {
		return domain.Claim{}, ErrReasonRequired
	}
	return s.apply(ctx, actor, claimID, domain.ClaimReturnedForDocs, "return_for_docs", reason, func(r Repo, claim *domain.Claim) error {
		if err := requireReviewer(actor); err != nil {
			return err
		}
		claim.RejectionReason = reason
		return nil
	})
}

// MarkPaid simulates disbursement (a real payment gateway is out of scope)
// and records a Payment row in the same transaction.
func (s *Service) MarkPaid(ctx context.Context, actor domain.Actor, claimID uuid.UUID) (domain.Claim, error) {
	return s.apply(ctx, actor, claimID, domain.ClaimPaid, "mark_paid", "", func(r Repo, claim *domain.Claim) error {
		if err := requireReviewer(actor); err != nil {
			return err
		}
		if claim.PayableAmount == nil {
			return domain.Conflictf("claim has no payable amount to disburse")
		}
		now := s.clock.Now()
		claim.PaidAt = &now
		payment := domain.Payment{
			ClaimID:          claim.ID,
			Amount:           *claim.PayableAmount,
			PaymentReference: fmt.Sprintf("SIM-%s", uuid.NewString()[:8]),
			Status:           domain.PaymentSimulated,
			PaidAt:           now,
		}
		return r.CreatePayment(ctx, &payment)
	})
}

// Close terminates the lifecycle (from rejected or paid).
func (s *Service) Close(ctx context.Context, actor domain.Actor, claimID uuid.UUID) (domain.Claim, error) {
	return s.apply(ctx, actor, claimID, domain.ClaimClosed, "close", "", func(r Repo, claim *domain.Claim) error {
		if err := requireReviewer(actor); err != nil {
			return err
		}
		now := s.clock.Now()
		claim.ClosedAt = &now
		return nil
	})
}

// apply is the shared transition runner: inside one transaction it loads the
// claim, checks the state table, lets the caller mutate the row and enforce
// its own actor rule, persists, and writes one audit entry — atomically.
func (s *Service) apply(
	ctx context.Context,
	actor domain.Actor,
	claimID uuid.UUID,
	to domain.ClaimStatus,
	action string,
	reason string,
	mutate func(r Repo, claim *domain.Claim) error,
) (domain.Claim, error) {
	var result domain.Claim
	err := s.atomic(ctx, func(r Repo) error {
		claim, err := r.GetClaim(ctx, claimID)
		if err != nil {
			return err
		}
		if !canTransition(claim.Status, to) {
			return ErrInvalidTransition
		}
		before := map[string]any{"status": claim.Status}

		if err := mutate(r, &claim); err != nil {
			return err
		}
		claim.Status = to
		claim.UpdatedAt = s.clock.Now()

		if err := r.SaveClaim(ctx, &claim); err != nil {
			return err
		}

		after := map[string]any{"status": claim.Status}
		if reason != "" {
			after["reason"] = reason
		}
		if claim.PayableAmount != nil {
			after["payable_amount"] = *claim.PayableAmount
		}
		if err := r.InsertAudit(ctx, &domain.AuditLog{
			EntityType:    "claim",
			EntityID:      claim.ID.String(),
			Action:        action,
			ActorUserID:   &actor.UserID,
			ActorUsername: actor.Username,
			BeforeData:    before,
			AfterData:     after,
			OccurredAt:    s.clock.Now(),
		}); err != nil {
			return err
		}

		result = claim
		return nil
	})
	if err != nil {
		return domain.Claim{}, err
	}
	return result, nil
}
