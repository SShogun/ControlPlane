package main

import (
	"fmt"
	"net/http"
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

	err := app.store.ApproveRevisionTx(r.Context(), revisionID, docID, reviewer.ID)
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

	err := app.store.RejectRevisionTx(r.Context(), revisionID, reviewer.ID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Revision rejected.")
	http.Redirect(w, r, "/approvals", http.StatusSeeOther)
}
