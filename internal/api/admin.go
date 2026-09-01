package api

import (
	"net/http"

	"github.com/ac0d3r/ipa-decrypter/internal/services"
	"github.com/gin-gonic/gin"
)

func (s *Server) registerAdminRoutes(admin *gin.RouterGroup) {
	admin.GET("/accounts", s.handleListAccounts)
	admin.POST("/accounts", s.handleCreateAccount)
	admin.GET("/accounts/:id", s.handleGetAccount)
	admin.POST("/accounts/:id/login", s.handleLoginAccount)
	admin.POST("/accounts/:id/refresh", s.handleRefreshAccount)
	admin.POST("/accounts/:id/revoke", s.handleRevokeAccount)
	admin.DELETE("/accounts/:id", s.handleDeleteAccount)

	admin.GET("/devices", s.handleListDevices)
	admin.POST("/devices", s.handleCreateDevice)
	admin.GET("/devices/:id", s.handleGetDevice)
	admin.PUT("/devices/:id", s.handleUpdateDevice)
	admin.POST("/devices/:id/health", s.handleHealthDevice)
	admin.DELETE("/devices/:id", s.handleDeleteDevice)

	admin.GET("/tasks", s.handleAdminTasks)
}

func (s *Server) handleAdminTasks(c *gin.Context) {
	items, err := s.Tasks.List()
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleListAccounts(c *gin.Context) {
	items, err := s.Accounts.List()
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleCreateAccount(c *gin.Context) {
	var in services.AccountCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeErr(c, err)
		return
	}
	item, err := s.Accounts.Create(in)
	if err != nil {
		writeAccountErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (s *Server) handleGetAccount(c *gin.Context) {
	item, err := s.Accounts.Get(c.Param("id"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleLoginAccount(c *gin.Context) {
	var body struct {
		AuthCode string `json:"auth_code"`
	}
	_ = c.ShouldBindJSON(&body)
	item, err := s.Accounts.Login(c.Param("id"), body.AuthCode)
	if err != nil {
		writeAccountErr(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleRefreshAccount(c *gin.Context) {
	item, err := s.Accounts.Refresh(c.Param("id"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleRevokeAccount(c *gin.Context) {
	item, err := s.Accounts.Revoke(c.Param("id"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleDeleteAccount(c *gin.Context) {
	if err := s.Accounts.Delete(c.Param("id")); err != nil {
		writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) handleListDevices(c *gin.Context) {
	items, err := s.Devices.List()
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleCreateDevice(c *gin.Context) {
	var in services.DeviceInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeErr(c, err)
		return
	}
	item, err := s.Devices.Create(in)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (s *Server) handleGetDevice(c *gin.Context) {
	item, err := s.Devices.Get(c.Param("id"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleUpdateDevice(c *gin.Context) {
	var in services.DeviceInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeErr(c, err)
		return
	}
	item, err := s.Devices.Update(c.Param("id"), in)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleHealthDevice(c *gin.Context) {
	item, err := s.Devices.Health(c.Param("id"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleDeleteDevice(c *gin.Context) {
	if err := s.Devices.Delete(c.Param("id")); err != nil {
		writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
