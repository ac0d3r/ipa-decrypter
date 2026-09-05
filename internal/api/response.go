package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ac0d3r/ipa-decrypter/internal/db"
	"github.com/ac0d3r/ipa-decrypter/internal/services"
	"gorm.io/gorm"
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
	case errors.Is(err, services.ErrNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		code = http.StatusNotFound
		msg = "not found"
	case errors.Is(err, services.ErrTaskNotDeletable):
		code = http.StatusConflict
		msg = "only completed or failed tasks can be deleted"
	case errors.Is(err, db.ErrVoucherInvalid):
		msg = "invalid voucher"
	case errors.Is(err, db.ErrVoucherUsed):
		msg = "voucher already used"
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
