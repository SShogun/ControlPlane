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

		ctx := contextSetUser(r.Context(), &user)
		next.ServeHTTP(w, r.WithContext(ctx))
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

func (app *Application) requireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := contextGetUser(r.Context())

			if user == nil || user.Role != role {
				app.sessionManager.Put(r.Context(), "flash", "You do not have permission to do that.")
				http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
