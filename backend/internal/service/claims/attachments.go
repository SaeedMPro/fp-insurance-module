package claims

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
)

// Attachment policy errors.
var (
	ErrAttachmentsFrozen = domain.Conflictf(
		"documents can only be added while the claim is a draft or has been returned for documents")
	ErrAttachmentTooLarge  = domain.Validationf("file exceeds the maximum allowed size")
	ErrAttachmentEmpty     = domain.Validationf("file is empty")
	ErrAttachmentTypeNotOK = domain.Validationf("unsupported file type")
	ErrAttachmentMissing   = domain.NotFoundf("attachment file not found")
)

// MaxAttachmentBytes bounds a single upload. Invoice scans and phone photos sit
// well under this; the limit exists so one request cannot fill the disk.
const MaxAttachmentBytes int64 = 5 << 20 // 5 MiB

// allowedAttachmentTypes maps accepted MIME types to the extension used on
// disk. Sniffed content — not the client's declared type — is matched against
// this, so renaming an executable to .pdf does not get it stored.
var allowedAttachmentTypes = map[string]string{
	"application/pdf": ".pdf",
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
}

// AllowedAttachmentTypes lists the accepted MIME types (for the API contract
// and the file picker's accept attribute).
func AllowedAttachmentTypes() []string {
	out := make([]string, 0, len(allowedAttachmentTypes))
	for t := range allowedAttachmentTypes {
		out = append(out, t)
	}
	return out
}

// FileStore is the blob side of attachments (platform/filestore).
type FileStore interface {
	Save(prefix, ext string, r io.Reader) (string, error)
	Open(key string) (io.ReadSeekCloser, int64, error)
	Remove(key string) error
}

// UploadInput is one incoming document.
type UploadInput struct {
	FileName string
	// Content is read at most MaxAttachmentBytes+1 bytes; the caller does not
	// need to have buffered the whole body.
	Content io.Reader
}

// ListAttachments returns a claim's documents. Anyone allowed to read the claim
// may see them: the owner, reviewers, admins and auditors.
func (s *Service) ListAttachments(ctx context.Context, actor domain.Actor, claimID uuid.UUID) ([]domain.Attachment, error) {
	if _, err := s.Get(ctx, actor, claimID); err != nil {
		return nil, err
	}
	return s.repo.ListAttachments(ctx, claimID)
}

// AddAttachment stores one document against a claim.
//
// Two rules are enforced here rather than in the handler, because they are
// business rules: only the claim's owner (or an admin acting for them) may add
// evidence, and only while the claim is still being prepared — a draft, or one
// a reviewer returned for missing paperwork. That second rule is what makes the
// "returned for documents" loop meaningful: the reviewer's decision is always
// based on evidence that could not change after they started reviewing.
func (s *Service) AddAttachment(ctx context.Context, actor domain.Actor, claimID uuid.UUID, in UploadInput) (domain.Attachment, error) {
	if s.files == nil {
		return domain.Attachment{}, domain.Internalf(nil, "attachment storage is not configured")
	}

	claim, err := s.repo.GetClaim(ctx, claimID)
	if err != nil {
		return domain.Attachment{}, err
	}
	if actor.Role != domain.RoleAdmin && claim.CreatedBy != actor.UserID {
		return domain.Attachment{}, ErrForbidden
	}
	if !claim.AcceptsAttachments() {
		return domain.Attachment{}, ErrAttachmentsFrozen
	}

	// Read one byte past the limit so an oversized upload is detected without
	// buffering the whole thing.
	body, err := io.ReadAll(io.LimitReader(in.Content, MaxAttachmentBytes+1))
	if err != nil {
		return domain.Attachment{}, domain.Internalf(err, "reading upload")
	}
	if int64(len(body)) > MaxAttachmentBytes {
		return domain.Attachment{}, ErrAttachmentTooLarge
	}
	if len(body) == 0 {
		return domain.Attachment{}, ErrAttachmentEmpty
	}

	ext, ok := allowedAttachmentTypes[detectContentType(body)]
	if !ok {
		return domain.Attachment{}, ErrAttachmentTypeNotOK
	}

	key, err := s.files.Save(claimID.String(), ext, bytes.NewReader(body))
	if err != nil {
		return domain.Attachment{}, domain.Internalf(err, "storing upload")
	}

	att := domain.Attachment{
		ClaimID:    claimID,
		FileName:   domain.SafeFileName(in.FileName),
		FilePath:   key,
		UploadedAt: s.clock.Now(),
	}

	// Metadata row and audit entry commit together; if either fails the blob is
	// removed so no orphan file is left on disk.
	err = s.atomic(ctx, func(r Repo) error {
		if err := r.CreateAttachment(ctx, &att); err != nil {
			return err
		}
		return r.InsertAudit(ctx, &domain.AuditLog{
			EntityType:    "claim",
			EntityID:      claimID.String(),
			Action:        "attachment_upload",
			ActorUserID:   &actor.UserID,
			ActorUsername: actor.Username,
			AfterData:     map[string]any{"file_name": att.FileName, "attachment_id": att.ID.String()},
			OccurredAt:    s.clock.Now(),
		})
	})
	if err != nil {
		_ = s.files.Remove(key)
		return domain.Attachment{}, err
	}
	return att, nil
}

// OpenAttachment returns the stored bytes for download, after checking the
// caller may read the parent claim.
func (s *Service) OpenAttachment(ctx context.Context, actor domain.Actor, claimID, attachmentID uuid.UUID) (domain.Attachment, io.ReadSeekCloser, int64, error) {
	if s.files == nil {
		return domain.Attachment{}, nil, 0, domain.Internalf(nil, "attachment storage is not configured")
	}
	if _, err := s.Get(ctx, actor, claimID); err != nil {
		return domain.Attachment{}, nil, 0, err
	}
	att, err := s.repo.GetAttachment(ctx, attachmentID)
	if err != nil {
		return domain.Attachment{}, nil, 0, err
	}
	// Guard against an attachment id from a different claim being used against
	// a claim the caller happens to be allowed to read.
	if att.ClaimID != claimID {
		return domain.Attachment{}, nil, 0, domain.NotFoundf("attachment not found")
	}

	reader, size, err := s.files.Open(att.FilePath)
	if err != nil {
		// A row without its blob (e.g. demo data, or a lost volume) is a
		// missing document, not a server fault.
		return domain.Attachment{}, nil, 0, ErrAttachmentMissing
	}
	return att, reader, size, nil
}

// detectContentType sniffs the real type from the leading bytes. http's
// DetectContentType does not know webp, so that one is matched explicitly.
func detectContentType(body []byte) string {
	if isWebP(body) {
		return "image/webp"
	}
	ct := http.DetectContentType(body)
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.TrimSpace(ct)
}

func isWebP(b []byte) bool {
	return len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WEBP"
}

// ExtensionFor returns the on-disk extension for a detected type (test helper
// and documentation of the mapping).
func ExtensionFor(mime string) string { return allowedAttachmentTypes[mime] }
