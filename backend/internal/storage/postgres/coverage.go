package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"insurance-module/internal/domain"
)

// ActiveRule returns the rule version in force for (plan, service type) on
// onDate. created_at DESC breaks ties when two versions share an
// effective_from (same-day re-publish): the newest published version wins.
func (s *Store) ActiveRule(ctx context.Context, planID, serviceTypeID uuid.UUID, onDate time.Time) (domain.CoverageRule, error) {
	var row ruleRow
	err := s.ctx(ctx).
		Where("plan_id = ? AND service_type_id = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to >= ?)",
			planID, serviceTypeID, onDate, onDate).
		Order("effective_from DESC, created_at DESC").
		First(&row).Error
	if err != nil {
		return domain.CoverageRule{}, notFound(err, "active coverage rule")
	}
	return row.toDomain(), nil
}

// OpenRule returns the currently-open version (effective_to IS NULL) for the
// pair, if any — used by rule publishing to close the predecessor.
func (s *Store) OpenRule(ctx context.Context, planID, serviceTypeID uuid.UUID) (domain.CoverageRule, bool, error) {
	var row ruleRow
	err := s.ctx(ctx).
		Where("plan_id = ? AND service_type_id = ? AND effective_to IS NULL", planID, serviceTypeID).
		Order("created_at DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.CoverageRule{}, false, nil
	}
	if err != nil {
		return domain.CoverageRule{}, false, err
	}
	return row.toDomain(), true, nil
}

func (s *Store) CloseRule(ctx context.Context, ruleID uuid.UUID, effectiveTo time.Time) error {
	return s.ctx(ctx).Model(&ruleRow{}).Where("id = ?", ruleID).
		Update("effective_to", effectiveTo).Error
}

func (s *Store) CreateRule(ctx context.Context, r *domain.CoverageRule) error {
	row := ruleFromDomain(*r)
	if err := s.ctx(ctx).Create(&row).Error; err != nil {
		return err
	}
	*r = row.toDomain()
	return nil
}

func (s *Store) ListRules(ctx context.Context, f domain.RuleFilter) ([]domain.CoverageRule, error) {
	q := s.ctx(ctx).Order("effective_from DESC")
	if f.PlanID != "" {
		q = q.Where("plan_id = ?", f.PlanID)
	}
	if f.ServiceTypeID != "" {
		q = q.Where("service_type_id = ?", f.ServiceTypeID)
	}
	var rows []ruleRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CoverageRule, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toDomain())
	}
	return out, nil
}

func (s *Store) ListServiceTypes(ctx context.Context) ([]domain.ServiceType, error) {
	var rows []serviceTypeRow
	if err := s.ctx(ctx).Order("name").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ServiceType, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toDomain())
	}
	return out, nil
}

func (s *Store) ListContracts(ctx context.Context) ([]domain.InsuranceContract, error) {
	var rows []contractRow
	if err := s.ctx(ctx).Order("start_date DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.InsuranceContract, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toDomain())
	}
	return out, nil
}

func (s *Store) CreateContract(ctx context.Context, c *domain.InsuranceContract) error {
	row := contractFromDomain(*c)
	if err := s.ctx(ctx).Create(&row).Error; err != nil {
		return err
	}
	*c = row.toDomain()
	return nil
}

func (s *Store) ListPlans(ctx context.Context, contractID string) ([]domain.CoveragePlan, error) {
	q := s.ctx(ctx).Order("name")
	if contractID != "" {
		q = q.Where("contract_id = ?", contractID)
	}
	var rows []planRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CoveragePlan, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toDomain())
	}
	return out, nil
}

func (s *Store) CreatePlan(ctx context.Context, p *domain.CoveragePlan) error {
	row := planFromDomain(*p)
	if err := s.ctx(ctx).Create(&row).Error; err != nil {
		return err
	}
	*p = row.toDomain()
	return nil
}

// GetPlanByName supports seeding/fixtures.
func (s *Store) GetPlanByName(ctx context.Context, name string) (domain.CoveragePlan, error) {
	var row planRow
	if err := s.ctx(ctx).Where("name = ?", name).First(&row).Error; err != nil {
		return domain.CoveragePlan{}, notFound(err, "plan")
	}
	return row.toDomain(), nil
}

// GetServiceTypeByCode supports seeding/fixtures.
func (s *Store) GetServiceTypeByCode(ctx context.Context, code string) (domain.ServiceType, error) {
	var row serviceTypeRow
	if err := s.ctx(ctx).Where("code = ?", code).First(&row).Error; err != nil {
		return domain.ServiceType{}, notFound(err, "service type")
	}
	return row.toDomain(), nil
}
