package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/SShogun/ControlPlane/internal/data"
	"github.com/SShogun/ControlPlane/internal/validator"
	"github.com/go-chi/chi/v5"
)

// notebookForm carries user input and validation state for notebook create/edit.
type notebookForm struct {
	Title      string
	Slug       string
	Visibility string
	Body       string
	Tags       string
	validator.Validator
}

func (app *Application) notebookCreateForm(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = notebookForm{}
	app.render(w, r, http.StatusOK, "notebook-create.page.tmpl", data)
}

func (app *Application) notebookCreateSubmit(w http.ResponseWriter, r *http.Request) {
	user := contextGetUser(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	form := notebookForm{
		Title:      r.PostForm.Get("title"),
		Slug:       r.PostForm.Get("slug"),
		Visibility: r.PostForm.Get("visibility"),
		Body:       r.PostForm.Get("body"),
		Tags:       r.PostForm.Get("tags"),
	}

	form.Check(validator.NotBlank(form.Title), "title", "Title is required")
	form.Check(validator.MaxChars(form.Title, 200), "title", "Title must be 200 characters or less")
	form.Check(validator.NotBlank(form.Slug), "slug", "Slug is required")
	form.Check(form.Visibility == "public" || form.Visibility == "team" || form.Visibility == "private", "visibility", "Invalid visibility")
	form.Check(validator.MaxChars(form.Body, 50000), "body", "Body must be 50,000 characters or less")

	if !form.Valid() {
		td := app.newTemplateData(r)
		td.Form = form
		td.Errors = form.FieldErrors
		app.render(w, r, http.StatusUnprocessableEntity, "notebook-create.page.tmpl", td)
		return
	}

	id, err := app.store.CreateDraft(r.Context(), data.CreateDraftParams{
		AuthorID:   user.ID,
		Title:      form.Title,
		Slug:       form.Slug,
		Visibility: form.Visibility,
		Body:       form.Body,
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
	if err := app.attachTagsFromInput(r, id, form.Tags); err != nil {
		app.serverError(w, err)
		return
	}

	// include title in audit details
	app.logAuditEvent(r, "draft_created", "notebook", id, fmt.Sprintf(`{"title": %q}`, form.Title))
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
	if len(revisions) == 0 {
		http.NotFound(w, r)
		return
	}

	viewData := app.newTemplateData(r)
	viewData.CurrentRevision = &revisions[0]
	tags, err := app.store.ListNotebookTags(r.Context(), id)
	if err != nil {
		app.serverError(w, err)
		return
	}
	viewData.Tags = tags

	app.render(w, r, http.StatusOK, "notebook-view.page.tmpl", viewData)
}

func (app *Application) handleDraftError(w http.ResponseWriter, r *http.Request, notebookID int, err error) {
	switch {
	case errors.Is(err, data.ErrNotFound):
		http.NotFound(w, r)
	case errors.Is(err, data.ErrForbidden):
		http.Error(w, "forbidden", http.StatusForbidden)
	case errors.Is(err, data.ErrNotDraft):
		app.sessionManager.Put(r.Context(), "flash", "Only draft revisions can be modified.")
		http.Redirect(w, r, fmt.Sprintf("/notebooks/%d", notebookID), http.StatusSeeOther)
	default:
		app.serverError(w, err)
	}
}

func (app *Application) editableDraft(w http.ResponseWriter, r *http.Request, notebookID int) (data.NotebookRevision, bool) {
	user := contextGetUser(r.Context())
	if user == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return data.NotebookRevision{}, false
	}
	revision, err := app.store.GetEditableDraft(r.Context(), notebookID, user.ID)
	if err != nil {
		app.handleDraftError(w, r, notebookID, err)
		return data.NotebookRevision{}, false
	}
	return revision, true
}

func (app *Application) notebookEditForm(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || !validator.PositiveInt(id) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	revision, ok := app.editableDraft(w, r, id)
	if !ok {
		return
	}

	td := app.newTemplateData(r)
	td.CurrentRevision = &revision
	td.Form = notebookForm{
		Title: revision.Title,
		Body:  revision.Body,
	}
	tags, err := app.store.ListNotebookTags(r.Context(), id)
	if err != nil {
		app.serverError(w, err)
		return
	}
	tagNames := make([]string, 0, len(tags))
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
	}
	td.Form = notebookForm{
		Title: revision.Title,
		Body:  revision.Body,
		Tags:  strings.Join(tagNames, ", "),
	}
	app.render(w, r, http.StatusOK, "notebook-edit.page.tmpl", td)
}

func (app *Application) notebookEditSubmit(w http.ResponseWriter, r *http.Request) {
	user := contextGetUser(r.Context())
	idStr := chi.URLParam(r, "id")
	notebookID, err := strconv.Atoi(idStr)
	if err != nil || !validator.PositiveInt(notebookID) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	revision, ok := app.editableDraft(w, r, notebookID)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	form := notebookForm{
		Title: r.PostForm.Get("title"),
		Body:  r.PostForm.Get("body"),
		Tags:  r.PostForm.Get("tags"),
	}
	form.Check(validator.NotBlank(form.Title), "title", "Title is required")
	form.Check(validator.MaxChars(form.Title, 200), "title", "Title must be 200 characters or less")
	form.Check(validator.MaxChars(form.Body, 50000), "body", "Body must be 50,000 characters or less")

	if !form.Valid() {
		td := app.newTemplateData(r)
		td.CurrentRevision = &revision
		td.Form = form
		td.Errors = form.FieldErrors
		app.render(w, r, http.StatusUnprocessableEntity, "notebook-edit.page.tmpl", td)
		return
	}

	revisionID, err := app.store.UpdateDraft(r.Context(), data.UpdateDraftParams{
		NotebookID: notebookID,
		AuthorID:   user.ID,
		Title:      form.Title,
		Body:       form.Body,
		Tags:       parseTags(form.Tags),
	})
	if err != nil {
		app.handleDraftError(w, r, notebookID, err)
		return
	}

	app.logAuditEvent(r, "draft_updated", "notebook_revision", revisionID, fmt.Sprintf(`{"title": %q}`, form.Title))
	app.sessionManager.Put(r.Context(), "flash", "Draft updated")
	http.Redirect(w, r, fmt.Sprintf("/notebooks/%d", notebookID), http.StatusSeeOther)
}

func (app *Application) notebookFlagSubmit(w http.ResponseWriter, r *http.Request) {
	user := contextGetUser(r.Context())
	idStr := chi.URLParam(r, "id")
	notebookID, err := strconv.Atoi(idStr)
	if err != nil || !validator.PositiveInt(notebookID) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	reason := strings.TrimSpace(r.PostForm.Get("reason"))
	if reason == "" {
		app.sessionManager.Put(r.Context(), "flash", "A moderation reason is required.")
		http.Redirect(w, r, fmt.Sprintf("/notebooks/%d", notebookID), http.StatusSeeOther)
		return
	}

	flagID, err := app.store.CreateModerationFlag(r.Context(), data.CreateModerationFlagParams{
		NotebookID:  notebookID,
		ModeratorID: user.ID,
		Reason:      reason,
	})
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.logAuditEvent(r, "moderation_flag_created", "moderation_flag", flagID, fmt.Sprintf(`{"reason": %q}`, reason))
	app.sessionManager.Put(r.Context(), "flash", "Notebook flagged for moderation.")
	http.Redirect(w, r, fmt.Sprintf("/notebooks/%d", notebookID), http.StatusSeeOther)
}

func (app *Application) notebookSubmitForApproval(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	notebookID, err := strconv.Atoi(idStr)
	if err != nil || !validator.PositiveInt(notebookID) {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if _, ok := app.editableDraft(w, r, notebookID); !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	msg := r.PostForm.Get("message")

	user := contextGetUser(r.Context())
	revisionID, err := app.store.SubmitDraft(r.Context(), notebookID, user.ID)
	if err != nil {
		app.handleDraftError(w, r, notebookID, err)
		return
	}

	app.logAuditEvent(r, "revision_submitted", "notebook_revision", revisionID, fmt.Sprintf(`{"message": %q}`, msg))
	app.sessionManager.Put(r.Context(), "flash", "Draft submitted for approval.")
	http.Redirect(w, r, fmt.Sprintf("/notebooks/%d", notebookID), http.StatusSeeOther)
}

type searchForm struct {
	Query string
}

func (app *Application) notebooksSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	td := app.newTemplateData(r)
	td.Form = searchForm{Query: query}

	if query != "" {
		notebooks, err := app.store.SearchNotebooks(r.Context(), query)
		if err != nil {
			app.serverError(w, err)
			return
		}
		td.Notebooks = notebooks
	}

	app.render(w, r, http.StatusOK, "notebooks-search.page.tmpl", td)
}

func parseTags(input string) []string {
	var tags []string
	for _, raw := range strings.Split(input, ",") {
		if name := strings.ToLower(strings.TrimSpace(raw)); name != "" {
			tags = append(tags, name)
		}
	}
	return tags
}

func (app *Application) attachTagsFromInput(r *http.Request, notebookID int, input string) error {
	for _, name := range parseTags(input) {
		tagID, err := app.store.CreateTag(r.Context(), name)
		if err != nil {
			return err
		}
		if err := app.store.AttachTag(r.Context(), notebookID, tagID); err != nil {
			return err
		}
	}
	return nil
}
