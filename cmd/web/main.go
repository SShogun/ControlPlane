package main

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/SShogun/ControlPlane/internal/data"
	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type State string

const (
	Development State = "dev"
	Production  State = "prod"
)

type Config struct {
	Port          int    `json:"port"`
	Database      string `json:"database"`
	State         State  `json:"state"`
	SecureCookies bool   `json:"secure_cookies"`
	CSRFSecret    []byte `json:"-"`
}

type Application struct {
	config         Config
	logger         *slog.Logger
	store          data.UserStore
	sessionManager *scs.SessionManager
	templateCache  map[string]*template.Template
}

type gracefulServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

func newTemplateCache(dir string) (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}
	pages, err := filepath.Glob(filepath.Join(dir, "*.page.tmpl"))
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)
		ts, err := template.ParseFiles(filepath.Join(dir, "base.layout.tmpl"), page)
		if err != nil {
			return nil, err
		}
		cache[name] = ts
	}
	return cache, nil
}

func serve(ctx context.Context, server gracefulServer, shutdownTimeout time.Duration) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve: listen: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("serve: shutdown: %w", err)
		}

		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: listen after shutdown: %w", err)
		}
		return nil
	}
}

func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	csrfSecret := os.Getenv("CSRF_SECRET")
	if csrfSecret == "" {
		logger.Warn("CSRF_SECRET not set, using insecure development key")
		csrfSecret = "dev-only-insecure-csrf-key!!!!!!"
	}
	if len(csrfSecret) < 32 {
		return fmt.Errorf("CSRF_SECRET must be at least 32 bytes; got %d", len(csrfSecret))
	}

	stateStr := os.Getenv("ENV")
	if stateStr == "" {
		stateStr = os.Getenv("ENVIRONMENT")
	}
	if stateStr == "" {
		stateStr = string(Development)
	}
	state := State(stateStr)

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://testuser:testpass@localhost:5433/controlplane_test?sslmode=disable"
		logger.Warn("DATABASE_URL not set, using local docker-compose.test.yml database")
	}

	cfg := Config{
		Port:          6767,
		Database:      databaseURL,
		State:         state,
		SecureCookies: state == Production,
		CSRFSecret:    []byte(csrfSecret),
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("open database pool: %w", err)
	}
	defer pool.Close()

	pingCtx, cancelPing := context.WithTimeout(ctx, 5*time.Second)
	err = pool.Ping(pingCtx)
	cancelPing()
	if err != nil {
		return fmt.Errorf("database not reachable: %w", err)
	}

	sessionManager := scs.New()
	sessionManager.Store = pgxstore.New(pool)
	sessionManager.Lifetime = 12 * time.Hour
	sessionManager.Cookie.Name = "myapp_session"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Secure = cfg.SecureCookies
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.SameSite = http.SameSiteStrictMode

	templateCache, err := newTemplateCache("./ui/templates")
	if err != nil {
		return fmt.Errorf("build template cache: %w", err)
	}

	app := &Application{
		config:         cfg,
		logger:         logger,
		store:          &data.PgxStore{DB: pool},
		sessionManager: sessionManager,
		templateCache:  templateCache,
	}

	server := http.Server{
		Addr:         ":" + strconv.Itoa(cfg.Port),
		Handler:      app.routes(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	serverCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("server starting", "addr", server.Addr)
	if err := serve(serverCtx, &server, 10*time.Second); err != nil {
		return err
	}
	logger.Info("server stopped")
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
