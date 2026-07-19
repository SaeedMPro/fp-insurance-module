package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	appmw "insurance-module/internal/api/middleware"
	"insurance-module/internal/models"
	"insurance-module/internal/ruleengine"
	"insurance-module/internal/workflow"
)

type handlers struct {
	d Deps
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// mapDomainError translates known service-layer sentinel errors to HTTP status
// codes per docs/API-CONTRACT.md; anything unrecognized is a 500.
func mapDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, workflow.ErrInvalidTransition):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, workflow.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, workflow.ErrReasonRequired):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ruleengine.ErrNoActiveRule),
		errors.Is(err, ruleengine.ErrNotEligible),
		errors.Is(err, ruleengine.ErrWaitingPeriod),
		errors.Is(err, ruleengine.ErrEmployeeInactive),
		errors.Is(err, ruleengine.ErrDependentMismatch):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func decodeJSON(r *http.Request, dst interface{}) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func pathUUID(r *http.Request, key string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, key))
}

func parseUUIDField(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// pagination reads page/page_size query params with sane defaults/bounds.
func pagination(r *http.Request) (page, pageSize int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	return page, pageSize
}

// dateRange reads optional ?from=YYYY-MM-DD&to=YYYY-MM-DD query params.
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

// currentActor builds a workflow.Actor directly from the verified JWT claims —
// no extra DB lookup is needed since the token already carries user id/role.
func currentActor(r *http.Request) (workflow.Actor, bool) {
	claims, ok := appmw.UserFromContext(r.Context())
	if !ok {
		return workflow.Actor{}, false
	}
	return workflow.Actor{UserID: claims.UserID, Username: claims.Username, Role: claims.Role}, true
}

// requireSelfOrRole allows admin/reviewer through unconditionally, and an
// "employee" role user only when their own employee_id matches the target.
func (h *handlers) authorizeEmployeeAccess(r *http.Request, employeeID uuid.UUID) bool {
	claims, ok := appmw.UserFromContext(r.Context())
	if !ok {
		return false
	}
	switch claims.Role {
	case models.RoleAdmin, models.RoleReviewer:
		return true
	case models.RoleEmployee:
		var user models.User
		if err := h.d.DB.WithContext(r.Context()).First(&user, "id = ?", claims.UserID).Error; err != nil {
			return false
		}
		return user.EmployeeID != nil && *user.EmployeeID == employeeID
	default:
		return false
	}
}

// authorizeClaimAccess allows admin/reviewer/auditor through unconditionally,
// and an "employee" role user only when they created the claim.
func (h *handlers) authorizeClaimAccess(r *http.Request, claim *models.Claim) bool {
	claims, ok := appmw.UserFromContext(r.Context())
	if !ok {
		return false
	}
	switch claims.Role {
	case models.RoleAdmin, models.RoleReviewer, models.RoleAuditor:
		return true
	case models.RoleEmployee:
		return claim.CreatedBy == claims.UserID
	default:
		return false
	}
}
