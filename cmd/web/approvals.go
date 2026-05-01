package web

import (
	"fmt"
	"net/http"

	"github.com/SShogun/ControlPlane/internal/data"
)

func (app *Application) approveRevisionSubmit(w http.ResponseWriter, r *http.Request) {

	reviewer := contextGetUser(r.Context())

	revisionID := app.readIDParam(r, "revID")
	docID := app.readIDParam(r, "docID")

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
	revisionID := app.readIDParam(r, "revID")

	err := app.store.UpdateRevisionStatus(r.Context(), data.UpdateRevisionStatusParams{
		Status: "rejected",
		ID:     revisionID,
	})
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.logAuditEvent(r, "revision_rejected", "revision", revisionID)

	app.sessionManager.Put(r.Context(), "flash", "Revision rejected.")
	http.Redirect(w, r, "/approvals", http.StatusSeeOther)
}
