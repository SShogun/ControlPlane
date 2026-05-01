package web

import (
	"bytes"
	"net/http"
)

type templateData struct {
	Flash string
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
	return &templateData{
		Flash: app.sessionManager.PopString(r.Context(), "flash"),
	}
}
