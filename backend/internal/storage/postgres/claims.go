package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
)

func (s *Store) GetClaim(ctx context.Context, id uuid.UUID) (domain.Claim, error) {
	var row claimRow
	if err := s.ctx(ctx).First(&row, "id = ?", id).Error; err != nil {
		return domain.Claim{}, notFound(err, "claim")
	}
	return row.toDomain(), nil
}

func (s *Store) CreateClaim(ctx context.Context, c *domain.Claim) error {
	row := claimFromDomain(*c)
	if err := s.ctx(ctx).Create(&row).Error; err != nil {
		return err
	}
	*c = row.toDomain()
	return nil
}

func (s *Store) SaveClaim(ctx context.Context, c *domain.Claim) error {
	row := claimFromDomain(*c)
	if err := s.ctx(ctx).Save(&row).Error; err != nil {
		return err
	}
	*c = row.toDomain()
	return nil
}

func (s *Store) ListClaims(ctx context.Context, f domain.ClaimFilter) ([]domain.Claim, int64, error) {
	q := s.ctx(ctx).Model(&claimRow{})
	if f.CreatedBy != nil {
		q = q.Where("created_by = ?", *f.CreatedBy)
	}
	if f.EmployeeID != "" {
		q = q.Where("employee_id = ?", f.EmployeeID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", string(f.Status))
	}
	if f.ServiceTypeID != "" {
		q = q.Where("service_type_id = ?", f.ServiceTypeID)
	}
	if f.From != nil {
		q = q.Where("receipt_date >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("receipt_date <= ?", *f.To)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []claimRow
	err := q.Order("created_at DESC").Offset(f.Page.Offset()).Limit(f.Page.Size).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.Claim, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toDomain())
	}
	return out, total, nil
}

// SumPayable totals payable_amount over committed claims for one employee /
// service type / plan inside [from, to), optionally excluding one claim (used
// when re-pricing a claim already counted in the sum).
func (s *Store) SumPayable(
	ctx context.Context,
	employeeID, serviceTypeID, planID uuid.UUID,
	statuses []domain.ClaimStatus,
	from, to time.Time,
	excludeClaimID *uuid.UUID,
) (float64, error) {
	strStatuses := make([]string, 0, len(statuses))
	for _, st := range statuses {
		strStatuses = append(strStatuses, string(st))
	}
	q := s.ctx(ctx).Model(&claimRow{}).
		Where("employee_id = ? AND service_type_id = ? AND plan_id = ?", employeeID, serviceTypeID, planID).
		Where("status IN ?", strStatuses).
		Where("receipt_date >= ? AND receipt_date < ?", from, to)
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

func (s *Store) CreatePayment(ctx context.Context, p *domain.Payment) error {
	row := paymentFromDomain(*p)
	return s.ctx(ctx).Create(&row).Error
}
