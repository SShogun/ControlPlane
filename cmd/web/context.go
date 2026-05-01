package web

import (
	"context"

	"github.com/SShogun/ControlPlane/internal/data"
)

type contextKey string

const userContextKey contextKey = "user"

func contextSetUser(ctx context.Context, user *data.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func contextGetUser(ctx context.Context) *data.User {
	user, ok := ctx.Value(userContextKey).(*data.User)
	if !ok {
		return nil
	}
	return user
}
