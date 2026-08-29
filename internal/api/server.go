package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ac0d3r/ipa-decrypter/internal/services"
)

type Server struct {
	*services.Registry
	WebDir string
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /health", s.handleHealth)

	s.registerPublicRoutes(mux)

	// Admin / backend management
	s.registerAdminRoutes(mux, "/api/v1/admin")
	// Legacy paths (same handlers)
	s.registerAdminRoutes(mux, "/api/v1")

	if s.WebDir != "" {
		fs := http.FileServer(http.Dir(s.WebDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			path := filepath.Join(s.WebDir, filepath.Clean(r.URL.Path))
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				fs.ServeHTTP(w, r)
				return
			}
			http.ServeFile(w, r, filepath.Join(s.WebDir, "index.html"))
		})
	}

	return withCORS(mux)
}

func (s *Server) registerAdminRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("GET "+prefix+"/accounts", s.handleListAccounts)
	mux.HandleFunc("POST "+prefix+"/accounts", s.handleCreateAccount)
	mux.HandleFunc("GET "+prefix+"/accounts/{id}", s.handleGetAccount)
	mux.HandleFunc("POST "+prefix+"/accounts/{id}/login", s.handleLoginAccount)
	mux.HandleFunc("POST "+prefix+"/accounts/{id}/refresh", s.handleRefreshAccount)
	mux.HandleFunc("POST "+prefix+"/accounts/{id}/revoke", s.handleRevokeAccount)
	mux.HandleFunc("DELETE "+prefix+"/accounts/{id}", s.handleDeleteAccount)

	mux.HandleFunc("GET "+prefix+"/devices", s.handleListDevices)
	mux.HandleFunc("POST "+prefix+"/devices", s.handleCreateDevice)
	mux.HandleFunc("GET "+prefix+"/devices/{id}", s.handleGetDevice)
	mux.HandleFunc("PUT "+prefix+"/devices/{id}", s.handleUpdateDevice)
	mux.HandleFunc("POST "+prefix+"/devices/{id}/health", s.handleHealthDevice)
	mux.HandleFunc("DELETE "+prefix+"/devices/{id}", s.handleDeleteDevice)

	mux.HandleFunc("GET "+prefix+"/tasks", s.handleAdminTasks)
}

func (s *Server) handleAdminTasks(w http.ResponseWriter, r *http.Request) {
	items, err := s.Tasks.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "ipa-decrypter"})
}

func (s *Server) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	items, err := s.Accounts.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var in services.AccountCreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, err)
		return
	}
	item, err := s.Accounts.Create(in)
	if err != nil {
		writeAccountErr(w, http.StatusCreated, item, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	item, err := s.Accounts.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleLoginAccount(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AuthCode string `json:"auth_code"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	item, err := s.Accounts.Login(r.PathValue("id"), body.AuthCode)
	if err != nil {
		writeAccountErr(w, http.StatusOK, item, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func writeAccountErr(w http.ResponseWriter, okStatus int, item any, err error) {
	if le, ok := err.(*services.AccountLoginError); ok {
		payload := map[string]any{
			"error":     le.Message,
			"needs_2fa": le.Needs2FA,
			"account":   le.Account,
		}
		code := http.StatusBadRequest
		if le.Needs2FA {
			code = http.StatusUnauthorized
		}
		writeJSON(w, code, payload)
		return
	}
	_ = okStatus
	_ = item
	writeErr(w, err)
}

func (s *Server) handleRefreshAccount(w http.ResponseWriter, r *http.Request) {
	item, err := s.Accounts.Refresh(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleRevokeAccount(w http.ResponseWriter, r *http.Request) {
	item, err := s.Accounts.Revoke(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	if err := s.Accounts.Delete(r.PathValue("id")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	items, err := s.Devices.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var in services.DeviceInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, err)
		return
	}
	item, err := s.Devices.Create(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	item, err := s.Devices.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	var in services.DeviceInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, err)
		return
	}
	item, err := s.Devices.Update(r.PathValue("id"), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleHealthDevice(w http.ResponseWriter, r *http.Request) {
	item, err := s.Devices.Health(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	if err := s.Devices.Delete(r.PathValue("id")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func serveFile(w http.ResponseWriter, r *http.Request, path, name string) {
	f, err := os.Open(path)
	if err != nil {
		writeErr(w, err)
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	http.ServeContent(w, r, name, stat.ModTime(), f)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

type apiError struct {
	status  int
	message string
}

func (e apiError) Error() string { return e.message }

func errStatus(code int, msg string) error {
	return apiError{status: code, message: msg}
}

func writeErr(w http.ResponseWriter, err error) {
	code := http.StatusBadRequest
	msg := err.Error()
	if ae, ok := err.(apiError); ok {
		code = ae.status
		msg = ae.message
	}
	if strings.Contains(msg, "not found") {
		code = http.StatusNotFound
	}
	writeJSON(w, code, map[string]string{"error": msg})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var _ = io.Discard
