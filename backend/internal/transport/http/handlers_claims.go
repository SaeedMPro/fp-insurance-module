package http

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
	"insurance-module/internal/service/claims"
)

type createClaimRequest struct {
	EmployeeID      *string                `json:"employee_id"`
	BeneficiaryType domain.BeneficiaryType `json:"beneficiary_type"`
	DependentID     *string                `json:"dependent_id"`
	ServiceTypeID   string                 `json:"service_type_id"`
	RequestedAmount float64                `json:"requested_amount"`
	ReceiptDate     time.Time              `json:"receipt_date"`
	Description     string                 `json:"description"`
}

func (s *Server) handleCreateClaim(w http.ResponseWriter, r *http.Request) {
	actor, ok := mustActor(w, r)
	if !ok {
		return
	}
	var req createClaimRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	employeeID, err := parseOptionalUUID(req.EmployeeID, "employee_id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	serviceTypeID, err := parseUUID(req.ServiceTypeID, "service_type_id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	dependentID, err := parseOptionalUUID(req.DependentID, "dependent_id")
	if err != nil {
		respondError(w, r, err)
		return
	}

	claim, err := s.claims.Create(r.Context(), actor, claims.CreateInput{
		EmployeeID:      employeeID,
		BeneficiaryType: req.BeneficiaryType,
		DependentID:     dependentID,
		ServiceTypeID:   serviceTypeID,
		RequestedAmount: req.RequestedAmount,
		ReceiptDate:     req.ReceiptDate,
		Description:     req.Description,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toClaimDTO(claim))
}

func (s *Server) handleListClaims(w http.ResponseWriter, r *http.Request) {
	actor, ok := mustActor(w, r)
	if !ok {
		return
	}
	from, to := dateRange(r)
	filter := domain.ClaimFilter{
		EmployeeID:    r.URL.Query().Get("employee_id"),
		Status:        domain.ClaimStatus(r.URL.Query().Get("status")),
		ServiceTypeID: r.URL.Query().Get("service_type_id"),
		From:          from,
		To:            to,
		Page:          pageParams(r),
	}
	items, total, err := s.claims.List(r.Context(), actor, filter)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, listDTO[claimDTO]{Items: mapSlice(items, toClaimDTO), Total: total})
}

func (s *Server) handleGetClaim(w http.ResponseWriter, r *http.Request) {
	actor, ok := mustActor(w, r)
	if !ok {
		return
	}
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	claim, err := s.claims.Get(r.Context(), actor, id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toClaimDTO(claim))
}

func (s *Server) handleClaimHistory(w http.ResponseWriter, r *http.Request) {
	actor, ok := mustActor(w, r)
	if !ok {
		return
	}
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	trail, err := s.claims.History(r.Context(), actor, id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(trail, toAuditLogDTO))
}

type reasonRequest struct {
	Reason string `json:"reason"`
}

// transition adapts one workflow action to an HTTP handler.
func (s *Server) transition(action func(r *http.Request, actor domain.Actor, id uuid.UUID) (domain.Claim, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := mustActor(w, r)
		if !ok {
			return
		}
		id, err := pathUUID(r, "id")
		if err != nil {
			respondError(w, r, err)
			return
		}
		claim, err := action(r, actor, id)
		if err != nil {
			respondError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toClaimDTO(claim))
	}
}

// withReason decodes the mandatory-reason body for reject / return-for-docs.
func withReason(r *http.Request) (string, error) {
	var req reasonRequest
	if err := decodeJSON(r, &req); err != nil {
		return "", err
	}
	return req.Reason, nil
}
