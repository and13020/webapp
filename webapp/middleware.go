package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
)

type contextKey string

const (
	contextAuthKey = contextKey("isAuthKey")
)

func (app *application) logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.infoLog.Printf("%s - %s %s %s", r.RemoteAddr, r.Proto, r.Method, r.URL.RequestURI())
		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.isAuthenticated(r) {
			// if not authenticated, redirect to old URL path
			http.Redirect(w, r, fmt.Sprintf("/login?redirectTo=%s", r.URL.Path), http.StatusSeeOther)
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}

func (app *application) isAuthenticated(r *http.Request) bool {
	isAuth, ok := r.Context().Value(contextAuthKey).(bool)
	if !ok {
		return false
	}
	return isAuth
}

func (app *application) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverError(w, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exists := app.session.Exists(r, loggedInUserKey)
		if !exists {
			next.ServeHTTP(w, r)
			return
		}

		_, err := app.user.GetUserByEmail(app.session.GetString(r, loggedInUserKey))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				app.session.Remove(r, loggedInUserKey)
				next.ServeHTTP(w, r)
				return
			}
			app.serverError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), contextAuthKey, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

func (app *application) serverError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())

	// Our app's error log - add to its output
	app.errorLog.Output(2, trace)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
