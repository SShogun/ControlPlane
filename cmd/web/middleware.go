package main

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

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
			allowed := user != nil && (user.Role == role || (role == "reviewer" && user.Role == "admin"))

			if !allowed {
				app.sessionManager.Put(r.Context(), "flash", "You do not have permission to do that.")
				http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (app *Application) rateLimitLogin(next http.Handler) http.Handler {
	type Client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*Client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		ip := r.RemoteAddr
		mu.Lock()
		if _, found := clients[ip]; !found {
			// Limit to 1 request per second, with a burst of 3
			clients[ip] = &Client{limiter: rate.NewLimiter(1, 3)}
		}
		clients[ip].lastSeen = time.Now()
		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			app.logger.Warn("rate limit exceeded", "ip", ip)
			http.Error(w, "Too many login attempts. Please wait.", http.StatusTooManyRequests)
			return
		}
		mu.Unlock()
		next.ServeHTTP(w, r)
	})
}
