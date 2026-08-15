package coverage

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
)

// PublishRuleInput is a new coverage-rule version to put into force.
type PublishRuleInput struct {
	PlanID            uuid.UUID
	ServiceTypeID     uuid.UUID
	CoveragePercent   domain.Percent
	PerClaimCap       *domain.Rial
	AnnualCap         *domain.Rial
	WaitingPeriodDays int
	EligibleRelations []domain.Relation
	EffectiveFrom     time.Time
}

// PublishRuleVersion is THE config-driven policy-change operation: within one
// transaction it closes the currently-open version for (plan, service type) —
// clamping the close date on a same-day re-publish so the row CHECK constraint
// holds — inserts the new version, and records a config_change audit entry.
// No code change or redeploy is ever needed to alter benefits.
func (s *Service) PublishRuleVersion(ctx context.Context, actor domain.Actor, in PublishRuleInput) (domain.CoverageRule, error) {
	if len(in.EligibleRelations) == 0 {
		return domain.CoverageRule{}, domain.Validationf("at least one eligible relation is required")
	}
	if in.CoveragePercent < 0 || in.CoveragePercent > domain.PercentFromFloat(100) {
		return domain.CoverageRule{}, domain.Validationf("coverage percent must be between 0 and 100")
	}

	var created domain.CoverageRule
	err := s.atomic(ctx, func(r Repo) error {
		var previous *domain.CoverageRule
		prev, found, err := r.OpenRule(ctx, in.PlanID, in.ServiceTypeID)
		if err != nil {
			return err
		}
		if found {
			closeDate := in.EffectiveFrom.AddDate(0, 0, -1)
			// Same-day (or backdated) re-publish: closing with "new - 1 day"
			// would violate CHECK (effective_to >= effective_from). Clamp to
			// the old rule's own start; on the boundary day the engine picks
			// the newest version (created_at tiebreak in ActiveRule).
			if closeDate.Before(prev.EffectiveFrom) {
				closeDate = prev.EffectiveFrom
			}
			if err := r.CloseRule(ctx, prev.ID, closeDate); err != nil {
				return err
			}
			previous = &prev
		}

		created = domain.CoverageRule{
			PlanID:            in.PlanID,
			ServiceTypeID:     in.ServiceTypeID,
			CoveragePercent:   in.CoveragePercent,
			PerClaimCap:       in.PerClaimCap,
			AnnualCap:         in.AnnualCap,
			WaitingPeriodDays: in.WaitingPeriodDays,
			EligibleRelations: in.EligibleRelations,
			EffectiveFrom:     in.EffectiveFrom,
			CreatedBy:         &actor.UserID,
		}
		if err := r.CreateRule(ctx, &created); err != nil {
			return err
		}

		before := map[string]any{}
		if previous != nil {
			before["previous_rule"] = *previous
		}
		return r.InsertAudit(ctx, &domain.AuditLog{
			EntityType:    "coverage_rule",
			EntityID:      created.ID.String(),
			Action:        "config_change",
			ActorUserID:   &actor.UserID,
			ActorUsername: actor.Username,
			BeforeData:    before,
			AfterData:     map[string]any{"new_rule": created},
			OccurredAt:    s.clock.Now(),
		})
	})
	if err != nil {
		return domain.CoverageRule{}, err
	}
	return created, nil
}

// ---- reference data (read/create passthroughs with no extra policy) ----

func (s *Service) ListRules(ctx context.Context, f domain.RuleFilter) ([]domain.CoverageRule, error) {
	return s.repo.ListRules(ctx, f)
}

func (s *Service) ListServiceTypes(ctx context.Context) ([]domain.ServiceType, error) {
	return s.repo.ListServiceTypes(ctx)
}

// CreateServiceType adds a claimable service category to the catalogue.
// New types appear in claim/rule dropdowns immediately; they still need a
// coverage rule before the pricing engine will accept claims for them.
func (s *Service) CreateServiceType(ctx context.Context, st domain.ServiceType) (domain.ServiceType, error) {
	code := strings.TrimSpace(st.Code)
	name := strings.TrimSpace(st.Name)
	if code == "" || name == "" {
		return domain.ServiceType{}, domain.Validationf("code and name are required")
	}
	if len(code) > 30 || len(name) > 100 {
		return domain.ServiceType{}, domain.Validationf("code or name exceeds maximum length")
	}
	if !isServiceTypeCode(code) {
		return domain.ServiceType{}, domain.Validationf("code must be lowercase letters, digits, and underscores")
	}
	if _, err := s.repo.GetServiceTypeByCode(ctx, code); err == nil {
		return domain.ServiceType{}, domain.Conflictf("service type code already exists")
	} else if domain.KindOf(err) != domain.KindNotFound {
		return domain.ServiceType{}, err
	}

	created := domain.ServiceType{Code: code, Name: name}
	if err := s.repo.CreateServiceType(ctx, &created); err != nil {
		return domain.ServiceType{}, err
	}
	return created, nil
}

func isServiceTypeCode(code string) bool {
	if code == "" || code[0] < 'a' || code[0] > 'z' {
		return false
	}
	for _, r := range code {
		if r == '_' || unicode.IsDigit(r) || (r >= 'a' && r <= 'z') {
			continue
		}
		return false
	}
	return true
}

func (s *Service) ListContracts(ctx context.Context) ([]domain.InsuranceContract, error) {
	return s.repo.ListContracts(ctx)
}

func (s *Service) CreateContract(ctx context.Context, c domain.InsuranceContract) (domain.InsuranceContract, error) {
	if err := s.repo.CreateContract(ctx, &c); err != nil {
		return domain.InsuranceContract{}, err
	}
	return c, nil
}

func (s *Service) ListPlans(ctx context.Context, contractID string) ([]domain.CoveragePlan, error) {
	return s.repo.ListPlans(ctx, contractID)
}

func (s *Service) CreatePlan(ctx context.Context, p domain.CoveragePlan) (domain.CoveragePlan, error) {
	if err := s.repo.CreatePlan(ctx, &p); err != nil {
		return domain.CoveragePlan{}, err
	}
	return p, nil
}
