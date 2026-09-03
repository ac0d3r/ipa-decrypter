package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ac0d3r/ipa-decrypter/internal/services"
)

func (s *Server) registerVoucherRoutes(admin *gin.RouterGroup) {
	admin.GET("/vouchers", s.handleListVouchers)
	admin.POST("/vouchers", s.handleCreateVouchers)
	admin.DELETE("/vouchers/:id", s.handleDeleteVoucher)
}

func (s *Server) handleListVouchers(c *gin.Context) {
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	page, err := s.Auth.ListVouchers(services.VoucherListQuery{
		Status: c.Query("status"),
		UserID: c.Query("user"),
		Offset: offset,
		Limit:  limit,
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (s *Server) handleCreateVouchers(c *gin.Context) {
	var in services.VoucherCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeErr(c, errStatus(http.StatusBadRequest, "invalid request body"))
		return
	}
	out, err := s.Auth.CreateVouchers(in)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (s *Server) handleDeleteVoucher(c *gin.Context) {
	if err := s.Auth.DeleteVoucher(c.Param("id")); err != nil {
		writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
