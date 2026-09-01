package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ac0d3r/ipa-decrypter/internal/db"
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

func writeErr(c *gin.Context, err error) {
	code := http.StatusBadRequest
	msg := err.Error()
	switch {
	case errors.Is(err, db.ErrQuotaExceeded):
		code = http.StatusTooManyRequests
		msg = "quota exceeded"
	case errors.Is(err, services.ErrDecryptBusy):
		code = http.StatusConflict
		msg = "decrypt already in progress"
	default:
		if ae, ok := err.(apiError); ok {
			code = ae.status
			msg = ae.message
		}
		if strings.Contains(strings.ToLower(msg), "not found") {
			code = http.StatusNotFound
			msg = "not found"
		}
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
