package api

import (
	"context"
	"net/http"

	"github.com/ac0d3r/ipa-decrypter/internal/db"
)

type ctxKey int

const userKey ctxKey = 1

func withUser(ctx context.Context, u db.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

func userFromContext(r *http.Request) (db.User, bool) {
	u, ok := r.Context().Value(userKey).(db.User)
	return u, ok
}
