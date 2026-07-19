// Package ruleengine computes how much of a claim is payable under the
// currently-configured coverage_rules — the "config-driven, not hard-coded"
// engine the proposal centres the whole project on. Changing a benefit
// (percentage, cap, eligibility) means inserting a new coverage_rules row;
// this package never encodes a service type, percentage, or cap in Go code.
package ruleengine

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"insurance-module/internal/models"
)

var (
	ErrNoActiveRule      = errors.New("ruleengine: no active coverage rule for this plan/service type on the receipt date")
	ErrNotEligible       = errors.New("ruleengine: beneficiary is not eligible for this service under the current rule")
	ErrWaitingPeriod     = errors.New("ruleengine: employee has not completed the required waiting period")
	ErrEmployeeInactive  = errors.New("ruleengine: employee is not active")
	ErrDependentMismatch = errors.New("ruleengine: dependent does not belong to this employee")
)

type Engine struct {
	db *gorm.DB
}

func NewEngine(db *gorm.DB) *Engine {
	return &Engine{db: db}
}

// CalcInput describes one claim to be priced.
type CalcInput struct {
	EmployeeID      uuid.UUID
	ServiceTypeID   uuid.UUID
	PlanID          uuid.UUID
	BeneficiaryType models.BeneficiaryType
	DependentID     *uuid.UUID
	RequestedAmount float64
	ReceiptDate     time.Time
	// ExcludeClaimID lets recalculation of a claim exclude its own previous
	// contribution to the annual-cap usage sum.
	ExcludeClaimID *uuid.UUID
}

// CalcResult is the priced outcome, plus enough detail to explain it in the UI/audit log.
type CalcResult struct {
	RuleID                  uuid.UUID
	CoveragePercentApplied  float64
	RawCoveredAmount        float64
	PayableAmount           float64
	AnnualCap               *float64
	PerClaimCap             *float64
	UsedAnnualBeforeClaim   float64
	RemainingAnnualCapAfter *float64
	CappedByPerClaim        bool
	CappedByAnnualCap       bool
}

// Calculate looks up the active rule, verifies eligibility, and prices the claim.
// It performs no writes — callers (the workflow engine) persist the result.
func (e *Engine) Calculate(ctx context.Context, in CalcInput) (*CalcResult, error) {
	var employee models.Employee
	if err := e.db.WithContext(ctx).First(&employee, "id = ?", in.EmployeeID).Error; err != nil {
		return nil, err
	}
	if employee.EmploymentStatus != models.EmploymentActive {
		return nil, ErrEmployeeInactive
	}

	relation := models.RelationSelf
	if in.BeneficiaryType == models.BeneficiaryDependent {
		if in.DependentID == nil {
			return nil, ErrDependentMismatch
		}
		var dep models.Dependent
		if err := e.db.WithContext(ctx).First(&dep, "id = ?", *in.DependentID).Error; err != nil {
			return nil, err
		}
		if dep.EmployeeID != in.EmployeeID {
			return nil, ErrDependentMismatch
		}
		relation = dep.Relation
	}

	rule, err := e.activeRule(ctx, in.PlanID, in.ServiceTypeID, in.ReceiptDate)
	if err != nil {
		return nil, err
	}

	if !containsRelation(rule.EligibleRelations, relation) {
		return nil, ErrNotEligible
	}

	if rule.WaitingPeriodDays > 0 {
		eligibleFrom := employee.HireDate.AddDate(0, 0, rule.WaitingPeriodDays)
		if in.ReceiptDate.Before(eligibleFrom) {
			return nil, ErrWaitingPeriod
		}
	}

	usedAnnual, err := e.usedAnnualAmount(ctx, in.EmployeeID, in.ServiceTypeID, in.PlanID, rule, in.ExcludeClaimID)
	if err != nil {
		return nil, err
	}

	result := Compute(*rule, in.RequestedAmount, usedAnnual)
	result.RuleID = rule.ID
	return &result, nil
}

// Compute is the pure pricing function, isolated so it can be table-tested
// exhaustively without a database: given a rule version, the requested amount,
// and how much annual cap has already been used, it returns the payable amount.
func Compute(rule models.CoverageRule, requestedAmount, usedAnnualBeforeClaim float64) CalcResult {
	res := CalcResult{
		CoveragePercentApplied: rule.CoveragePercent,
		AnnualCap:              rule.AnnualCap,
		PerClaimCap:            rule.PerClaimCap,
		UsedAnnualBeforeClaim:  usedAnnualBeforeClaim,
	}

	rawCovered := requestedAmount * rule.CoveragePercent / 100.0
	res.RawCoveredAmount = rawCovered

	covered := rawCovered
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
	res.PayableAmount = round2(payable)
	return res
}

