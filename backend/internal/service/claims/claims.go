// Package claims owns the claim lifecycle: creation, role-scoped listing and
// access, and the multi-stage workflow (submit → review → approve/reject/
// return-for-docs → pay → close). Every transition is validated against an
// explicit state table and written to the audit trail — with actor, reason,
// and before/after status — in the same transaction as the change itself.
package claims

import (
	"context"
	"time"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
	"insurance-module/internal/service/coverage"
)

// Sentinel business errors (domain-kinded for transport mapping).
var (
	ErrInvalidTransition = domain.Conflictf("transition not allowed from the claim's current status")
	ErrForbidden         = domain.Forbiddenf("actor is not permitted to perform this action")
	ErrReasonRequired    = domain.Validationf("a reason is required for this action")
)

// transitions enumerates every legal (from → to) status change. Anything not
// listed is rejected, so the workflow can never skip a step or run backwards
// except through the explicit returned_for_docs → submitted resubmission path.
var transitions = map[domain.ClaimStatus][]domain.ClaimStatus{
	domain.ClaimDraft:           {domain.ClaimSubmitted},
	domain.ClaimSubmitted:       {domain.ClaimUnderReview},
	domain.ClaimUnderReview:     {domain.ClaimApproved, domain.ClaimRejected, domain.ClaimReturnedForDocs},
	domain.ClaimReturnedForDocs: {domain.ClaimSubmitted},
	domain.ClaimApproved:        {domain.ClaimPaid},
	domain.ClaimRejected:        {domain.ClaimClosed},
	domain.ClaimPaid:            {domain.ClaimClosed},
}

