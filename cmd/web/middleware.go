package web

import "net/http"

func (app *Application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := app.sessionManager.GetInt(r.Context(), "userID")
		if id == 0 {
			next.ServeHTTP(w, r)
			return
		}
		user, err := app.store.GetUser(r.Context(), id)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := contextSetUser(r.Context(), &user) // basically now the context is stapled (quite literally) with the user database profile
		next.ServeHTTP(w, r.WithContext(ctx))     // passing forward the request which has the new context in it
	})
}

func (app *Application) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contextGetUser(r.Context()) == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}
