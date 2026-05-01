package web

import (
	"context"
	"html/template"
	"log"
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
}

type Application struct {
	config         Config
	conn           *pgxpool.Pool
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
	cfg := Config{
		Port:          6767,
		Database:      os.Getenv("DATABASE_URL"),
		State:         Development,
		SecureCookies: false,
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

	var sessionManager *scs.SessionManager
	sessionManager = scs.New()
	// attach pgx-backed store for scs session manager
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

	server.ListenAndServe()
}
