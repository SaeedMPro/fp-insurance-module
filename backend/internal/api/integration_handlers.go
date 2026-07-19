package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"insurance-module/internal/models"
)

type syncEmployee struct {
	PersonnelNo      string                  `json:"personnel_no"`
	FullName         string                  `json:"full_name"`
	NationalID       string                  `json:"national_id"`
	EmploymentStatus models.EmploymentStatus `json:"employment_status"`
	HireDate         time.Time               `json:"hire_date"`
	Department       string                  `json:"department"`
	PlanID           *string                 `json:"plan_id"`
}

type syncEmployeesRequest struct {
	Employees []syncEmployee `json:"employees"`
}

type syncEmployeesResponse struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
}

// integrationSyncEmployees simulates pulling employee master data from the
// parent system (real integration is out of scope per the proposal): it upserts
// by personnel_no.
func (h *handlers) integrationSyncEmployees(w http.ResponseWriter, r *http.Request) {
	var req syncEmployeesRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp := syncEmployeesResponse{}
	err := h.d.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		for _, se := range req.Employees {
			var existing models.Employee
			err := tx.Where("personnel_no = ?", se.PersonnelNo).First(&existing).Error
			var planID *uuid.UUID
			if se.PlanID != nil {
				id, perr := parseUUIDField(*se.PlanID)
				if perr != nil {
					return perr
				}
				planID = &id
			}
			if err == gorm.ErrRecordNotFound {
				emp := models.Employee{
					PersonnelNo:      se.PersonnelNo,
					FullName:         se.FullName,
					NationalID:       se.NationalID,
					EmploymentStatus: se.EmploymentStatus,
					HireDate:         se.HireDate,
					Department:       se.Department,
					PlanID:           planID,
				}
				if err := tx.Create(&emp).Error; err != nil {
					return err
				}
				resp.Created++
				continue
			}
			if err != nil {
				return err
			}
			existing.FullName = se.FullName
			existing.NationalID = se.NationalID
			existing.EmploymentStatus = se.EmploymentStatus
			existing.HireDate = se.HireDate
			existing.Department = se.Department
			if planID != nil {
				existing.PlanID = planID
			}
			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
			resp.Updated++
		}
		return nil
	})
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

type claimStatusResponse struct {
	ID            string   `json:"id"`
	Status        string   `json:"status"`
	PayableAmount *float64 `json:"payable_amount"`
}

func (h *handlers) integrationClaimStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var claim models.Claim
	if err := h.d.DB.WithContext(r.Context()).First(&claim, "id = ?", id).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, claimStatusResponse{
		ID:            claim.ID.String(),
		Status:        string(claim.Status),
		PayableAmount: claim.PayableAmount,
	})
}
