package api

import (
	"net/http"

	appmw "insurance-module/internal/api/middleware"
	"insurance-module/internal/audit"
	"insurance-module/internal/auth"
	"insurance-module/internal/models"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

func (h *handlers) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var user models.User
	if err := h.d.DB.WithContext(r.Context()).First(&user, "username = ?", req.Username).Error; err != nil {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if !user.IsActive || !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := auth.GenerateToken(h.d.Cfg.JWTSecret, h.d.Cfg.JWTTTL, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue token")
		return
	}

	_ = h.d.Audit.Log(r.Context(), nil, audit.Entry{
		EntityType:    "user",
		EntityID:      user.ID.String(),
		Action:        "login",
		ActorUserID:   &user.ID,
		ActorUsername: user.Username,
	})
	writeJSON(w, http.StatusOK, loginResponse{Token: token, User: user})
}

func (h *handlers) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := appmw.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var user models.User
	if err := h.d.DB.WithContext(r.Context()).First(&user, "id = ?", claims.UserID).Error; err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}
