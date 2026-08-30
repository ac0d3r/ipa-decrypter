package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ac0d3r/ipa-decrypter/internal/services"
)

func (s *Server) registerUserAdminRoutes(mux *http.ServeMux) {
	prefix := "/api/v1/admin/users"
	mux.HandleFunc("GET "+prefix, s.requireAdmin(s.handleListUsers))
	mux.HandleFunc("POST "+prefix, s.requireAdmin(s.handleCreateUser))
	mux.HandleFunc("PUT "+prefix+"/{id}", s.requireAdmin(s.handleUpdateUser))
	mux.HandleFunc("DELETE "+prefix+"/{id}", s.requireAdmin(s.handleDeleteUser))
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	items, err := s.Auth.ListUsers()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var in services.UserCreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, errStatus(http.StatusBadRequest, "invalid request body"))
		return
	}
	item, err := s.Auth.CreateUser(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	var in services.UserUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, errStatus(http.StatusBadRequest, "invalid request body"))
		return
	}
	item, err := s.Auth.UpdateUser(r.PathValue("id"), in)
	if err != nil {
		writeUserErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	actor, ok := userFromContext(r)
	if !ok {
		writeErr(w, errStatus(http.StatusUnauthorized, "unauthorized"))
		return
	}
	if err := s.Auth.DeleteUser(r.PathValue("id"), actor.ID); err != nil {
		writeUserErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeUserErr(w http.ResponseWriter, err error) {
	if errors.Is(err, services.ErrNotFound) {
		writeErr(w, errStatus(http.StatusNotFound, "not found"))
		return
	}
	writeErr(w, err)
}
