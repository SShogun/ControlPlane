package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *Application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(app.sessionManager.LoadAndSave)

	r.Get("/", app.Home)
	r.Get("/login", app.loginForm)
	r.Post("/login", app.loginSubmit)

	r.Group(func(r chi.Router) {
		r.Use(app.requireAuthentication)
		r.Get("/dashboard", app.dashboard)
		r.Post("/logout", app.logout)
		//notebooks create thingies
		r.Get("/notebooks/new", app.notebookCreateForm)
		r.Post("/notebooks/new", app.notebookCreateSubmit)
		//notebooks view things
		r.Get("/notebooks", app.listNotebooks)
		r.Get("/notebooks/{id}", app.notebookView)
		//notebooks edit things
		r.Get("/notebooks/{id}/edit", app.notebookEditForm)
		r.Post("/notebooks/{id}/edit", app.notebookEditSubmit)
		//notebooks search things
		r.Get("/notebooks/search", app.notebooksSearch)
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(app.requireAuthentication)
		r.Use(app.requireRole("admin"))
		r.Get("/audit", app.adminAudit)
	})
	return r

}

func (app *Application) Home(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)                          //then we store the flash message in the data
	app.render(w, r, http.StatusOK, "home.page.tmpl", data) //we bind the template & the data into the http page using render function
}

func (app *Application) dashboard(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	app.render(w, r, http.StatusOK, "dashboard.page.tmpl", data)
}

func (app *Application) requireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}

func (app *Application) adminAudit(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	app.render(w, r, http.StatusOK, "admin-audit.page.tmpl", data)
}
