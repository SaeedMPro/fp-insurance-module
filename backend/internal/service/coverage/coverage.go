// Package coverage is the policy side of the system: the config-driven pricing
// engine (percentage / per-claim cap / annual cap / waiting period /
// eligibility, all read from versioned coverage rules) and the administration
// of those rules and their reference data (contracts, plans, service types).
//
// Changing a benefit means publishing a new rule version through this service —
// never a code change. The pricing math itself lives in Compute, a pure
// function, so it is exhaustively table-testable without a database.
package coverage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
)

// Sentinel business errors. They carry domain kinds so the transport layer
// maps them without knowing this package.
var (
	ErrNoActiveRule      = domain.Unprocessablef("no active coverage rule for this plan/service type on the receipt date")
	ErrNotEligible       = domain.Unprocessablef("beneficiary is not eligible for this service under the current rule")
	ErrWaitingPeriod     = domain.Unprocessablef("employee has not completed the required waiting period")
	ErrEmployeeInactive  = domain.Unprocessablef("employee is not active")
	ErrDependentMismatch = domain.Unprocessablef("dependent does not belong to this employee")
)

// Repo is the persistence surface this service consumes (implemented by
// storage/postgres.Store).
type Repo interface {
	GetEmployee(ctx context.Context, id uuid.UUID) (domain.Employee, error)
	GetDependent(ctx context.Context, id uuid.UUID) (domain.Dependent, error)
	ActiveRule(ctx context.Context, planID, serviceTypeID uuid.UUID, onDate time.Time) (domain.CoverageRule, error)
	OpenRule(ctx context.Context, planID, serviceTypeID uuid.UUID) (domain.CoverageRule, bool, error)
	CloseRule(ctx context.Context, ruleID uuid.UUID, effectiveTo time.Time) error
	CreateRule(ctx context.Context, r *domain.CoverageRule) error
	ListRules(ctx context.Context, f domain.RuleFilter) ([]domain.CoverageRule, error)
	SumPayable(ctx context.Context, employeeID, serviceTypeID, planID uuid.UUID,
		statuses []domain.ClaimStatus, from, to time.Time, excludeClaimID *uuid.UUID) (domain.Rial, error)
	ListServiceTypes(ctx context.Context) ([]domain.ServiceType, error)
	GetServiceTypeByCode(ctx context.Context, code string) (domain.ServiceType, error)
	CreateServiceType(ctx context.Context, st *domain.ServiceType) error
	ListContracts(ctx context.Context) ([]domain.InsuranceContract, error)
	CreateContract(ctx context.Context, c *domain.InsuranceContract) error
	ListPlans(ctx context.Context, contractID string) ([]domain.CoveragePlan, error)
	CreatePlan(ctx context.Context, p *domain.CoveragePlan) error
	InsertAudit(ctx context.Context, entry *domain.AuditLog) error
}

// Atomic runs fn inside one storage transaction, handing it a Repo bound to
// that transaction (wired from storage/postgres.Store.Atomic).
type Atomic func(ctx context.Context, fn func(Repo) error) error

type Service struct {
	repo   Repo
	atomic Atomic
	clock  domain.Clock
}

func NewService(repo Repo, atomic Atomic, clock domain.Clock) *Service {
	if clock == nil {
		clock = domain.SystemClock{}
	}
	return &Service{repo: repo, atomic: atomic, clock: clock}
}

// committedStatuses are the claim states whose payable amounts count against
// the annual cap.
var committedStatuses = []domain.ClaimStatus{
	domain.ClaimApproved, domain.ClaimPaymentCalculated, domain.ClaimPaid, domain.ClaimClosed,
}

// CalcInput describes one claim to be priced.
type CalcInput struct {
	EmployeeID      uuid.UUID
	ServiceTypeID   uuid.UUID
	PlanID          uuid.UUID
	BeneficiaryType domain.BeneficiaryType
	DependentID     *uuid.UUID
	RequestedAmount domain.Rial
	ReceiptDate     time.Time
	// ExcludeClaimID lets re-pricing a claim exclude its own previous
	// contribution to the annual-cap usage sum.
	ExcludeClaimID *uuid.UUID
}

// CalcResult is the priced outcome, with enough detail to explain the decision.
type CalcResult struct {
	RuleID                  uuid.UUID
	CoveragePercentApplied  domain.Percent
	RawCoveredAmount        domain.Rial
	PayableAmount           domain.Rial
	AnnualCap               *domain.Rial
	PerClaimCap             *domain.Rial
	UsedAnnualBeforeClaim   domain.Rial
	RemainingAnnualCapAfter *domain.Rial
	CappedByPerClaim        bool
	CappedByAnnualCap       bool
}

// Calculate looks up the active rule, verifies eligibility, and prices the
// claim. It performs no writes — callers persist the result.
func (s *Service) Calculate(ctx context.Context, in CalcInput) (*CalcResult, error) {
	employee, err := s.repo.GetEmployee(ctx, in.EmployeeID)
	if err != nil {
		return nil, err
	}
	if employee.EmploymentStatus != domain.EmploymentActive {
		return nil, ErrEmployeeInactive
	}

	relation := domain.RelationSelf
	if in.BeneficiaryType == domain.BeneficiaryDependent {
		if in.DependentID == nil {
			return nil, ErrDependentMismatch
		}
		dep, err := s.repo.GetDependent(ctx, *in.DependentID)
		if err != nil {
			return nil, err
		}
		if dep.EmployeeID != in.EmployeeID {
			return nil, ErrDependentMismatch
		}
		relation = dep.Relation
	}

	rule, err := s.activeRule(ctx, in.PlanID, in.ServiceTypeID, in.ReceiptDate)
	if err != nil {
		return nil, err
	}

	if !rule.EligibleFor(relation) {
		return nil, ErrNotEligible
	}

	if rule.WaitingPeriodDays > 0 {
		// Compare civil days in the business timezone: a receipt dated the day
		// eligibility starts must qualify regardless of host zone.
		eligibleFrom := domain.BusinessDay(employee.HireDate).AddDate(0, 0, rule.WaitingPeriodDays)
		if domain.BusinessDay(in.ReceiptDate).Before(eligibleFrom) {
			return nil, ErrWaitingPeriod
		}
	}

	usedAnnual, err := s.usedAnnualAmount(ctx, in.EmployeeID, in.ServiceTypeID, in.PlanID, rule, in.ExcludeClaimID)
	if err != nil {
		return nil, err
	}

	result := Compute(rule, in.RequestedAmount, usedAnnual)
	result.RuleID = rule.ID
	return &result, nil
}

