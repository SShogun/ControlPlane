package main

import (
	"fmt"
	"net/http"

	"github.com/SShogun/ControlPlane/internal/data"
)

func (app *Application) approveRevisionSubmit(w http.ResponseWriter, r *http.Request) {

	reviewer := contextGetUser(r.Context())

	revisionID := app.readIDParam(r, "revID")
	docID := app.readIDParam(r, "docID")
	if revisionID == 0 {
		revisionID = app.readIntForm(r, "revision_id")
	}
	if docID == 0 {
		docID = app.readIntForm(r, "notebook_id")
	}
	if revisionID == 0 || docID == 0 {
		http.Error(w, "invalid approval submission", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		app.serverError(w, err)
		return
	}
	note := r.PostForm.Get("note")

	err := app.store.ApproveRevisionTx(r.Context(), revisionID, docID, reviewer.ID, note)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Revision approved and published!")
	http.Redirect(w, r, fmt.Sprintf("/notebooks/%d", docID), http.StatusSeeOther)
}

func (app *Application) approvalQueue(w http.ResponseWriter, r *http.Request) {

	revisions, err := app.store.ListSubmittedRevisions(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	data := app.newTemplateData(r)
	data.Revisions = revisions

	app.render(w, r, http.StatusOK, "approval-queue.page.tmpl", data)
}

func (app *Application) approvalReviewView(w http.ResponseWriter, r *http.Request) {
	// Show a single submitted revision with full content and review forms
	revID := app.readIDParam(r, "id")
	if revID == 0 {
		http.Error(w, "invalid revision id", http.StatusBadRequest)
		return
	}

	revisions, err := app.store.ListSubmittedRevisions(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	var target *data.NotebookRevision
	for _, r := range revisions {
		if r.ID == revID {
			rev := r
			target = &rev
			break
		}
	}
	if target == nil {
		http.NotFound(w, r)
		return
	}

	td := app.newTemplateData(r)
	td.CurrentRevision = target
	app.render(w, r, http.StatusOK, "approval-review.page.tmpl", td)
}

func (app *Application) rejectRevisionSubmit(w http.ResponseWriter, r *http.Request) {
	reviewer := contextGetUser(r.Context())
	revisionID := app.readIDParam(r, "revID")
	if revisionID == 0 {
		revisionID = app.readIntForm(r, "revision_id")
	}
	if revisionID == 0 {
		http.Error(w, "invalid rejection submission", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		app.serverError(w, err)
		return
	}
	note := r.PostForm.Get("note")

	err := app.store.RejectRevisionTx(r.Context(), revisionID, reviewer.ID, note)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Revision rejected.")
	http.Redirect(w, r, "/approvals", http.StatusSeeOther)
}
