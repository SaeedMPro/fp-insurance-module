package http

import (
	"net/http"
	"time"

	"insurance-module/internal/domain"
	"insurance-module/internal/service/employees"
)

func (s *Server) handleListEmployees(w http.ResponseWriter, r *http.Request) {
	items, total, err := s.employees.List(r.Context(), domain.EmployeeFilter{
		Query: r.URL.Query().Get("q"),
		Page:  pageParams(r),
	})
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, listDTO[employeeDTO]{Items: mapSlice(items, toEmployeeDTO), Total: total})
}

type createEmployeeRequest struct {
	PersonnelNo string    `json:"personnel_no"`
	FullName    string    `json:"full_name"`
	NationalID  string    `json:"national_id"`
	HireDate    time.Time `json:"hire_date"`
	Department  string    `json:"department"`
	PlanID      *string   `json:"plan_id"`
}

func (s *Server) handleCreateEmployee(w http.ResponseWriter, r *http.Request) {
	var req createEmployeeRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	planID, err := parseOptionalUUID(req.PlanID, "plan_id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	employee, err := s.employees.Create(r.Context(), employees.CreateInput{
		PersonnelNo: req.PersonnelNo,
		FullName:    req.FullName,
		NationalID:  req.NationalID,
		HireDate:    req.HireDate,
		Department:  req.Department,
		PlanID:      planID,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toEmployeeDTO(employee))
}

func (s *Server) handleGetEmployee(w http.ResponseWriter, r *http.Request) {
	actor, ok := mustActor(w, r)
	if !ok {
		return
	}
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	employee, err := s.employees.Get(r.Context(), actor, id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toEmployeeDTO(employee))
}

type updateEmployeeRequest struct {
	EmploymentStatus *domain.EmploymentStatus `json:"employment_status"`
	PlanID           *string                  `json:"plan_id"`
	Department       *string                  `json:"department"`
	FullName         *string                  `json:"full_name"`
}

func (s *Server) handleUpdateEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	var req updateEmployeeRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	planID, err := parseOptionalUUID(req.PlanID, "plan_id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	employee, err := s.employees.Update(r.Context(), id, employees.UpdateInput{
		EmploymentStatus: req.EmploymentStatus,
		PlanID:           planID,
		Department:       req.Department,
		FullName:         req.FullName,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toEmployeeDTO(employee))
}

func (s *Server) handleListDependents(w http.ResponseWriter, r *http.Request) {
	actor, ok := mustActor(w, r)
	if !ok {
		return
	}
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	deps, err := s.employees.ListDependents(r.Context(), actor, id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(deps, toDependentDTO))
}

type createDependentRequest struct {
	FullName   string          `json:"full_name"`
	Relation   domain.Relation `json:"relation"`
	NationalID string          `json:"national_id"`
	BirthDate  *time.Time      `json:"birth_date"`
}

func (s *Server) handleCreateDependent(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	var req createDependentRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	dep, err := s.employees.CreateDependent(r.Context(), id, employees.CreateDependentInput{
		FullName:   req.FullName,
		Relation:   req.Relation,
		NationalID: req.NationalID,
		BirthDate:  req.BirthDate,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toDependentDTO(dep))
}

func (s *Server) handleRemainingCaps(w http.ResponseWriter, r *http.Request) {
	actor, ok := mustActor(w, r)
	if !ok {
		return
	}
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	caps, err := s.employees.RemainingCaps(r.Context(), actor, id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(caps, toRemainingCapDTO))
}
