package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ac0d3r/ipa-decrypter/internal/services"
)

func (s *Server) registerUserRoutes(admin *gin.RouterGroup) {
	admin.GET("/users", s.handleListUsers)
	admin.POST("/users", s.handleCreateUser)
	admin.PUT("/users/:id", s.handleUpdateUser)
	admin.DELETE("/users/:id", s.handleDeleteUser)
}

func (s *Server) handleListUsers(c *gin.Context) {
	items, err := s.Auth.ListUsers()
	if err != nil {
		writeErr(c, err)
		return
	}
	writeJSON(c, http.StatusOK, items)
}

func (s *Server) handleCreateUser(c *gin.Context) {
	var in services.UserCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeErr(c, errStatus(http.StatusBadRequest, "invalid request body"))
		return
	}
	item, err := s.Auth.CreateUser(in)
	if err != nil {
		writeErr(c, err)
		return
	}
	writeJSON(c, http.StatusCreated, item)
}

func (s *Server) handleUpdateUser(c *gin.Context) {
	var in services.UserUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeErr(c, errStatus(http.StatusBadRequest, "invalid request body"))
		return
	}
	item, err := s.Auth.UpdateUser(c.Param("id"), in)
	if err != nil {
		writeUserErr(c, err)
		return
	}
	writeJSON(c, http.StatusOK, item)
}

func (s *Server) handleDeleteUser(c *gin.Context) {
	if err := s.Auth.DeleteUser(c.Param("id"), currentUser(c).ID); err != nil {
		writeUserErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func writeUserErr(c *gin.Context, err error) {
	if errors.Is(err, services.ErrNotFound) {
		writeErr(c, errStatus(http.StatusNotFound, "not found"))
		return
	}
	writeErr(c, err)
}
