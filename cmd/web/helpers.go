package web

import (
	"bytes"
	"net/http"
	"strconv"

	"github.com/SShogun/ControlPlane/internal/data"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
)

type templateData struct {
	Flash           string
	User            *data.User
	IsAuthenticated bool
	Form            any
	Errors          map[string]string
	CSRFToken       string
	Notebooks       []data.Notebook
	CurrentRevision *data.NotebookRevision
	Tags            []data.Tag
	Revisions       []data.NotebookRevision
}

func (app *Application) render(w http.ResponseWriter, r *http.Request, status int, page string, data *templateData) {
	t, ok := app.templateCache[page]
	if !ok {
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		app.logger.Error("failed to write response", "error", err)
	}
}

func (app *Application) newTemplateData(r *http.Request) *templateData {
	user := contextGetUser(r.Context())
	return &templateData{
		Flash:           app.sessionManager.PopString(r.Context(), "flash"),
		User:            user,
		IsAuthenticated: user != nil,
		CSRFToken:       csrf.Token(r),
	}
}

func (app *Application) logAuditEvent(r *http.Request, action, entityType string, entityID int) {
	user := contextGetUser(r.Context())
	if user == nil {
		return
	}

	err := app.store.InsertAuditLog(r.Context(), data.InsertAuditLogParams{
		UserID:     user.ID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
	})

	if err != nil {
		app.logger.Error("failed to insert audit log", "error", err)
	}
}

func (app *Application) readIDParam(r *http.Request, name string) int {
	idStr := chi.URLParam(r, name)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0
	}
	return id
}

func (app *Application) serverError(w http.ResponseWriter, err error) {
	app.logger.Error("server error", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
