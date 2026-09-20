package main

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
)

type fakeGracefulServer struct {
	listenStarted  chan struct{}
	shutdownCalled chan struct{}
	stop           chan struct{}
	listenErr      error
	shutdownErr    error
	shutdownOnce   sync.Once
}

func newFakeGracefulServer() *fakeGracefulServer {
	return &fakeGracefulServer{
		listenStarted:  make(chan struct{}),
		shutdownCalled: make(chan struct{}),
		stop:           make(chan struct{}),
	}
}

func (s *fakeGracefulServer) ListenAndServe() error {
	close(s.listenStarted)
	if s.listenErr != nil {
		return s.listenErr
	}
	<-s.stop
	return http.ErrServerClosed
}

func (s *fakeGracefulServer) Shutdown(context.Context) error {
	s.shutdownOnce.Do(func() {
		close(s.shutdownCalled)
		close(s.stop)
	})
	return s.shutdownErr
}

func TestServeShutsDownWhenContextIsCancelled(t *testing.T) {
	server := newFakeGracefulServer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, server, time.Second)
	}()

	select {
	case <-server.listenStarted:
	case <-time.After(time.Second):
		t.Fatal("server did not start")
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("serve did not return after cancellation")
	}

	select {
	case <-server.shutdownCalled:
	default:
		t.Fatal("expected graceful shutdown to be called")
	}
}

func TestServeReturnsUnexpectedListenError(t *testing.T) {
	wantErr := errors.New("listen failed")
	server := newFakeGracefulServer()
	server.listenErr = wantErr

	err := serve(context.Background(), server, time.Second)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected listen error %v, got %v", wantErr, err)
	}

	select {
	case <-server.shutdownCalled:
		t.Fatal("shutdown should not run after an immediate listen failure")
	default:
	}
}

func TestServeReturnsShutdownError(t *testing.T) {
	wantErr := errors.New("shutdown failed")
	server := newFakeGracefulServer()
	server.shutdownErr = wantErr
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, server, time.Second)
	}()

	select {
	case <-server.listenStarted:
	case <-time.After(time.Second):
		t.Fatal("server did not start")
	}

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected shutdown error %v, got %v", wantErr, err)
		}
	case <-time.After(time.Second):
		t.Fatal("serve did not return after shutdown error")
	}
}
