package main

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

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

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
		r.Get("/notebooks/search", app.notebooksSearch)
		r.Get("/notebooks/{id}", app.notebookView)
		r.Get("/notebooks/{id}/edit", app.notebookEditForm)
		r.Post("/notebooks/{id}/edit", app.notebookEditSubmit)
		r.Post("/notebooks/{id}/submit", app.notebookSubmitForApproval)
		r.Post("/notebooks/{id}/flag", app.notebookFlagSubmit)

		r.Group(func(r chi.Router) {
			r.Use(app.requireRole("reviewer"))

			r.Get("/approvals", app.approvalQueue)
			r.Post("/approvals/approve", app.approveRevisionSubmit)
			r.Post("/approvals/reject", app.rejectRevisionSubmit)
			r.Get("/moderation", app.moderationQueue)
			r.Post("/moderation/resolve", app.resolveModerationFlagSubmit)
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
	user := contextGetUser(r.Context())
	data := app.newTemplateData(r)
	if user != nil {
		revisions, err := app.store.ListRecentDrafts(r.Context(), user.ID)
		if err != nil {
			app.serverError(w, err)
			return
		}
		data.Revisions = revisions
	}
	app.render(w, r, http.StatusOK, "dashboard.page.tmpl", data)
}

func (app *Application) adminAudit(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	events, err := app.store.ListAuditEvents(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}
	data.AuditEvents = events
	app.render(w, r, http.StatusOK, "admin-audit.page.tmpl", data)
}

func (app *Application) moderationQueue(w http.ResponseWriter, r *http.Request) {
	flags, err := app.store.ListModerationFlags(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}
	data := app.newTemplateData(r)
	data.ModerationFlags = flags
	app.render(w, r, http.StatusOK, "moderation.page.tmpl", data)
}

func (app *Application) resolveModerationFlagSubmit(w http.ResponseWriter, r *http.Request) {
	flagID := app.readIntForm(r, "flag_id")
	if flagID == 0 {
		http.Error(w, "invalid moderation flag", http.StatusBadRequest)
		return
	}
	if err := app.store.ResolveModerationFlag(r.Context(), flagID); err != nil {
		app.serverError(w, err)
		return
	}
	app.logAuditEvent(r, "moderation_flag_resolved", "moderation_flag", flagID)
	app.sessionManager.Put(r.Context(), "flash", "Moderation flag resolved.")
	http.Redirect(w, r, "/moderation", http.StatusSeeOther)
}
