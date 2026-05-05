package web

import (
	"context"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	conn           *pgxpool.Pool
	logger         *slog.Logger
	store          data.UserStore
	sessionManager *scs.SessionManager
	templateCache  map[string]*template.Template
}

func newTemplateCache(dir string) (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}
	pages, err := filepath.Glob(filepath.Join(dir, "*.page.tmpl"))
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)
		ts, err := template.ParseFiles(page)
		if err != nil {
			return nil, err
		}
		cache[name] = ts
	}
	return cache, nil
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	csrfSecret := os.Getenv("CSRF_SECRET")
	if csrfSecret == "" {
		logger.Warn("CSRF_SECRET not set, using insecure development key")
		csrfSecret = "dev-only-insecure-csrf-key!!!!!" // exactly 32 bytes
	}
	if len(csrfSecret) < 32 {
		log.Fatalf("CSRF_SECRET must be at least 32 bytes; got %d", len(csrfSecret))
	}

	stateStr := os.Getenv("ENV")
	if stateStr == "" {
		stateStr = string(Development)
	}
	state := State(stateStr)

	cfg := Config{
		Port:          6767,
		Database:      os.Getenv("DATABASE_URL"),
		State:         state,
		SecureCookies: state == Production,
		CSRFSecret:    []byte(csrfSecret),
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.Database)
	if err != nil {
		panic(err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		log.Fatalf("Database not reachable: %v", err)
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
		log.Fatalf("failed to build template cache: %v", err)
	}

	app := &Application{
		config:         cfg,
		conn:           pool,
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

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
