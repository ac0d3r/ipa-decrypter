package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ac0d3r/ipa-decrypter/internal/services"
)

type apiError struct {
	status  int
	message string
}

func (e apiError) Error() string { return e.message }

func errStatus(code int, msg string) error {
	return apiError{status: code, message: msg}
}

func writeJSON(c *gin.Context, code int, v any) {
	c.JSON(code, v)
}

func writeErr(c *gin.Context, err error) {
	code := http.StatusBadRequest
	msg := err.Error()
	if ae, ok := err.(apiError); ok {
		code = ae.status
		msg = ae.message
	}
	if strings.Contains(strings.ToLower(msg), "not found") {
		code = http.StatusNotFound
		msg = "not found"
	}
	c.JSON(code, gin.H{"error": msg})
}

func writeAccountErr(c *gin.Context, err error) {
	if le, ok := err.(*services.AccountLoginError); ok {
		payload := gin.H{
			"error":     le.Message,
			"needs_2fa": le.Needs2FA,
			"account":   le.Account,
		}
		code := http.StatusBadRequest
		if le.Needs2FA {
			code = http.StatusUnauthorized
		}
		c.JSON(code, payload)
		return
	}
	writeErr(c, err)
}