func canTransition(from, to domain.ClaimStatus) bool {
	for _, allowed := range transitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// Repo is the persistence surface this service consumes.
type Repo interface {
	GetClaim(ctx context.Context, id uuid.UUID) (domain.Claim, error)
	CreateClaim(ctx context.Context, c *domain.Claim) error
	SaveClaim(ctx context.Context, c *domain.Claim) error
	ListClaims(ctx context.Context, f domain.ClaimFilter) ([]domain.Claim, int64, error)
	CreatePayment(ctx context.Context, p *domain.Payment) error
	GetEmployee(ctx context.Context, id uuid.UUID) (domain.Employee, error)
	GetUser(ctx context.Context, id uuid.UUID) (domain.User, error)
	InsertAudit(ctx context.Context, entry *domain.AuditLog) error
	QueryAudit(ctx context.Context, f domain.AuditFilter) ([]domain.AuditLog, int64, error)
	ListAttachments(ctx context.Context, claimID uuid.UUID) ([]domain.Attachment, error)
	GetAttachment(ctx context.Context, id uuid.UUID) (domain.Attachment, error)
	CreateAttachment(ctx context.Context, a *domain.Attachment) error
}

// Atomic runs fn inside one storage transaction.
type Atomic func(ctx context.Context, fn func(Repo) error) error

// Pricer prices a claim against the active coverage rule (the coverage service).
type Pricer interface {
	Calculate(ctx context.Context, in coverage.CalcInput) (*coverage.CalcResult, error)
}

type Service struct {
	repo   Repo
	atomic Atomic
	pricer Pricer
	clock  domain.Clock
	files  FileStore // nil disables attachments (see attachments.go)
}

func NewService(repo Repo, atomic Atomic, pricer Pricer, clock domain.Clock, files FileStore) *Service {
	if clock == nil {
		clock = domain.SystemClock{}
	}
	return &Service{repo: repo, atomic: atomic, pricer: pricer, clock: clock, files: files}
}

// CreateInput describes a new draft claim.
type CreateInput struct {
	EmployeeID      *uuid.UUID // admins may set it; employees are forced to their own
	BeneficiaryType domain.BeneficiaryType
	DependentID     *uuid.UUID
	ServiceTypeID   uuid.UUID
	RequestedAmount domain.Rial
	ReceiptDate     time.Time
	Description     string
}

// Create makes a draft claim. Employees always file for themselves (their
// linked employee record); admins may file for any employee.
func (s *Service) Create(ctx context.Context, actor domain.Actor, in CreateInput) (domain.Claim, error) {
	if actor.Role != domain.RoleEmployee && actor.Role != domain.RoleAdmin {
		return domain.Claim{}, domain.Forbiddenf("only employees or admins may submit claims")
	}

	employeeID, err := s.resolveEmployeeID(ctx, actor, in.EmployeeID)
	if err != nil {
		return domain.Claim{}, err
	}

	employee, err := s.repo.GetEmployee(ctx, employeeID)
	if err != nil {
		return domain.Claim{}, err
	}
	if employee.PlanID == nil {
		return domain.Claim{}, domain.Unprocessablef("employee has no coverage plan assigned")
	}
	if in.RequestedAmount <= 0 {
		return domain.Claim{}, domain.Validationf("requested amount must be positive")
	}
	if in.BeneficiaryType == domain.BeneficiaryDependent && in.DependentID == nil {
		return domain.Claim{}, domain.Validationf("dependent_id is required for a dependent claim")
	}

	claim := domain.Claim{
		EmployeeID:      employeeID,
		BeneficiaryType: in.BeneficiaryType,
		DependentID:     in.DependentID,
		ServiceTypeID:   in.ServiceTypeID,
		PlanID:          *employee.PlanID,
		RequestedAmount: in.RequestedAmount,
		ReceiptDate:     in.ReceiptDate,
		Description:     in.Description,
		Status:          domain.ClaimDraft,
		CreatedBy:       actor.UserID,
	}
	if err := s.repo.CreateClaim(ctx, &claim); err != nil {
		return domain.Claim{}, err
	}
	return claim, nil
}

func (s *Service) resolveEmployeeID(ctx context.Context, actor domain.Actor, requested *uuid.UUID) (uuid.UUID, error) {
	if actor.Role == domain.RoleAdmin {
		if requested == nil {
			return uuid.UUID{}, domain.Validationf("employee_id is required")
		}
		return *requested, nil
	}
	user, err := s.repo.GetUser(ctx, actor.UserID)
	if err != nil {
		return uuid.UUID{}, domain.Validationf("could not resolve caller's employee record")
	}
	if user.EmployeeID == nil {
		return uuid.UUID{}, domain.Validationf("this account is not linked to an employee record")
	}
	return *user.EmployeeID, nil
}

// List returns claims visible to the actor: employees see only their own,
// staff and auditors see all (optionally filtered).
func (s *Service) List(ctx context.Context, actor domain.Actor, f domain.ClaimFilter) ([]domain.Claim, int64, error) {
	if actor.Role == domain.RoleEmployee {
		f.CreatedBy = &actor.UserID
		f.EmployeeID = ""
	}
	return s.repo.ListClaims(ctx, f)
}

// Get enforces the read policy: owner, reviewer, admin, or auditor.
func (s *Service) Get(ctx context.Context, actor domain.Actor, id uuid.UUID) (domain.Claim, error) {
	claim, err := s.repo.GetClaim(ctx, id)
	if err != nil {
		return domain.Claim{}, err
	}
	if !s.canRead(actor, claim) {
		return domain.Claim{}, domain.Forbiddenf("not permitted to access this claim")
	}
	return claim, nil
}

// History returns the audit trail of one claim, newest first.
func (s *Service) History(ctx context.Context, actor domain.Actor, id uuid.UUID) ([]domain.AuditLog, error) {
	if _, err := s.Get(ctx, actor, id); err != nil {
		return nil, err
	}
	entries, _, err := s.repo.QueryAudit(ctx, domain.AuditFilter{
		EntityType: "claim",
		EntityID:   id.String(),
		Page:       domain.NewPage(1, 200),
	})
	return entries, err
}

func (s *Service) canRead(actor domain.Actor, claim domain.Claim) bool {
	switch actor.Role {
	case domain.RoleAdmin, domain.RoleReviewer, domain.RoleAuditor:
		return true
	case domain.RoleEmployee:
		return claim.CreatedBy == actor.UserID
	default:
		return false
	}
}

func requireReviewer(actor domain.Actor) error {
	if !actor.IsStaff() {
		return ErrForbidden
	}
	return nil
}
