package main

import (
	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func routes(conn *pgxpool.Pool) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	sessionManager := scs.New()
	sessionManager.Store = pgxstore.New(conn)
	//session management
	r.Get("/", home)
	r.Post("/login", login)
	r.Get("/notebook/list", notebookList)
	r.Post("/notebook/create", notebookCreate)
	r.Get("/notebook/{id}", notebookView)
	r.Post("/notebook/edit", notebookEdit)
	r.Post("/notebook/search", notebookSearch)

	return r
}
