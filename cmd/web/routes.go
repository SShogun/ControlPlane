package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"
)

func (app *Application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(app.sessionManager.LoadAndSave)
	if !app.config.SecureCookies {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, csrf.PlaintextHTTPRequest(r))
			})
		})
	}

	csrfMiddleware := csrf.Protect(
		app.config.CSRFSecret,
		csrf.Secure(app.config.SecureCookies),
		csrf.Path("/"),
		csrf.CookieName("csrf_token"),
		csrf.FieldName("csrf_token"),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			app.logger.Error("CSRF error", "reason", csrf.FailureReason(r), "host", r.Host, "referer", r.Referer())
			http.Error(w, csrf.FailureReason(r).Error(), http.StatusForbidden)
		})),
	)
	r.Use(csrfMiddleware)

	r.Use(app.authenticate)

	r.Get("/", app.home)
	r.Get("/login", app.loginForm)
	r.With(app.rateLimitLogin).Post("/login", app.loginSubmit)

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
