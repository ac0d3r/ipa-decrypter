package ipatool

import (
	"strings"
	"testing"
)

func TestRedactArgs(t *testing.T) {
	got := redactArgs([]string{"auth", "login", "--email", "a@b.com", "--password", "secret", "--auth-code", "123456"})
	if strings.Contains(got, "secret") || strings.Contains(got, "123456") {
		t.Fatalf("password leaked: %s", got)
	}
	if !strings.Contains(got, "***") {
		t.Fatalf("expected redaction: %s", got)
	}
}
