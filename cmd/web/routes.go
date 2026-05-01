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
	r.Use(app.authenticate)

	r.Get("/", app.home)
	r.Get("/login", app.loginForm)
	r.Post("/login", app.loginSubmit)

	r.Group(func(r chi.Router) {
		r.Use(app.requireAuthentication)
		r.Get("/dashboard", app.dashboard)
		r.Post("/logout", app.logout)
		r.Get("/notebooks/new", app.notebookCreateForm)
		r.Post("/notebooks/new", app.notebookCreateSubmit)
		r.Get("/notebooks", app.listNotebooks)
		r.Get("/notebooks/{id}", app.notebookView)
		r.Get("/notebooks/{id}/edit", app.notebookEditForm)
		r.Post("/notebooks/{id}/edit", app.notebookEditSubmit)
		r.Get("/notebooks/search", app.notebooksSearch)

		r.Group(func(r chi.Router) {
			r.Use(app.requireRole("reviewer"))

			r.Get("/approvals", app.approvalQueue)
			r.Post("/approvals/approve", app.approveRevisionSubmit)
			r.Post("/approvals/reject", app.rejectRevisionSubmit)
		})
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(app.requireAuthentication)
		r.Use(app.requireRole("admin"))
		r.Get("/audit", app.adminAudit)
	})
	return r

}

func (app *Application) home(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	app.render(w, r, http.StatusOK, "home.page.tmpl", data)
}

func (app *Application) dashboard(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	app.render(w, r, http.StatusOK, "dashboard.page.tmpl", data)
}

func (app *Application) adminAudit(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	app.render(w, r, http.StatusOK, "admin-audit.page.tmpl", data)
}
