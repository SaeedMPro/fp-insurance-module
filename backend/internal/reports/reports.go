// Package reports provides the management-facing aggregation queries: spend per
// employee, per service type, per month, and a dashboard summary — the
// "gzarsh-giri" (reporting) requirement from the proposal.
package reports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"insurance-module/internal/models"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Filter narrows every report to a receipt_date window. Zero values mean unbounded.
type Filter struct {
	From *time.Time
	To   *time.Time
}

func (f Filter) apply(q *gorm.DB) *gorm.DB {
	if f.From != nil {
		q = q.Where("claims.receipt_date >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("claims.receipt_date <= ?", *f.To)
	}
	return q
}

// paidStatuses are the claim states that represent a real, committed spend —
// draft/submitted/under_review/rejected/returned carry no payable_amount worth summing.
var paidStatuses = []models.ClaimStatus{
	models.ClaimApproved, models.ClaimPaymentCalculated, models.ClaimPaid, models.ClaimClosed,
}

type Summary struct {
	TotalClaims             int64   `json:"total_claims"`
	TotalPaidAmount         float64 `json:"total_paid_amount"`
	PendingReview           int64   `json:"pending_review"`
	ApprovedAwaitingPayment int64   `json:"approved_awaiting_payment"`
	Rejected                int64   `json:"rejected"`
}

func (s *Service) Summary(ctx context.Context, f Filter) (*Summary, error) {
	out := &Summary{}

	base := f.apply(s.db.WithContext(ctx).Model(&models.Claim{}))
	if err := base.Session(&gorm.Session{}).Count(&out.TotalClaims).Error; err != nil {
		return nil, err
	}

	var totalPaid *float64
	if err := f.apply(s.db.WithContext(ctx).Model(&models.Claim{})).
		Where("status IN ?", paidStatuses).
		Select("COALESCE(SUM(payable_amount), 0)").Scan(&totalPaid).Error; err != nil {
		return nil, err
	}
	if totalPaid != nil {
		out.TotalPaidAmount = *totalPaid
	}

	if err := f.apply(s.db.WithContext(ctx).Model(&models.Claim{})).
		Where("status IN ?", []models.ClaimStatus{models.ClaimSubmitted, models.ClaimUnderReview}).
		Count(&out.PendingReview).Error; err != nil {
		return nil, err
	}

	if err := f.apply(s.db.WithContext(ctx).Model(&models.Claim{})).
		Where("status = ?", models.ClaimApproved).
		Count(&out.ApprovedAwaitingPayment).Error; err != nil {
		return nil, err
	}

	if err := f.apply(s.db.WithContext(ctx).Model(&models.Claim{})).
		Where("status = ?", models.ClaimRejected).
		Count(&out.Rejected).Error; err != nil {
		return nil, err
	}

	return out, nil
}

type EmployeeSpend struct {
	EmployeeID   uuid.UUID `json:"employee_id"`
	EmployeeName string    `json:"employee_name"`
	PersonnelNo  string    `json:"personnel_no"`
	TotalPaid    float64   `json:"total_paid"`
	ClaimCount   int64     `json:"claim_count"`
}

func (s *Service) SpendByEmployee(ctx context.Context, f Filter) ([]EmployeeSpend, error) {
	var out []EmployeeSpend
	q := f.apply(s.db.WithContext(ctx).Table("claims").
		Joins("JOIN employees ON employees.id = claims.employee_id").
		Where("claims.status IN ?", paidStatuses).
		Select(`employees.id AS employee_id, employees.full_name AS employee_name,
			employees.personnel_no AS personnel_no,
			COALESCE(SUM(claims.payable_amount), 0) AS total_paid,
			COUNT(claims.id) AS claim_count`).
		Group("employees.id, employees.full_name, employees.personnel_no").
		Order("total_paid DESC"))
	err := q.Scan(&out).Error
	return out, err
}

type ServiceTypeSpend struct {
	ServiceTypeCode string  `json:"service_type_code"`
	ServiceTypeName string  `json:"service_type_name"`
	TotalPaid       float64 `json:"total_paid"`
	ClaimCount      int64   `json:"claim_count"`
}

func (s *Service) SpendByServiceType(ctx context.Context, f Filter) ([]ServiceTypeSpend, error) {
	var out []ServiceTypeSpend
	q := f.apply(s.db.WithContext(ctx).Table("claims").
		Joins("JOIN service_types ON service_types.id = claims.service_type_id").
		Where("claims.status IN ?", paidStatuses).
		Select(`service_types.code AS service_type_code,
			COALESCE(NULLIF(service_types.name_fa, ''), service_types.name) AS service_type_name,
			COALESCE(SUM(claims.payable_amount), 0) AS total_paid,
			COUNT(claims.id) AS claim_count`).
		Group("service_types.code, service_types.name, service_types.name_fa").
		Order("total_paid DESC"))
	err := q.Scan(&out).Error
	return out, err
}

type MonthSpend struct {
	Month      string  `json:"month"`
	TotalPaid  float64 `json:"total_paid"`
	ClaimCount int64   `json:"claim_count"`
}

func (s *Service) SpendByMonth(ctx context.Context, f Filter) ([]MonthSpend, error) {
	var out []MonthSpend
	q := f.apply(s.db.WithContext(ctx).Table("claims").
		Where("claims.status IN ?", paidStatuses).
		Select(`TO_CHAR(claims.receipt_date, 'YYYY-MM') AS month,
			COALESCE(SUM(claims.payable_amount), 0) AS total_paid,
			COUNT(claims.id) AS claim_count`).
		Group("TO_CHAR(claims.receipt_date, 'YYYY-MM')").
		Order("month"))
	err := q.Scan(&out).Error
	return out, err
}
