package api

import (
	"net/http"

	"insurance-module/internal/audit"
)

type auditListResponse struct {
	Items interface{} `json:"items"`
	Total int64       `json:"total"`
}

func (h *handlers) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	page, pageSize := pagination(r)
	from, to := dateRange(r)

	filter := audit.Filter{
		EntityType: r.URL.Query().Get("entity_type"),
		EntityID:   r.URL.Query().Get("entity_id"),
		Action:     r.URL.Query().Get("action"),
		From:       from,
		To:         to,
		Page:       page,
		PageSize:   pageSize,
	}
	if actor := r.URL.Query().Get("actor_user_id"); actor != "" {
		if id, err := parseUUIDField(actor); err == nil {
			filter.ActorUserID = &id
		}
	}

	rows, total, err := h.d.Audit.Query(r.Context(), filter)
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, auditListResponse{Items: rows, Total: total})
}
