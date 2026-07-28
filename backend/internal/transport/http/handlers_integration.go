package http

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"insurance-module/internal/domain"
	"insurance-module/internal/service/integration"
)

type syncEmployeeRequest struct {
	PersonnelNo      string                  `json:"personnel_no"`
	FullName         string                  `json:"full_name"`
	NationalID       string                  `json:"national_id"`
	EmploymentStatus domain.EmploymentStatus `json:"employment_status"`
	HireDate         time.Time               `json:"hire_date"`
	Department       string                  `json:"department"`
	PlanID           *string                 `json:"plan_id"`
}

type syncEmployeesRequest struct {
	Employees []syncEmployeeRequest `json:"employees"`
}

type syncEmployeesResponse struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
}

func (s *Server) handleSyncEmployees(w http.ResponseWriter, r *http.Request) {
	var req syncEmployeesRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	batch := make([]integration.SyncEmployee, 0, len(req.Employees))
	for _, e := range req.Employees {
		planID, err := parseOptionalUUID(e.PlanID, "plan_id")
		if err != nil {
			respondError(w, r, err)
			return
		}
		batch = append(batch, integration.SyncEmployee{
			PersonnelNo:      e.PersonnelNo,
			FullName:         e.FullName,
			NationalID:       e.NationalID,
			EmploymentStatus: e.EmploymentStatus,
			HireDate:         e.HireDate,
			Department:       e.Department,
			PlanID:           planID,
		})
	}
	res, err := s.integration.SyncEmployees(r.Context(), batch)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, syncEmployeesResponse{Created: res.Created, Updated: res.Updated})
}

type claimStatusResponse struct {
	ID            uuid.UUID          `json:"id"`
	Status        domain.ClaimStatus `json:"status"`
	PayableAmount *float64           `json:"payable_amount"`
}

func (s *Server) handleIntegrationClaimStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	claim, err := s.integration.ClaimStatus(r.Context(), id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, claimStatusResponse{
		ID: claim.ID, Status: claim.Status, PayableAmount: domain.FloatPtrFromRialPtr(claim.PayableAmount),
	})
}
