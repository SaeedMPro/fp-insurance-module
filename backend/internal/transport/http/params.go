package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"insurance-module/internal/domain"
)

// pageParams reads ?page= and ?page_size= with the domain's clamping rules.
func pageParams(r *http.Request) domain.Page {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	return domain.NewPage(page, size)
}

// dateRange reads optional ?from=YYYY-MM-DD and ?to=YYYY-MM-DD params.
func dateRange(r *http.Request) (from, to *time.Time) {
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			from = &t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			to = &t
		}
	}
	return from, to
}

func pathUUID(r *http.Request, key string) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	if err != nil {
		return uuid.UUID{}, domain.Validationf("invalid id")
	}
	return id, nil
}

// parseUUID validates a request-body UUID string with a field-specific error.
func parseUUID(s, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, domain.Validationf("invalid %s", field)
	}
	return id, nil
}

// parseOptionalUUID maps nil → nil and validates non-nil values.
func parseOptionalUUID(s *string, field string) (*uuid.UUID, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	id, err := parseUUID(*s, field)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
