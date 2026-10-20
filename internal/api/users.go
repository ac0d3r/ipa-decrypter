package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ac0d3r/ipa-decrypter/internal/db"
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
	out := make([]gin.H, 0, len(items))
	for i := range items {
		out = append(out, userJSON(items[i]))
	}
	c.JSON(http.StatusOK, out)
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
	c.JSON(http.StatusCreated, userJSON(item))
}

func (s *Server) handleUpdateUser(c *gin.Context) {
	var in services.UserUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeErr(c, errStatus(http.StatusBadRequest, "invalid request body"))
		return
	}
	item, err := s.Auth.UpdateUser(c.Param("id"), in)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, userJSON(item))
}

func (s *Server) handleDeleteUser(c *gin.Context) {
	if err := s.Auth.DeleteUser(c.Param("id"), currentUserID(c)); err != nil {
		writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func userJSON(u db.User) gin.H {
	return gin.H{
		"id":         u.ID,
		"username":   u.Username,
		"is_admin":   u.IsAdmin,
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
	}
}
