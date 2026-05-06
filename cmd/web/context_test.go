package main

import (
	"context"
	"testing"

	"github.com/SShogun/ControlPlane/internal/data"
)

func TestContextUserHelpers(t *testing.T) {
	user := &data.User{ID: 1, Email: "test@example.com"}

	ctx := contextSetUser(context.Background(), user)
	got := contextGetUser(ctx)

	if got == nil {
		t.Fatal("expected user in context")
	}
	if got.ID != user.ID || got.Email != user.Email {
		t.Fatalf("expected user %v; got %v", user, got)
	}
}

func TestContextGetUserReturnsNilWhenMissing(t *testing.T) {
	if got := contextGetUser(context.Background()); got != nil {
		t.Fatalf("expected nil user; got %v", got)
	}
}
