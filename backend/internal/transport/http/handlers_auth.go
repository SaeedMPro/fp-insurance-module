package http

import "net/http"

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string  `json:"token"`
	User  userDTO `json:"user"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	user, token, err := s.users.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{Token: token, User: toUserDTO(user)})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	actor, ok := mustActor(w, r)
	if !ok {
		return
	}
	user, err := s.users.Get(r.Context(), actor.UserID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserDTO(user))
}
