package web

import (
	"bytes"
	"net/http"

	"github.com/SShogun/ControlPlane/internal/data"
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
	buf.WriteTo(w)
}

func (app *Application) newTemplateData(r *http.Request) *templateData {
	user := contextGetUser(r.Context())
	return &templateData{
		Flash:           app.sessionManager.PopString(r.Context(), "flash"),
		User:            user,
		IsAuthenticated: user != nil,
	}
}
