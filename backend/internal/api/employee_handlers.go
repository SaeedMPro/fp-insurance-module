package api

import (
	"net/http"
	"time"

	"gorm.io/gorm"

	"insurance-module/internal/models"
)

type employeeListResponse struct {
	Items []models.Employee `json:"items"`
	Total int64             `json:"total"`
}

func (h *handlers) listEmployees(w http.ResponseWriter, r *http.Request) {
	page, pageSize := pagination(r)
	q := h.d.DB.WithContext(r.Context()).Model(&models.Employee{})
	if search := r.URL.Query().Get("q"); search != "" {
		like := "%" + search + "%"
		q = q.Where("full_name ILIKE ? OR personnel_no ILIKE ?", like, like)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		mapDomainError(w, err)
		return
	}

	var items []models.Employee
	if err := q.Order("full_name").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, employeeListResponse{Items: items, Total: total})
}

type createEmployeeRequest struct {
	PersonnelNo string    `json:"personnel_no"`
	FullName    string    `json:"full_name"`
	NationalID  string    `json:"national_id"`
	HireDate    time.Time `json:"hire_date"`
	Department  string    `json:"department"`
	PlanID      *string   `json:"plan_id"`
}

func (h *handlers) createEmployee(w http.ResponseWriter, r *http.Request) {
	var req createEmployeeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	e := models.Employee{
		PersonnelNo:      req.PersonnelNo,
		FullName:         req.FullName,
		NationalID:       req.NationalID,
		EmploymentStatus: models.EmploymentActive,
		HireDate:         req.HireDate,
		Department:       req.Department,
	}
	if req.PlanID != nil {
		planID, err := parseUUIDField(*req.PlanID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid plan_id")
			return
		}
		e.PlanID = &planID
	}
	if err := h.d.DB.WithContext(r.Context()).Create(&e).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *handlers) getEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if !h.authorizeEmployeeAccess(r, id) {
		writeError(w, http.StatusForbidden, "not permitted to view this employee")
		return
	}
	var e models.Employee
	if err := h.d.DB.WithContext(r.Context()).First(&e, "id = ?", id).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

type updateEmployeeRequest struct {
	EmploymentStatus *models.EmploymentStatus `json:"employment_status"`
	PlanID           *string                  `json:"plan_id"`
	Department       *string                  `json:"department"`
	FullName         *string                  `json:"full_name"`
}

func (h *handlers) updateEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateEmployeeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var e models.Employee
	if err := h.d.DB.WithContext(r.Context()).First(&e, "id = ?", id).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	if req.EmploymentStatus != nil {
		e.EmploymentStatus = *req.EmploymentStatus
	}
	if req.Department != nil {
		e.Department = *req.Department
	}
	if req.FullName != nil {
		e.FullName = *req.FullName
	}
	if req.PlanID != nil {
		planID, err := parseUUIDField(*req.PlanID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid plan_id")
			return
		}
		e.PlanID = &planID
	}
	if err := h.d.DB.WithContext(r.Context()).Save(&e).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *handlers) listDependents(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if !h.authorizeEmployeeAccess(r, id) {
		writeError(w, http.StatusForbidden, "not permitted to view this employee")
		return
	}
	var deps []models.Dependent
	if err := h.d.DB.WithContext(r.Context()).Where("employee_id = ?", id).Order("full_name").Find(&deps).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, deps)
}

type createDependentRequest struct {
	FullName   string          `json:"full_name"`
	Relation   models.Relation `json:"relation"`
	NationalID string          `json:"national_id"`
	BirthDate  *time.Time      `json:"birth_date"`
}

func (h *handlers) createDependent(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req createDependentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	dep := models.Dependent{
		EmployeeID: id,
		FullName:   req.FullName,
		Relation:   req.Relation,
		NationalID: req.NationalID,
		BirthDate:  req.BirthDate,
	}
	if err := h.d.DB.WithContext(r.Context()).Create(&dep).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dep)
}

func (h *handlers) remainingCaps(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if !h.authorizeEmployeeAccess(r, id) {
		writeError(w, http.StatusForbidden, "not permitted to view this employee")
		return
	}
	var e models.Employee
	if err := h.d.DB.WithContext(r.Context()).First(&e, "id = ?", id).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	if e.PlanID == nil {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}
	caps, err := h.d.Rules.RemainingCaps(r.Context(), e.ID, *e.PlanID, time.Now())
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, caps)
}