// Compute is the pure pricing function: given a rule version, the requested
// amount, and how much annual cap is already used, it returns the payable
// amount and which cap (if any) bound it.
//
// All arithmetic is exact integer rial; the single rounding
// decision — half-up to the whole rial — happens inside Percent.ApplyTo, so
// caps are compared against an already-whole amount and no fractional value
// can escape into a payment or an annual-cap total.
func Compute(rule domain.CoverageRule, requestedAmount, usedAnnualBeforeClaim domain.Rial) CalcResult {
	res := CalcResult{
		CoveragePercentApplied: rule.CoveragePercent,
		AnnualCap:              rule.AnnualCap,
		PerClaimCap:            rule.PerClaimCap,
		UsedAnnualBeforeClaim:  usedAnnualBeforeClaim,
	}

	covered := rule.CoveragePercent.ApplyTo(requestedAmount)
	res.RawCoveredAmount = covered

	if rule.PerClaimCap != nil && covered > *rule.PerClaimCap {
		covered = *rule.PerClaimCap
		res.CappedByPerClaim = true
	}

	payable := covered
	if rule.AnnualCap != nil {
		remainingBefore := *rule.AnnualCap - usedAnnualBeforeClaim
		if remainingBefore < 0 {
			remainingBefore = 0
		}
		if payable > remainingBefore {
			payable = remainingBefore
			res.CappedByAnnualCap = true
		}
		remainingAfter := remainingBefore - payable
		res.RemainingAnnualCapAfter = &remainingAfter
	}
	if payable < 0 {
		payable = 0
	}
	res.PayableAmount = payable
	return res
}

// RemainingCaps returns one entry per service type that has an active rule for
// the plan on onDate; service types with no rule are omitted.
func (s *Service) RemainingCaps(ctx context.Context, employeeID, planID uuid.UUID, onDate time.Time) ([]domain.RemainingCap, error) {
	serviceTypes, err := s.repo.ListServiceTypes(ctx)
	if err != nil {
		return nil, err
	}

	var out []domain.RemainingCap
	for _, st := range serviceTypes {
		rule, err := s.activeRule(ctx, planID, st.ID, onDate)
		if errors.Is(err, ErrNoActiveRule) {
			continue
		}
		if err != nil {
			return nil, err
		}
		used, err := s.usedAnnualAmount(ctx, employeeID, st.ID, planID, rule, nil)
		if err != nil {
			return nil, err
		}
		rc := domain.RemainingCap{
			ServiceTypeCode: st.Code,
			ServiceTypeName: st.Name,
			CoveragePercent: rule.CoveragePercent,
			PerClaimCap:     rule.PerClaimCap,
			AnnualCap:       rule.AnnualCap,
			UsedAnnual:      used,
		}
		if rule.AnnualCap != nil {
			remaining := *rule.AnnualCap - used
			if remaining < 0 {
				remaining = 0
			}
			rc.RemainingAnnual = &remaining
		}
		out = append(out, rc)
	}
	return out, nil
}

func (s *Service) activeRule(ctx context.Context, planID, serviceTypeID uuid.UUID, onDate time.Time) (domain.CoverageRule, error) {
	rule, err := s.repo.ActiveRule(ctx, planID, serviceTypeID, onDate)
	if err != nil {
		// Storage classifies a missing row as NotFound; for pricing this is a
		// business refusal (422), and the sentinel keeps the old contract.
		if domain.KindOf(err) == domain.KindNotFound {
			return domain.CoverageRule{}, ErrNoActiveRule
		}
		return domain.CoverageRule{}, err
	}
	return rule, nil
}

// usedAnnualAmount sums committed payables inside the rule's contract-year
// window (a 12-month window anchored to the rule's effective_from anniversary
// containing "now" — caps reset relative to the policy's own start date).
func (s *Service) usedAnnualAmount(ctx context.Context, employeeID, serviceTypeID, planID uuid.UUID, rule domain.CoverageRule, excludeClaimID *uuid.UUID) (domain.Rial, error) {
	yearStart, yearEnd := contractYearWindow(rule.EffectiveFrom, s.clock.Now())
	return s.repo.SumPayable(ctx, employeeID, serviceTypeID, planID, committedStatuses, yearStart, yearEnd, excludeClaimID)
}

// contractYearWindow is the 12-month window anchored on the rule's
// effective_from anniversary that contains now, evaluated in the business
// timezone so the window does not shift with the host's zone.
func contractYearWindow(effectiveFrom, now time.Time) (time.Time, time.Time) {
	loc := domain.BusinessLocation()
	anchor := effectiveFrom.In(loc)
	localNow := now.In(loc)
	start := time.Date(localNow.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, loc)
	if start.After(localNow) {
		start = start.AddDate(-1, 0, 0)
	}
	return start, start.AddDate(1, 0, 0)
}
