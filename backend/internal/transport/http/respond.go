// Package http is the transport layer: chi router, authentication/RBAC
// middleware, request/response DTOs (the wire format lives HERE, not on
// domain or storage types), and the single mapping from domain errors to
// HTTP status codes.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"insurance-module/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type errorBody struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorBody{Error: message})
}

// respondError is the one place a domain error becomes an HTTP response.
// Internal (unclassified) errors are logged with detail but presented
// generically — nothing internal leaks to clients.
func respondError(w http.ResponseWriter, r *http.Request, err error) {
	kind := domain.KindOf(err)
	status := statusFor(kind)
	if kind == domain.KindInternal {
		slog.ErrorContext(r.Context(), "internal error", "error", err, "path", r.URL.Path)
	}
	writeError(w, status, domain.MessageOf(err))
}

func statusFor(kind domain.Kind) int {
	switch kind {
	case domain.KindNotFound:
		return http.StatusNotFound
	case domain.KindUnauthorized:
		return http.StatusUnauthorized
	case domain.KindForbidden:
		return http.StatusForbidden
	case domain.KindConflict:
		return http.StatusConflict
	case domain.KindValidation:
		return http.StatusBadRequest
	case domain.KindUnprocessable:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

func decodeJSON(r *http.Request, dst any) error {
	defer func() { _ = r.Body.Close() }()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.Validationf("invalid request body")
	}
	return nil
}
