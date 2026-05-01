package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *Application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(app.sessionManager.LoadAndSave)

	r.Get("/", app.Home)

	return r

}

func (app *Application) Home(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)                                  //we are getting a new template data object, in which the flash message is the string we get from the request
	data.Flash = app.sessionManager.PopString(r.Context(), "flash") //then we store the flash message in the data
	app.render(w, r, http.StatusOK, "home.page.tmpl", data)         //we bind the template & the data into the http page using render function
}
