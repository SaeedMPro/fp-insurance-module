// Package reports exposes the management-facing aggregations: dashboard
// summary and spend per employee / service type / month.
package reports

import (
	"context"

	"insurance-module/internal/domain"
)

type Repo interface {
	ReportSummary(ctx context.Context, r domain.ReportRange) (domain.ReportSummary, error)
	SpendByEmployee(ctx context.Context, r domain.ReportRange) ([]domain.EmployeeSpend, error)
	SpendByServiceType(ctx context.Context, r domain.ReportRange) ([]domain.ServiceTypeSpend, error)
	SpendByMonth(ctx context.Context, r domain.ReportRange) ([]domain.MonthSpend, error)
}

type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service { return &Service{repo: repo} }

func (s *Service) Summary(ctx context.Context, r domain.ReportRange) (domain.ReportSummary, error) {
	return s.repo.ReportSummary(ctx, r)
}

func (s *Service) SpendByEmployee(ctx context.Context, r domain.ReportRange) ([]domain.EmployeeSpend, error) {
	return s.repo.SpendByEmployee(ctx, r)
}

func (s *Service) SpendByServiceType(ctx context.Context, r domain.ReportRange) ([]domain.ServiceTypeSpend, error) {
	return s.repo.SpendByServiceType(ctx, r)
}

func (s *Service) SpendByMonth(ctx context.Context, r domain.ReportRange) ([]domain.MonthSpend, error) {
	return s.repo.SpendByMonth(ctx, r)
}
