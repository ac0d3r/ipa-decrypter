package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ac0d3r/ipa-decrypter/internal/services"
	"github.com/ac0d3r/ipa-decrypter/internal/version"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

type Server struct {
	*services.Registry
	WebDir  string
	limiter *rateLimiter
}

func (s *Server) Engine() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	s.limiter = newRateLimiter()

	r.GET("/health", s.handleHealth)

	s.registerPublicRoutes(r)
	s.registerAuthRoutes(r)

	admin := r.Group("/api/v1/admin", s.requireAdmin())
	s.registerAdminRoutes(admin)
	s.registerUserRoutes(admin)
	s.registerVoucherRoutes(admin)

	r.NoRoute(s.handleNoRoute)

	return r
}

func (s *Server) handleNoRoute(c *gin.Context) {
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		writeErr(c, errStatus(http.StatusNotFound, "not found"))
		return
	}
	if s.WebDir == "" {
		c.Status(http.StatusNotFound)
		return
	}
	rel := strings.TrimPrefix(filepath.Clean(c.Request.URL.Path), "/")
	path := filepath.Join(s.WebDir, rel)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		c.File(path)
		return
	}
	c.File(filepath.Join(s.WebDir, "index.html"))
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "ipa-decrypter", "version": version.Version})
}
