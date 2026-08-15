package postgres

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"insurance-module/internal/domain"
)

// paidStatuses are the claim states representing a real, committed spend.
var paidStatuses = []string{
	string(domain.ClaimApproved),
	string(domain.ClaimPaymentCalculated),
	string(domain.ClaimPaid),
	string(domain.ClaimClosed),
}

func applyRange(q *gorm.DB, r domain.ReportRange) *gorm.DB {
	if r.From != nil {
		q = q.Where("claims.receipt_date >= ?", *r.From)
	}
	if r.To != nil {
		q = q.Where("claims.receipt_date <= ?", *r.To)
	}
	return q
}

func (s *Store) ReportSummary(ctx context.Context, r domain.ReportRange) (domain.ReportSummary, error) {
	var out domain.ReportSummary

	if err := applyRange(s.ctx(ctx).Model(&claimRow{}), r).Count(&out.TotalClaims).Error; err != nil {
		return out, err
	}

	var totalPaid *float64
	if err := applyRange(s.ctx(ctx).Model(&claimRow{}), r).
		Where("status IN ?", paidStatuses).
		Select("COALESCE(SUM(payable_amount), 0)").Scan(&totalPaid).Error; err != nil {
		return out, err
	}
	if totalPaid != nil {
		out.TotalPaidAmount = domain.RialFromFloat(*totalPaid)
	}

	if err := applyRange(s.ctx(ctx).Model(&claimRow{}), r).
		Where("status IN ?", []string{string(domain.ClaimSubmitted), string(domain.ClaimUnderReview)}).
		Count(&out.PendingReview).Error; err != nil {
		return out, err
	}
	if err := applyRange(s.ctx(ctx).Model(&claimRow{}), r).
		Where("status = ?", string(domain.ClaimApproved)).
		Count(&out.ApprovedAwaitingPayment).Error; err != nil {
		return out, err
	}
	if err := applyRange(s.ctx(ctx).Model(&claimRow{}), r).
		Where("status = ?", string(domain.ClaimRejected)).
		Count(&out.Rejected).Error; err != nil {
		return out, err
	}
	return out, nil
}

func (s *Store) SpendByEmployee(ctx context.Context, r domain.ReportRange) ([]domain.EmployeeSpend, error) {
	// SUM() over NUMERIC scans into float64; convert to Rial per row below.
	var rows []struct {
		EmployeeID   uuid.UUID
		EmployeeName string
		PersonnelNo  string
		TotalPaid    float64
		ClaimCount   int64
	}
	q := applyRange(s.ctx(ctx).Table("claims"), r).
		Joins("JOIN employees ON employees.id = claims.employee_id").
		Where("claims.status IN ?", paidStatuses).
		Select(`employees.id AS employee_id, employees.full_name AS employee_name,
			employees.personnel_no AS personnel_no,
			COALESCE(SUM(claims.payable_amount), 0) AS total_paid,
			COUNT(claims.id) AS claim_count`).
		Group("employees.id, employees.full_name, employees.personnel_no").
		Order("total_paid DESC")
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.EmployeeSpend, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.EmployeeSpend{
			EmployeeID: row.EmployeeID, EmployeeName: row.EmployeeName,
			PersonnelNo: row.PersonnelNo, TotalPaid: domain.RialFromFloat(row.TotalPaid),
			ClaimCount: row.ClaimCount,
		})
	}
	return out, nil
}

func (s *Store) SpendByServiceType(ctx context.Context, r domain.ReportRange) ([]domain.ServiceTypeSpend, error) {
	var rows []struct {
		ServiceTypeCode string
		ServiceTypeName string
		TotalPaid       float64
		ClaimCount      int64
	}
	q := applyRange(s.ctx(ctx).Table("claims"), r).
		Joins("JOIN service_types ON service_types.id = claims.service_type_id").
		Where("claims.status IN ?", paidStatuses).
		Select(`service_types.code AS service_type_code,
			service_types.name AS service_type_name,
			COALESCE(SUM(claims.payable_amount), 0) AS total_paid,
			COUNT(claims.id) AS claim_count`).
		Group("service_types.code, service_types.name").
		Order("total_paid DESC")
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ServiceTypeSpend, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ServiceTypeSpend{
			ServiceTypeCode: row.ServiceTypeCode, ServiceTypeName: row.ServiceTypeName,
			TotalPaid: domain.RialFromFloat(row.TotalPaid), ClaimCount: row.ClaimCount,
		})
	}
	return out, nil
}

func (s *Store) SpendByMonth(ctx context.Context, r domain.ReportRange) ([]domain.MonthSpend, error) {
	var rows []struct {
		Month      string
		TotalPaid  float64
		ClaimCount int64
	}
	q := applyRange(s.ctx(ctx).Table("claims"), r).
		Where("claims.status IN ?", paidStatuses).
		Select(`TO_CHAR(claims.receipt_date, 'YYYY-MM') AS month,
			COALESCE(SUM(claims.payable_amount), 0) AS total_paid,
			COUNT(claims.id) AS claim_count`).
		Group("TO_CHAR(claims.receipt_date, 'YYYY-MM')").
		Order("month")
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.MonthSpend, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.MonthSpend{
			Month: row.Month, TotalPaid: domain.RialFromFloat(row.TotalPaid), ClaimCount: row.ClaimCount,
		})
	}
	return out, nil
}
