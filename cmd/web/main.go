package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"

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
	sessionManager *scs.SessionManager
	templateCache  map[string]*template.Template
}

// * template caching function
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

		// parse layouts and partials into the template set
		layouts, err := filepath.Glob(filepath.Join(dir, "*.layout.tmpl"))
		if err != nil {
			return nil, err
		}
		if len(layouts) > 0 {
			ts, err = ts.ParseFiles(layouts...)
			if err != nil {
				return nil, err
			}
		}

		cache[name] = ts
	}
	return cache, nil
}

func (c Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port")
	}
	if c.Database == "" {
		return fmt.Errorf("database url required")
	}
	if c.State != Development && c.State != Production {
		return fmt.Errorf("invalid state")
	}
	return nil
}

func main() {
	cfg := Config{
		Port:          6769,
		Database:      os.Getenv("DATABASE_URL"),
		State:         Development,
		SecureCookies: false,
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}
	//load config
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.Database)
	if err != nil {
		panic(err)
	}
	//open pgx pool
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		log.Fatal("Database not reachable", err)
	}
	//ping the pool

	var sessionManager *scs.SessionManager
	sessionManager = scs.New()
	sessionManager.Store = pgxstore.New(pool)
	sessionManager.Lifetime = 12 * time.Hour

	sessionManager.Cookie.Name = "myapp_session"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Secure = cfg.SecureCookies
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.SameSite = http.SameSiteStrictMode
	//create scs session manager
	templateCache, err := newTemplateCache("./ui/templates")
	if err != nil {
		log.Fatalf("failed to build template cache: %v", err)
	}
	//make template cache

	app := &Application{
		config:         cfg,
		conn:           pool,
		sessionManager: sessionManager,
		templateCache:  templateCache,
	}
	//make application

	server := http.Server{
		Addr:           ":" + strconv.Itoa(cfg.Port),
		Handler:        app.routes(),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	//call app.routes()
	go func() {
		log.Printf("starting server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			defer pool.Close()
			log.Fatal("servere error %v", err)
		}
	}()
	// goroutine for logging server close

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}
	pool.Close()
	log.Printf("server stopped gracefully")
}
