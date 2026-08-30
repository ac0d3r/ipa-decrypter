package api

import (
	"context"
	"net/http"

	"github.com/ac0d3r/ipa-decrypter/internal/db"
)

type ctxKey int

const adminUserKey ctxKey = 1

func withAdminUser(ctx context.Context, u db.AdminUser) context.Context {
	return context.WithValue(ctx, adminUserKey, u)
}

func adminUserFromContext(r *http.Request) (db.AdminUser, bool) {
	u, ok := r.Context().Value(adminUserKey).(db.AdminUser)
	return u, ok
}
