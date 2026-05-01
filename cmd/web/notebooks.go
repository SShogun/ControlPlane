package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/SShogun/ControlPlane/internal/data"
	"github.com/go-chi/chi/v5"
)

type notebookForm struct {
	Title  string
	Body   string
	Errors map[string]string
}

/*
GET listNotebooks
GET viewNotebook/?=id
GET createDraftForm
POST createDraftForm
GET editDraftForm
POST editDraftForm
GET searchNotebooks
GET notebooks/?=
*/

func (app *Application) notebookCreateForm(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	app.render(w, r, http.StatusOK, "notebook-create.page.tmpl", data)
}

func (app *Application) notebookCreateSubmit(w http.ResponseWriter, r *http.Request) {
	user := contextGetUser(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	form := notebookForm{
		Title:  r.PostForm.Get("title"),
		Body:   r.PostForm.Get("body"),
		Errors: map[string]string{},
	}

	if form.Title == "" {
		form.Errors["title"] = "Title is required"
	}

	if len(form.Errors) > 0 {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "notebook-create.page.tmpl", data)
		return
	}

	id, err := app.store.CreateDraft(r.Context(), data.CreateDraftParams{
		AuthorID: user.ID,
		Title:    form.Title,
		Body:     form.Body,
	})
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	// create an initial revision for the newly created notebook
	_, err = app.store.CreateNotebookRevision(r.Context(), data.CreateNotebookRevisionParams{
		DocumentID: id,
		AuthorID:   user.ID,
		Title:      form.Title,
		Body:       form.Body,
		Status:     "draft",
	})
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	app.logAuditEvent(r, "draft_created", "notebook_revision", id)
	app.sessionManager.Put(r.Context(), "flash", "Draft Created")
	http.Redirect(w, r, fmt.Sprintf("/notebooks/%d", id), http.StatusSeeOther)
}

func (app *Application) listNotebooks(w http.ResponseWriter, r *http.Request) {
	notebooks, err := app.store.ListNotebooks(r.Context())
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	viewData := app.newTemplateData(r)
	viewData.Notebooks = notebooks
	app.render(w, r, http.StatusOK, "notebooks.page.tmpl", viewData)
}

func (app *Application) notebookView(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	revisions, err := app.store.ListNotebookRevisions(r.Context(), id)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	viewData := app.newTemplateData(r)
	if len(revisions) > 0 {
		viewData.CurrentRevision = &revisions[0]
	}
	tags, _ := app.store.ListNotebookTags(r.Context(), id)
	viewData.Tags = tags

	app.render(w, r, http.StatusOK, "notebook-view.page.tmpl", viewData)
}

func (app *Application) notebookEditForm(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	app.render(w, r, http.StatusOK, "notebook-edit.page.tmpl", data)
}

func (app *Application) notebookEditSubmit(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/notebooks", http.StatusSeeOther)
}

func (app *Application) notebooksSearch(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	app.render(w, r, http.StatusOK, "notebooks-search.page.tmpl", data)
}
