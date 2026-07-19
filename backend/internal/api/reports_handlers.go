package api

import (
	"net/http"

	"insurance-module/internal/reports"
)

func (h *handlers) reportsFilter(r *http.Request) reports.Filter {
	from, to := dateRange(r)
	return reports.Filter{From: from, To: to}
}

func (h *handlers) reportSummary(w http.ResponseWriter, r *http.Request) {
	out, err := h.d.Reports.Summary(r.Context(), h.reportsFilter(r))
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) reportSpendByEmployee(w http.ResponseWriter, r *http.Request) {
	out, err := h.d.Reports.SpendByEmployee(r.Context(), h.reportsFilter(r))
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) reportSpendByServiceType(w http.ResponseWriter, r *http.Request) {
	out, err := h.d.Reports.SpendByServiceType(r.Context(), h.reportsFilter(r))
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) reportSpendByMonth(w http.ResponseWriter, r *http.Request) {
	out, err := h.d.Reports.SpendByMonth(r.Context(), h.reportsFilter(r))
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
