package http

import (
	"net/http"

	"insurance-module/internal/domain"
)

func (s *Server) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	from, to := dateRange(r)
	filter := domain.AuditFilter{
		EntityType: r.URL.Query().Get("entity_type"),
		EntityID:   r.URL.Query().Get("entity_id"),
		Action:     r.URL.Query().Get("action"),
		From:       from,
		To:         to,
		Page:       pageParams(r),
	}
	if actorParam := r.URL.Query().Get("actor_user_id"); actorParam != "" {
		if id, err := parseUUID(actorParam, "actor_user_id"); err == nil {
			filter.ActorUserID = &id
		}
	}

	items, total, err := s.audit.Query(r.Context(), filter)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, listDTO[auditLogDTO]{Items: mapSlice(items, toAuditLogDTO), Total: total})
}

func reportRange(r *http.Request) domain.ReportRange {
	from, to := dateRange(r)
	return domain.ReportRange{From: from, To: to}
}

func (s *Server) handleReportSummary(w http.ResponseWriter, r *http.Request) {
	out, err := s.reports.Summary(r.Context(), reportRange(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, summaryDTO{
		TotalClaims:             out.TotalClaims,
		TotalPaidAmount:         out.TotalPaidAmount.Float(),
		PendingReview:           out.PendingReview,
		ApprovedAwaitingPayment: out.ApprovedAwaitingPayment,
		Rejected:                out.Rejected,
	})
}

func (s *Server) handleSpendByEmployee(w http.ResponseWriter, r *http.Request) {
	out, err := s.reports.SpendByEmployee(r.Context(), reportRange(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(out, func(v domain.EmployeeSpend) employeeSpendDTO {
		return employeeSpendDTO{
			EmployeeID: v.EmployeeID, EmployeeName: v.EmployeeName,
			PersonnelNo: v.PersonnelNo, TotalPaid: v.TotalPaid.Float(), ClaimCount: v.ClaimCount,
		}
	}))
}

func (s *Server) handleSpendByServiceType(w http.ResponseWriter, r *http.Request) {
	out, err := s.reports.SpendByServiceType(r.Context(), reportRange(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(out, func(v domain.ServiceTypeSpend) serviceTypeSpendDTO {
		return serviceTypeSpendDTO{
			ServiceTypeCode: v.ServiceTypeCode, ServiceTypeName: v.ServiceTypeName,
			TotalPaid: v.TotalPaid.Float(), ClaimCount: v.ClaimCount,
		}
	}))
}

func (s *Server) handleSpendByMonth(w http.ResponseWriter, r *http.Request) {
	out, err := s.reports.SpendByMonth(r.Context(), reportRange(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(out, func(v domain.MonthSpend) monthSpendDTO {
		return monthSpendDTO{Month: v.Month, TotalPaid: v.TotalPaid.Float(), ClaimCount: v.ClaimCount}
	}))
}
