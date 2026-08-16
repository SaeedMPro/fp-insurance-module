package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Attachment is one supporting document uploaded against a claim — typically a
// scan or photo of the invoice. The binary itself lives on disk (see
// platform/filestore); this record holds the metadata and the storage key.
type Attachment struct {
	ID         uuid.UUID
	ClaimID    uuid.UUID
	FileName   string // original name, shown to the user — never used as a disk path
	FilePath   string // storage key relative to the attachments root
	UploadedAt time.Time
}

// AttachmentUploadableStatuses are the claim states in which documents may
// still be added: while the employee is preparing the claim, and after a
// reviewer sends it back for missing paperwork. Once a claim is in review or
// decided, its evidence is frozen so the reviewer's basis cannot change
// underneath the decision.
var AttachmentUploadableStatuses = []ClaimStatus{ClaimDraft, ClaimReturnedForDocs}

// AcceptsAttachments reports whether a claim in this status may receive more
// documents.
func (c Claim) AcceptsAttachments() bool {
	for _, s := range AttachmentUploadableStatuses {
		if c.Status == s {
			return true
		}
	}
	return false
}

// SafeFileName strips any directory component a client may have sent and
// bounds the length, so the stored display name is always a plain file name.
func SafeFileName(name string) string {
	name = strings.TrimSpace(name)
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	if name == "" {
		name = "document"
	}
	const maxLen = 200 // the column allows 255; leave room for multi-byte names
	if len(name) > maxLen {
		name = name[:maxLen]
	}
	return name
}
