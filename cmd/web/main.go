package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Port          int    `json:"port"`
	Environment   string `json:"environment"`
	Database      string `json:"database"`
	SessionSecret string `json:"session_secret"`
}

type application struct {
	config         Config
	conn           *pgxpool.Pool
	sessionManager *scs.SessionManager
	templateCache  map[string]*template.Template
}

func main() {
	cfg := Config{
		Port:          8080,
		Environment:   os.Getenv("ENVIRONMENT"),
		Database:      os.Getenv("DATABASE_URL"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
	}
	//config
	conn, err := pgxpool.New(context.Background(), cfg.Database)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	//pool creation

	app := application{
		config:        cfg,
		conn:          conn,
		templateCache: make(map[string]*template.Template),
	}
	//application struct initialization
	serv := http.Server{
		Addr:           ":" + strconv.Itoa(cfg.Port),
		Handler:        routes(conn),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	serv.ListenAndServe()
	//server setup
}
