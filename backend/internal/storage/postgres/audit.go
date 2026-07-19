package postgres

import (
	"context"

	"insurance-module/internal/domain"
)

func (s *Store) InsertAudit(ctx context.Context, entry *domain.AuditLog) error {
	row := auditFromDomain(*entry)
	return s.ctx(ctx).Create(&row).Error
}

func (s *Store) QueryAudit(ctx context.Context, f domain.AuditFilter) ([]domain.AuditLog, int64, error) {
	q := s.ctx(ctx).Model(&auditRow{})
	if f.EntityType != "" {
		q = q.Where("entity_type = ?", f.EntityType)
	}
	if f.EntityID != "" {
		q = q.Where("entity_id = ?", f.EntityID)
	}
	if f.ActorUserID != nil {
		q = q.Where("actor_user_id = ?", *f.ActorUserID)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.From != nil {
		q = q.Where("occurred_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("occurred_at <= ?", *f.To)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []auditRow
	err := q.Order("occurred_at DESC").Offset(f.Page.Offset()).Limit(f.Page.Size).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.AuditLog, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toDomain())
	}
	return out, total, nil
}
