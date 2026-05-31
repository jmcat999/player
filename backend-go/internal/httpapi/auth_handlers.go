package httpapi

import (
	"encoding/json"
	"net/http"

	"player-stats-backend-go/internal/auth"
)

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是有效 JSON")
		return
	}
	response, err := s.authService.Login(r.Context(), req)
	if err != nil {
		status, message := auth.ErrorStatus(err)
		writeError(w, status, message)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	principal, ok := PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "请先登录管理员账号")
		return
	}
	writeJSON(w, http.StatusOK, principal)
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	principal, ok := PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "请先登录管理员账号")
		return
	}
	var req auth.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是有效 JSON")
		return
	}
	if err := s.authService.ChangePassword(r.Context(), principal, req); err != nil {
		status, message := auth.ErrorStatus(err)
		writeError(w, status, message)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{})
}
