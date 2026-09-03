package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ac0d3r/ipa-decrypter/internal/db"
	"github.com/ac0d3r/ipa-decrypter/internal/services"
	"github.com/gin-gonic/gin"
)

const (
	sessionCookie = "session"
	userKey       = "user"
	userIDKey     = "user_id"
)

func setUser(c *gin.Context, u db.User) {
	c.Set(userKey, u)
	c.Set(userIDKey, u.ID)
}

func currentUser(c *gin.Context) db.User {
	v, _ := c.Get(userKey)
	u, _ := v.(db.User)
	return u
}

func currentUserID(c *gin.Context) string {
	return c.GetString(userIDKey)
}

func sessionToken(c *gin.Context) string {
	if token, err := c.Cookie(sessionCookie); err == nil && token != "" {
		return token
	}
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func (s *Server) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		u, err := s.Auth.Validate(sessionToken(c))
		if err != nil {
			writeErr(c, errStatus(http.StatusUnauthorized, "unauthorized"))
			c.Abort()
			return
		}
		setUser(c, u)
		c.Next()
	}
}

func (s *Server) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		u, err := s.Auth.ValidateAdmin(sessionToken(c))
		if err != nil {
			if errors.Is(err, services.ErrForbidden) {
				writeErr(c, errStatus(http.StatusForbidden, "forbidden"))
			} else {
				writeErr(c, errStatus(http.StatusUnauthorized, "unauthorized"))
			}
			c.Abort()
			return
		}
		setUser(c, u)
		c.Next()
	}
}

func (s *Server) quotaMiddleware(qtype string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := currentUser(c)
		if err := u.CheckQuota(qtype); err != nil {
			writeErr(c, err)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *Server) registerAuthRoutes(r gin.IRouter) {
	auth := r.Group("/api/v1/auth")
	auth.POST("/login", s.handleLogin)
	auth.POST("/register", s.rateLimit(registerRateMax), s.handleRegister)
	auth.POST("/logout", s.handleLogout)
	auth.GET("/me", s.requireAuth(), s.handleMe)
	auth.POST("/redeem", s.requireAuth(), s.handleRedeem)
}

func (s *Server) handleLogin(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, errStatus(http.StatusBadRequest, "invalid request body"))
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	if body.Username == "" || body.Password == "" {
		writeErr(c, errStatus(http.StatusBadRequest, "username and password required"))
		return
	}
	result, err := s.Auth.Login(body.Username, body.Password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			writeErr(c, errStatus(http.StatusUnauthorized, "invalid username or password"))
			return
		}
		writeErr(c, err)
		return
	}
	setSessionCookie(c, result.Token, result.ExpiresAt)

	c.JSON(http.StatusOK, gin.H{
		"username":   result.Username,
		"is_admin":   result.IsAdmin,
		"expires_at": result.ExpiresAt,
		"quota":      result.Quota,
	})
}

func (s *Server) handleRegister(c *gin.Context) {
	var in services.RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeErr(c, errStatus(http.StatusBadRequest, "invalid request body"))
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	in.Code = strings.TrimSpace(in.Code)
	if in.Username == "" || in.Password == "" || in.Code == "" {
		writeErr(c, errStatus(http.StatusBadRequest, "username, password and code required"))
		return
	}
	result, err := s.Auth.Register(in)
	if err != nil {
		writeErr(c, err)
		return
	}
	setSessionCookie(c, result.Token, result.ExpiresAt)
	c.JSON(http.StatusCreated, gin.H{
		"username":   result.Username,
		"is_admin":   result.IsAdmin,
		"expires_at": result.ExpiresAt,
		"quota":      result.Quota,
	})
}

func (s *Server) handleRedeem(c *gin.Context) {
	var body struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeErr(c, errStatus(http.StatusBadRequest, "invalid request body"))
		return
	}
	u, added, err := s.Auth.RedeemVoucher(currentUserID(c), body.Code)
	if err != nil {
		writeErr(c, err)
		return
	}
	payload := gin.H{
		"username": u.Username,
		"is_admin": u.IsAdmin,
		"quota":    u.Quota(),
		"added":    added,
	}
	c.JSON(http.StatusOK, payload)
}

func (s *Server) handleLogout(c *gin.Context) {
	_ = s.Auth.Logout(sessionToken(c))
	clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

func (s *Server) handleMe(c *gin.Context) {
	u := currentUser(c)
	payload := gin.H{
		"username": u.Username,
		"is_admin": u.IsAdmin,
	}
	if u.SessionExpiresAt != nil {
		payload["expires_at"] = *u.SessionExpiresAt
	}
	if !u.IsAdmin {
		payload["quota"] = u.Quota()
	}
	c.JSON(http.StatusOK, payload)
}

func setSessionCookie(c *gin.Context, token string, expires time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
	})
}

func clearSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
