package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/SShogun/ControlPlane/internal/data"
	"github.com/SShogun/ControlPlane/internal/validator"
	"github.com/go-chi/chi/v5"
)

// notebookForm carries user input and validation state for notebook create/edit.
type notebookForm struct {
	Title string
	Body  string
	validator.Validator
}

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
		Title: r.PostForm.Get("title"),
		Body:  r.PostForm.Get("body"),
	}

	form.Check(validator.NotBlank(form.Title), "title", "Title is required")
	form.Check(validator.MaxChars(form.Title, 200), "title", "Title must be 200 characters or less")
	form.Check(validator.MaxChars(form.Body, 50000), "body", "Body must be 50,000 characters or less")

	if !form.Valid() {
		td := app.newTemplateData(r)
		td.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "notebook-create.page.tmpl", td)
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
	if err != nil || !validator.PositiveInt(id) {
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
	tags, err := app.store.ListNotebookTags(r.Context(), id)
	if err != nil {
		app.serverError(w, err)
		return
	}
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
