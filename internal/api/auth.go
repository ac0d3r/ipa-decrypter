package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ac0d3r/ipa-decrypter/internal/services"
)

const adminSessionCookie = "admin_session"

func (s *Server) registerAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/admin/login", s.handleAdminLogin)
	mux.HandleFunc("POST /api/v1/admin/logout", s.handleAdminLogout)
	mux.HandleFunc("GET /api/v1/admin/me", s.requireAdmin(s.handleAdminMe))
}

func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, errStatus(http.StatusBadRequest, "invalid request body"))
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	if body.Username == "" || body.Password == "" {
		writeErr(w, errStatus(http.StatusBadRequest, "username and password required"))
		return
	}
	result, err := s.Auth.Login(body.Username, body.Password)
	if err != nil {
		if err == services.ErrInvalidCredentials {
			writeErr(w, errStatus(http.StatusUnauthorized, "invalid username or password"))
			return
		}
		writeErr(w, err)
		return
	}
	setAdminSessionCookie(w, result.Token, result.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{
		"username":   result.Username,
		"expires_at": result.ExpiresAt,
	})
}

func (s *Server) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	token := adminSessionToken(r)
	_ = s.Auth.Logout(token)
	clearAdminSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminMe(w http.ResponseWriter, r *http.Request) {
	u, ok := adminUserFromContext(r)
	if !ok {
		writeErr(w, errStatus(http.StatusUnauthorized, "unauthorized"))
		return
	}
	payload := map[string]any{"username": u.Username}
	if u.SessionExpiresAt != nil {
		payload["expires_at"] = *u.SessionExpiresAt
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := adminSessionToken(r)
		u, err := s.Auth.Validate(token)
		if err != nil {
			writeErr(w, errStatus(http.StatusUnauthorized, "unauthorized"))
			return
		}
		next(w, r.WithContext(withAdminUser(r.Context(), u)))
	}
}

func adminSessionToken(r *http.Request) string {
	if c, err := r.Cookie(adminSessionCookie); err == nil && c.Value != "" {
		return c.Value
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func setAdminSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
	})
}

func clearAdminSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
