package http

import (
	"net/http"

	"insurance-module/internal/domain"
	"insurance-module/internal/service/users"
)

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	items, err := s.users.List(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(items, toUserDTO))
}

type createUserRequest struct {
	Username   string      `json:"username"`
	Password   string      `json:"password"`
	FullName   string      `json:"full_name"`
	Role       domain.Role `json:"role"`
	EmployeeID *string     `json:"employee_id"`
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	employeeID, err := parseOptionalUUID(req.EmployeeID, "employee_id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	user, err := s.users.Create(r.Context(), users.CreateInput{
		Username:   req.Username,
		Password:   req.Password,
		FullName:   req.FullName,
		Role:       req.Role,
		EmployeeID: employeeID,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toUserDTO(user))
}

type updateUserRequest struct {
	Role     *domain.Role `json:"role"`
	IsActive *bool        `json:"is_active"`
	Password *string      `json:"password"`
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	var req updateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	user, err := s.users.Update(r.Context(), id, users.UpdateInput{
		Role:     req.Role,
		IsActive: req.IsActive,
		Password: req.Password,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserDTO(user))
}
