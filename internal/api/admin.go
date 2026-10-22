package api

import (
	"bytes"
	"encoding/hex"
	"io"
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
	admin.POST("/accounts/:id/credential", s.handleUploadCredential)
	admin.DELETE("/accounts/:id/credential", s.handleClearCredential)
	admin.DELETE("/accounts/:id", s.handleDeleteAccount)

	admin.GET("/devices", s.handleListDevices)
	admin.POST("/devices", s.handleCreateDevice)
	admin.GET("/devices/:id", s.handleGetDevice)
	admin.PUT("/devices/:id", s.handleUpdateDevice)
	admin.POST("/devices/:id/health", s.handleHealthDevice)
	admin.DELETE("/devices/:id", s.handleDeleteDevice)

	admin.GET("/tasks", s.handleAdminTasks)
	admin.DELETE("/tasks/:id", s.handleDeleteTask)
}

func (s *Server) handleAdminTasks(c *gin.Context) {
	items, err := s.Tasks.List()
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleDeleteTask(c *gin.Context) {
	if err := s.Tasks.Delete(c.Param("id")); err != nil {
		writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
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

// handleUploadCredential stores a 16-byte FairPlay credential.bin for the
// account. Accepts a multipart file field ("credential"/"file"), a raw
// 16-byte body, or a 32-char hex body.
func (s *Server) handleUploadCredential(c *gin.Context) {
	var (
		data []byte
		err  error
	)
	if file, fhErr := c.FormFile("credential"); fhErr == nil {
		fh, err := file.Open()
		if err != nil {
			writeErr(c, err)
			return
		}
		defer fh.Close()
		data, err = io.ReadAll(io.LimitReader(fh, 4096))
		if err != nil {
			writeErr(c, err)
			return
		}
	} else if file, fhErr := c.FormFile("file"); fhErr == nil {
		fh, err := file.Open()
		if err != nil {
			writeErr(c, err)
			return
		}
		defer fh.Close()
		data, err = io.ReadAll(io.LimitReader(fh, 4096))
		if err != nil {
			writeErr(c, err)
			return
		}
	} else {
		data, err = io.ReadAll(io.LimitReader(c.Request.Body, 4096))
		if err != nil {
			writeErr(c, err)
			return
		}
	}
	data = normalizeCredential(data)
	item, err := s.Accounts.SetCredential(c.Param("id"), data)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

// normalizeCredential accepts raw 16 bytes or a 32-char hex string.
func normalizeCredential(data []byte) []byte {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 32 {
		if decoded, err := hex.DecodeString(string(trimmed)); err == nil {
			return decoded
		}
	}
	return trimmed
}

func (s *Server) handleClearCredential(c *gin.Context) {
	item, err := s.Accounts.ClearCredential(c.Param("id"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
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
