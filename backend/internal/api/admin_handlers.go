package api

import (
	"net/http"

	"insurance-module/internal/auth"
	"insurance-module/internal/models"
)

func (h *handlers) listUsers(w http.ResponseWriter, r *http.Request) {
	var out []models.User
	if err := h.d.DB.WithContext(r.Context()).Order("username").Find(&out).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type createUserRequest struct {
	Username   string      `json:"username"`
	Password   string      `json:"password"`
	FullName   string      `json:"full_name"`
	Role       models.Role `json:"role"`
	EmployeeID *string     `json:"employee_id"`
}

func (h *handlers) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash password")
		return
	}
	u := models.User{
		Username:     req.Username,
		PasswordHash: hash,
		FullName:     req.FullName,
		Role:         req.Role,
		IsActive:     true,
	}
	if req.EmployeeID != nil {
		empID, err := parseUUIDField(*req.EmployeeID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid employee_id")
			return
		}
		u.EmployeeID = &empID
	}
	if err := h.d.DB.WithContext(r.Context()).Create(&u).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

type updateUserRequest struct {
	Role     *models.Role `json:"role"`
	IsActive *bool        `json:"is_active"`
	Password *string      `json:"password"`
}

func (h *handlers) updateUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var u models.User
	if err := h.d.DB.WithContext(r.Context()).First(&u, "id = ?", id).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	if req.Role != nil {
		u.Role = *req.Role
	}
	if req.IsActive != nil {
		u.IsActive = *req.IsActive
	}
	if req.Password != nil && *req.Password != "" {
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not hash password")
			return
		}
		u.PasswordHash = hash
	}
	if err := h.d.DB.WithContext(r.Context()).Save(&u).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}
