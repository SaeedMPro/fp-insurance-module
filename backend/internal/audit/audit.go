// Package audit records every state-changing action in the system (create, edit,
// approve, reject, payment, config change, login) with actor, timestamp, and a
// before/after snapshot, and lets that trail be queried back per entity or per user.
package audit

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

// Entry describes one auditable event. EntityID is a string so both UUID entities
// (claims, rules) and non-UUID identifiers can be logged uniformly.
type Entry struct {
	EntityType    string
	EntityID      string
	Action        string
	ActorUserID   *uuid.UUID
	ActorUsername string
	Before        map[string]interface{}
	After         map[string]interface{}
	Metadata      map[string]interface{}
}

// Log writes one audit record. It uses the caller's *gorm.DB handle (which may be
// a transaction) so an audit entry commits atomically with the change it describes.
func (s *Service) Log(ctx context.Context, tx *gorm.DB, e Entry) error {
	if tx == nil {
		tx = s.db
	}
	row := models.AuditLog{
		EntityType:    e.EntityType,
		EntityID:      e.EntityID,
		Action:        e.Action,
		ActorUserID:   e.ActorUserID,
		ActorUsername: e.ActorUsername,
		BeforeData:    models.JSONMap(e.Before),
		AfterData:     models.JSONMap(e.After),
		Metadata:      models.JSONMap(e.Metadata),
		OccurredAt:    time.Now(),
	}
	return tx.WithContext(ctx).Create(&row).Error
}

// Filter narrows a query over the audit trail. Zero-value fields are unconstrained.
type Filter struct {
	EntityType  string
	EntityID    string
	ActorUserID *uuid.UUID
	Action      string
	From        *time.Time
	To          *time.Time
	Page        int
	PageSize    int
}

// Query returns matching audit rows (newest first) and the total match count for pagination.
func (s *Service) Query(ctx context.Context, f Filter) ([]models.AuditLog, int64, error) {
	q := s.db.WithContext(ctx).Model(&models.AuditLog{})
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

	page, pageSize := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var rows []models.AuditLog
	err := q.Order("occurred_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error
	return rows, total, err
}

// Trail is a convenience wrapper for "full history of one entity", used by the
// claim detail page's audit/history view.
func (s *Service) Trail(ctx context.Context, entityType, entityID string) ([]models.AuditLog, error) {
	rows, _, err := s.Query(ctx, Filter{EntityType: entityType, EntityID: entityID, Page: 1, PageSize: 200})
	return rows, err
}