// RemainingCap summarises one service type's benefit for an employee's remaining-
// caps dashboard: the active rule's terms plus how much of the annual cap is used.
type RemainingCap struct {
	ServiceTypeCode string   `json:"service_type_code"`
	ServiceTypeName string   `json:"service_type_name"`
	CoveragePercent float64  `json:"coverage_percent"`
	PerClaimCap     *float64 `json:"per_claim_cap"`
	AnnualCap       *float64 `json:"annual_cap"`
	UsedAnnual      float64  `json:"used_annual"`
	RemainingAnnual *float64 `json:"remaining_annual"`
}

// RemainingCaps returns one RemainingCap per service type that has an active rule
// for the given plan on onDate. Service types with no rule configured for this
// plan are omitted (there is no coverage to report).
func (e *Engine) RemainingCaps(ctx context.Context, employeeID, planID uuid.UUID, onDate time.Time) ([]RemainingCap, error) {
	var serviceTypes []models.ServiceType
	if err := e.db.WithContext(ctx).Order("name").Find(&serviceTypes).Error; err != nil {
		return nil, err
	}

	var out []RemainingCap
	for _, st := range serviceTypes {
		rule, err := e.activeRule(ctx, planID, st.ID, onDate)
		if errors.Is(err, ErrNoActiveRule) {
			continue
		}
		if err != nil {
			return nil, err
		}
		used, err := e.usedAnnualAmount(ctx, employeeID, st.ID, planID, rule, nil)
		if err != nil {
			return nil, err
		}
		rc := RemainingCap{
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

func (e *Engine) activeRule(ctx context.Context, planID, serviceTypeID uuid.UUID, onDate time.Time) (*models.CoverageRule, error) {
	var rule models.CoverageRule
	// created_at DESC breaks ties when two versions share an effective_from
	// (same-day re-publish): the newest published version wins.
	err := e.db.WithContext(ctx).
		Where("plan_id = ? AND service_type_id = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to >= ?)",
			planID, serviceTypeID, onDate, onDate).
		Order("effective_from DESC, created_at DESC").
		First(&rule).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNoActiveRule
	}
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// usedAnnualAmount sums payable_amount for claims already committed against the
// same employee/plan/service-type within the rule's contract year, excluding
// claims that never reached a payable state and optionally excluding one claim
// (used when recalculating a claim that is itself part of the sum).
func (e *Engine) usedAnnualAmount(ctx context.Context, employeeID, serviceTypeID, planID uuid.UUID, rule *models.CoverageRule, excludeClaimID *uuid.UUID) (float64, error) {
	yearStart, yearEnd := contractYearWindow(rule.EffectiveFrom)

	q := e.db.WithContext(ctx).Model(&models.Claim{}).
		Where("employee_id = ? AND service_type_id = ? AND plan_id = ?", employeeID, serviceTypeID, planID).
		Where("status IN ?", []models.ClaimStatus{
			models.ClaimApproved, models.ClaimPaymentCalculated, models.ClaimPaid, models.ClaimClosed,
		}).
		Where("receipt_date >= ? AND receipt_date < ?", yearStart, yearEnd)

	if excludeClaimID != nil {
		q = q.Where("id <> ?", *excludeClaimID)
	}

	var total *float64
	if err := q.Select("COALESCE(SUM(payable_amount), 0)").Scan(&total).Error; err != nil {
		return 0, err
	}
	if total == nil {
		return 0, nil
	}
	return *total, nil
}

// contractYearWindow anchors the annual cap to a 12-month window starting on the
// rule's effective_from anniversary that contains "now" — so caps reset yearly
// relative to the policy's own start date rather than the calendar year.
func contractYearWindow(effectiveFrom time.Time) (time.Time, time.Time) {
	now := time.Now()
	start := time.Date(now.Year(), effectiveFrom.Month(), effectiveFrom.Day(), 0, 0, 0, 0, time.UTC)
	if start.After(now) {
		start = start.AddDate(-1, 0, 0)
	}
	return start, start.AddDate(1, 0, 0)
}

func containsRelation(list []string, r models.Relation) bool {
	for _, v := range list {
		if v == string(r) {
			return true
		}
	}
	return false
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
