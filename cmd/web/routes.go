package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type templateData struct {
	Flash string
}

func (app *Application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(app.sessionManager.LoadAndSave)

	r.Get("/", app.Home)

	return r
}

func (app *Application) newTemplateData(r *http.Request) *templateData {
	return &templateData{}
}

func (app *Application) Home(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Flash = app.sessionManager.PopString(r.Context(), "flash")
	app.render(w, r, http.StatusOK, "home.page.tmpl", data)
}
