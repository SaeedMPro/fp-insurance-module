package claims_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"insurance-module/internal/app/apptest"
	"insurance-module/internal/domain"
	"insurance-module/internal/service/claims"
)

// A minimal but genuinely valid PDF: content sniffing looks at the leading
// bytes, so the fixture has to start with a real %PDF- signature.
var samplePDF = []byte("%PDF-1.4\n1 0 obj<</Type/Catalog>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF\n")

// 1x1 transparent PNG.
var samplePNG = []byte{
	0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 'I', 'D', 'A', 'T',
	0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05,
	0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00,
	0x00, 0x00, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82,
}

func upload(t *testing.T, svcs interface {
	AddAttachment(context.Context, domain.Actor, uuid.UUID, claims.UploadInput) (domain.Attachment, error)
}, actor domain.Actor, claimID uuid.UUID, name string, body []byte) (domain.Attachment, error) {
	t.Helper()
	return svcs.AddAttachment(context.Background(), actor, claimID, claims.UploadInput{
		FileName: name, Content: bytes.NewReader(body),
	})
}

func TestAttachments_UploadListDownloadOnDraft(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()
	fx := setup(t, store, svcs)
	claim := newClaim(t, store, svcs, fx, 400000) // starts as draft

	att, err := upload(t, svcs.Claims, fx.employeeActor, claim.ID, "فاکتور داروخانه.pdf", samplePDF)
	require.NoError(t, err)
	require.Equal(t, "فاکتور داروخانه.pdf", att.FileName, "the display name is preserved as sent")
	require.NotEmpty(t, att.FilePath)
	require.NotContains(t, att.FilePath, "..")

	list, err := svcs.Claims.ListAttachments(ctx, fx.employeeActor, claim.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)

	_, reader, size, err := svcs.Claims.OpenAttachment(ctx, fx.employeeActor, claim.ID, att.ID)
	require.NoError(t, err)
	defer reader.Close()
	require.Equal(t, int64(len(samplePDF)), size)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, samplePDF, got, "downloaded bytes must match what was uploaded")

	// The upload is auditable like every other state-changing action.
	trail, err := svcs.Claims.History(ctx, fx.reviewerActor, claim.ID)
	require.NoError(t, err)
	var found bool
	for _, e := range trail {
		if e.Action == "attachment_upload" {
			found = true
		}
	}
	require.True(t, found, "attachment upload must appear in the audit trail")
}

// The whole point of the feature: a claim sent back for documents must accept
// new evidence, and stop accepting it again once resubmitted.
func TestAttachments_ReturnedForDocsLoop(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()
	fx := setup(t, store, svcs)
	claim := newClaim(t, store, svcs, fx, 400000)

	_, err := svcs.Claims.Submit(ctx, fx.employeeActor, claim.ID)
	require.NoError(t, err)

	// Submitted: evidence is frozen.
	_, err = upload(t, svcs.Claims, fx.employeeActor, claim.ID, "late.pdf", samplePDF)
	require.ErrorIs(t, err, claims.ErrAttachmentsFrozen)

	_, err = svcs.Claims.StartReview(ctx, fx.reviewerActor, claim.ID)
	require.NoError(t, err)
	_, err = svcs.Claims.ReturnForDocs(ctx, fx.reviewerActor, claim.ID, "لطفاً نسخهٔ پزشک را پیوست کنید")
	require.NoError(t, err)

	// Returned for documents: uploads are open again.
	_, err = upload(t, svcs.Claims, fx.employeeActor, claim.ID, "prescription.png", samplePNG)
	require.NoError(t, err)

	_, err = svcs.Claims.Resubmit(ctx, fx.employeeActor, claim.ID)
	require.NoError(t, err)

	// Resubmitted: frozen again.
	_, err = upload(t, svcs.Claims, fx.employeeActor, claim.ID, "another.pdf", samplePDF)
	require.ErrorIs(t, err, claims.ErrAttachmentsFrozen)
}

func TestAttachments_RejectsUnsupportedAndEmpty(t *testing.T) {
	store, svcs := apptest.Open(t)
	fx := setup(t, store, svcs)
	claim := newClaim(t, store, svcs, fx, 100000)

	// A script renamed to .pdf is caught by content sniffing, not the name.
	_, err := upload(t, svcs.Claims, fx.employeeActor, claim.ID, "invoice.pdf",
		[]byte("<html><script>alert(1)</script></html>"))
	require.ErrorIs(t, err, claims.ErrAttachmentTypeNotOK)

	_, err = upload(t, svcs.Claims, fx.employeeActor, claim.ID, "empty.pdf", nil)
	require.ErrorIs(t, err, claims.ErrAttachmentEmpty)

	_, err = upload(t, svcs.Claims, fx.employeeActor, claim.ID, "big.pdf",
		append(append([]byte{}, samplePDF...), bytes.Repeat([]byte("A"), int(claims.MaxAttachmentBytes))...))
	require.ErrorIs(t, err, claims.ErrAttachmentTooLarge)
}

func TestAttachments_AccessControl(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()
	owner := setup(t, store, svcs)
	other := setup(t, store, svcs)
	claim := newClaim(t, store, svcs, owner, 100000)

	att, err := upload(t, svcs.Claims, owner.employeeActor, claim.ID, "receipt.pdf", samplePDF)
	require.NoError(t, err)

	// A different employee can neither upload to nor read this claim.
	_, err = upload(t, svcs.Claims, other.employeeActor, claim.ID, "sneaky.pdf", samplePDF)
	require.Error(t, err)
	_, err = svcs.Claims.ListAttachments(ctx, other.employeeActor, claim.ID)
	require.Error(t, err)
	_, _, _, err = svcs.Claims.OpenAttachment(ctx, other.employeeActor, claim.ID, att.ID)
	require.Error(t, err)

	// A reviewer may read the documents but not add to them.
	list, err := svcs.Claims.ListAttachments(ctx, owner.reviewerActor, claim.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	_, err = upload(t, svcs.Claims, owner.reviewerActor, claim.ID, "reviewer.pdf", samplePDF)
	require.ErrorIs(t, err, claims.ErrForbidden)
}

// An attachment id belonging to another claim must not be readable by pointing
// it at a claim the caller happens to be allowed to see.
func TestAttachments_CrossClaimIDRejected(t *testing.T) {
	store, svcs := apptest.Open(t)
	ctx := context.Background()
	fx := setup(t, store, svcs)
	claimA := newClaim(t, store, svcs, fx, 100000)
	claimB := newClaim(t, store, svcs, fx, 200000)

	att, err := upload(t, svcs.Claims, fx.employeeActor, claimA.ID, "a.pdf", samplePDF)
	require.NoError(t, err)

	_, _, _, err = svcs.Claims.OpenAttachment(ctx, fx.employeeActor, claimB.ID, att.ID)
	require.Error(t, err)
	require.Equal(t, domain.KindNotFound, domain.KindOf(err))
}

func TestSafeFileName(t *testing.T) {
	cases := map[string]string{
		"invoice.pdf":            "invoice.pdf",
		"../../etc/passwd":       "passwd",
		`C:\Users\x\receipt.jpg`: "receipt.jpg",
		"  spaced.pdf  ":         "spaced.pdf",
		"":                       "document",
		"/":                      "document",
	}
	for in, want := range cases {
		require.Equal(t, want, domain.SafeFileName(in), "input %q", in)
	}
	require.LessOrEqual(t, len(domain.SafeFileName(strings.Repeat("a", 500))), 200)
}
