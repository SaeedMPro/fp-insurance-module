// Package audit exposes the audit trail for querying (the admin/auditor
// screens). Writing entries happens inside the owning services' transactions,
// via their repositories — this service is the read side.
package audit

import (
	"context"

	"insurance-module/internal/domain"
)

type Repo interface {
	QueryAudit(ctx context.Context, f domain.AuditFilter) ([]domain.AuditLog, int64, error)
}

type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service { return &Service{repo: repo} }

func (s *Service) Query(ctx context.Context, f domain.AuditFilter) ([]domain.AuditLog, int64, error) {
	return s.repo.QueryAudit(ctx, f)
}
