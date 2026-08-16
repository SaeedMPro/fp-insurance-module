package http

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
	"insurance-module/internal/service/claims"
)

type attachmentDTO struct {
	ID          uuid.UUID `json:"id"`
	ClaimID     uuid.UUID `json:"claim_id"`
	FileName    string    `json:"file_name"`
	UploadedAt  time.Time `json:"uploaded_at"`
	DownloadURL string    `json:"download_url"`
}

// toAttachmentDTO deliberately omits the on-disk path: it is an internal
// storage key, useless to clients and needless information for an attacker.
func toAttachmentDTO(a domain.Attachment) attachmentDTO {
	return attachmentDTO{
		ID:          a.ID,
		ClaimID:     a.ClaimID,
		FileName:    a.FileName,
		UploadedAt:  a.UploadedAt,
		DownloadURL: fmt.Sprintf("/api/v1/claims/%s/attachments/%s/download", a.ClaimID, a.ID),
	}
}

func (s *Server) handleListAttachments(w http.ResponseWriter, r *http.Request) {
	actor, ok := mustActor(w, r)
	if !ok {
		return
	}
	claimID, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	items, err := s.claims.ListAttachments(r.Context(), actor, claimID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(items, toAttachmentDTO))
}

func (s *Server) handleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	actor, ok := mustActor(w, r)
	if !ok {
		return
	}
	claimID, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}

	// Two separate bounds. MaxBytesReader caps the whole request, so an
	// oversized upload is refused at the socket rather than buffered. The
	// ParseMultipartForm argument is only how much may stay in RAM — anything
	// larger spills to a temp file that RemoveAll cleans up below — so it is
	// kept small and independent of the file-size limit.
	const maxInMemory = 1 << 20 // 1 MiB
	r.Body = http.MaxBytesReader(w, r.Body, claims.MaxAttachmentBytes+(1<<20))
	// #nosec G120 -- the request body is already capped by the MaxBytesReader
	// on the line above; this argument only sizes the in-memory buffer.
	if err := r.ParseMultipartForm(maxInMemory); err != nil {
		respondError(w, r, domain.Validationf("upload is not a valid multipart form or exceeds the size limit"))
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, r, domain.Validationf("no file was sent in the 'file' field"))
		return
	}
	defer func() { _ = file.Close() }()

	name := ""
	if header != nil {
		name = header.Filename
	}

	att, err := s.claims.AddAttachment(r.Context(), actor, claimID, claims.UploadInput{
		FileName: name,
		Content:  file,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAttachmentDTO(att))
}

func (s *Server) handleDownloadAttachment(w http.ResponseWriter, r *http.Request) {
	actor, ok := mustActor(w, r)
	if !ok {
		return
	}
	claimID, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	attachmentID, err := pathUUID(r, "attachmentID")
	if err != nil {
		respondError(w, r, err)
		return
	}

	att, reader, size, err := s.claims.OpenAttachment(r.Context(), actor, claimID, attachmentID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	defer func() { _ = reader.Close() }()

	// Serve as a download, never inline: an uploaded HTML/SVG rendered in the
	// browser under our origin would be a stored-XSS vector.
	w.Header().Set("Content-Type", contentTypeFor(att.FilePath))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", contentDisposition(att.FileName))
	http.ServeContent(w, r, att.FileName, att.UploadedAt, reader)
}

func contentTypeFor(storageKey string) string {
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(storageKey))); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

// contentDisposition emits both a plain and an RFC 5987 encoded name so Persian
// file names survive; ASCII-only clients fall back to the sanitised form.
func contentDisposition(fileName string) string {
	ascii := strings.Map(func(r rune) rune {
		if r < 32 || r > 126 || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, fileName)
	if strings.TrimLeft(ascii, "_") == "" {
		ascii = "document"
	}
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
		ascii, url.PathEscape(fileName))
}
