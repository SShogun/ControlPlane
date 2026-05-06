package main

import (
	"net/http"
)

func (app *Application) loginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.sessionManager.Put(r.Context(), "flash", "Invalid login submission")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	email := r.PostForm.Get("email")
	password := r.PostForm.Get("password")

	user, err := app.store.GetUserByEmail(r.Context(), email)
	if err != nil {
		app.sessionManager.Put(r.Context(), "flash", "invalid credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if !app.store.CheckPassword(user, password) {
		app.sessionManager.Put(r.Context(), "flash", "invalid credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := app.sessionManager.RenewToken(r.Context()); err != nil {
		app.serverError(w, err)
		return
	}
	app.sessionManager.Put(r.Context(), "userID", user.ID)
	app.sessionManager.Put(r.Context(), "flash", "You have been logged in!")
	app.logAuditEvent(r, "user_login", "user", user.ID)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

type userLoginForm struct {
	Email string
}

func (app *Application) loginForm(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, r, http.StatusOK, "login.page.tmpl", data)
}

func (app *Application) logout(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.Destroy(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	app.sessionManager.Put(r.Context(), "flash", "You have been logged out!")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
