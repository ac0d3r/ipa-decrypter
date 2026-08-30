package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ac0d3r/ipa-decrypter/internal/services"
)

const sessionCookie = "session"

func (s *Server) registerAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/v1/auth/me", s.requireAuth(s.handleMe))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
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
		if errors.Is(err, services.ErrInvalidCredentials) {
			writeErr(w, errStatus(http.StatusUnauthorized, "invalid username or password"))
			return
		}
		writeErr(w, err)
		return
	}
	setSessionCookie(w, result.Token, result.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{
		"username":   result.Username,
		"is_admin":   result.IsAdmin,
		"expires_at": result.ExpiresAt,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	_ = s.Auth.Logout(sessionToken(r))
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r)
	if !ok {
		writeErr(w, errStatus(http.StatusUnauthorized, "unauthorized"))
		return
	}
	payload := map[string]any{
		"username": u.Username,
		"is_admin": u.IsAdmin,
	}
	if u.SessionExpiresAt != nil {
		payload["expires_at"] = *u.SessionExpiresAt
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := s.Auth.Validate(sessionToken(r))
		if err != nil {
			writeErr(w, errStatus(http.StatusUnauthorized, "unauthorized"))
			return
		}
		next(w, r.WithContext(withUser(r.Context(), u)))
	}
}

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := s.Auth.ValidateAdmin(sessionToken(r))
		if err != nil {
			if errors.Is(err, services.ErrForbidden) {
				writeErr(w, errStatus(http.StatusForbidden, "forbidden"))
				return
			}
			writeErr(w, errStatus(http.StatusUnauthorized, "unauthorized"))
			return
		}
		next(w, r.WithContext(withUser(r.Context(), u)))
	}
}

func sessionToken(r *http.Request) string {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		return c.Value
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
