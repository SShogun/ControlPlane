package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/SShogun/ControlPlane/internal/data"
	"github.com/SShogun/ControlPlane/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

type adminTeamForm struct {
	Email    string
	Password string
	TeamID   string
	Role     string
	validator.Validator
}

func (app *Application) adminTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := app.store.ListTeams(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	data := app.newTemplateData(r)
	data.Teams = teams
	data.Form = adminTeamForm{Role: "member"}
	app.render(w, r, http.StatusOK, "admin-teams.page.tmpl", data)
}

func (app *Application) adminTeamOnboardSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	form := adminTeamForm{
		Email:    strings.ToLower(strings.TrimSpace(r.PostForm.Get("email"))),
		Password: r.PostForm.Get("password"),
		TeamID:   r.PostForm.Get("team_id"),
		Role:     strings.TrimSpace(r.PostForm.Get("role")),
	}

	form.Check(validator.NotBlank(form.Email), "email", "Email is required")
	form.Check(validator.NotBlank(form.Password), "password", "Password is required")
	form.Check(validator.MinChars(form.Password, 8), "password", "Password must be at least 8 characters")
	form.Check(validator.NotBlank(form.TeamID), "team_id", "Team is required")
	form.Check(form.Role == "member" || form.Role == "reviewer", "role", "Role must be member or reviewer")

	teams, err := app.store.ListTeams(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	teamID := app.readIntForm(r, "team_id")
	if teamID == 0 {
		form.Check(false, "team_id", "Team is required")
	}

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Teams = teams
		data.Form = form
		data.Errors = form.FieldErrors
		app.render(w, r, http.StatusUnprocessableEntity, "admin-teams.page.tmpl", data)
		return
	}

	if _, err := app.store.GetTeam(r.Context(), teamID); err != nil {
		app.serverError(w, err)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(form.Password), bcrypt.DefaultCost)
	if err != nil {
		app.serverError(w, err)
		return
	}

	userID, err := app.store.CreateUser(r.Context(), data.CreateUserParams{Email: form.Email, PasswordHash: hash})
	if err != nil {
		app.serverError(w, err)
		return
	}
	if err := app.store.AddMembership(r.Context(), userID, teamID, form.Role); err != nil {
		app.serverError(w, err)
		return
	}

	app.logAuditEvent(r, "member_onboarded", "user", userID, fmt.Sprintf(`{"email": %q, "team_id": %d, "role": %q}`, form.Email, teamID, form.Role))
	app.sessionManager.Put(r.Context(), "flash", "Member onboarded successfully.")
	http.Redirect(w, r, "/admin/teams", http.StatusSeeOther)
}

func (app *Application) notebookDeleteSubmit(w http.ResponseWriter, r *http.Request) {
	app.deleteNotebookIfAllowed(w, r, false)
}

func (app *Application) adminNotebookDeleteSubmit(w http.ResponseWriter, r *http.Request) {
	app.deleteNotebookIfAllowed(w, r, true)
}

func (app *Application) deleteNotebookIfAllowed(w http.ResponseWriter, r *http.Request, allowAdminPublished bool) {
	user := contextGetUser(r.Context())
	notebookID := app.readIDParam(r, "id")
	if notebookID == 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	revisions, err := app.store.ListNotebookRevisions(r.Context(), notebookID)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if len(revisions) == 0 {
		http.NotFound(w, r)
		return
	}

	current := revisions[0]
	action := ""
	entityType := "notebook"
	entityID := notebookID
	details := fmt.Sprintf(`{"title": %q, "status": %q}`, current.Title, current.Status)
	redirectTo := "/dashboard"

	if current.Status == "draft" && user.ID == current.AuthorID {
		action = "draft_deleted"
		entityType = "notebook_revision"
		entityID = current.ID
		if len(revisions) == 1 {
			if err := app.store.DeleteNotebook(r.Context(), notebookID); err != nil {
				app.serverError(w, err)
				return
			}
		} else {
			if err := app.store.DeleteNotebookRevision(r.Context(), current.ID); err != nil {
				app.serverError(w, err)
				return
			}
		}
	} else if allowAdminPublished && user.Role == "admin" && current.Status == "approved" {
		action = "notebook_deleted"
		redirectTo = "/admin/teams"
		if err := app.store.DeleteNotebook(r.Context(), notebookID); err != nil {
			app.serverError(w, err)
			return
		}
	} else {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	app.logAuditEvent(r, action, entityType, entityID, details)
	app.sessionManager.Put(r.Context(), "flash", "Notebook deleted.")
	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}
