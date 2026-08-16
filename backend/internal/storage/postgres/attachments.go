package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
)

type attachmentRow struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ClaimID    uuid.UUID `gorm:"type:uuid"`
	FileName   string
	FilePath   string
	UploadedAt time.Time
}

func (attachmentRow) TableName() string { return "claim_attachments" }

func (r attachmentRow) toDomain() domain.Attachment {
	return domain.Attachment{
		ID: r.ID, ClaimID: r.ClaimID, FileName: r.FileName,
		FilePath: r.FilePath, UploadedAt: r.UploadedAt,
	}
}

func attachmentFromDomain(a domain.Attachment) attachmentRow {
	return attachmentRow{
		ID: a.ID, ClaimID: a.ClaimID, FileName: a.FileName,
		FilePath: a.FilePath, UploadedAt: a.UploadedAt,
	}
}

func (s *Store) ListAttachments(ctx context.Context, claimID uuid.UUID) ([]domain.Attachment, error) {
	var rows []attachmentRow
	if err := s.ctx(ctx).Where("claim_id = ?", claimID).
		Order("uploaded_at").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Attachment, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toDomain())
	}
	return out, nil
}

func (s *Store) GetAttachment(ctx context.Context, id uuid.UUID) (domain.Attachment, error) {
	var row attachmentRow
	if err := s.ctx(ctx).First(&row, "id = ?", id).Error; err != nil {
		return domain.Attachment{}, notFound(err, "attachment")
	}
	return row.toDomain(), nil
}

func (s *Store) CreateAttachment(ctx context.Context, a *domain.Attachment) error {
	row := attachmentFromDomain(*a)
	if err := s.ctx(ctx).Create(&row).Error; err != nil {
		return err
	}
	*a = row.toDomain()
	return nil
}
