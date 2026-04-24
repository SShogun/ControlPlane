package main

import (
	"bytes"
	"log"
	"net/http"
)

func (app *Application) render(w http.ResponseWriter, r *http.Request, status int, page string, data *templateData) {
	t, ok := app.templateCache[page]
	if !ok {
		log.Printf("template %q not found in cache", page)
		http.Error(w, "the server encountered a problem and could not process your request", http.StatusInternalServerError)
		return
	}

	var buff bytes.Buffer
	err := t.Execute(&buff, data)
	if err != nil {
		log.Printf("failed to execute template %q: %v", page, err)
		http.Error(w, "the server encountered a problem and could not process your request", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	_, nerr := w.Write(buff.Bytes())
	if nerr != nil {
		log.Printf("failed to write template response: %v", nerr)
	}
}
