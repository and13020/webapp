package main

import (
	"net/http"

	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	defaultMiddleware := alice.New(app.recover, app.logger)
	secureMiddleware := alice.New(app.session.Enable, app.authenticate)

	// adding static html path/route
	mux.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir(app.publicPath))))

	mux.Handle("/", secureMiddleware.ThenFunc(app.home))
	mux.Handle("/about", secureMiddleware.ThenFunc(app.about))
	mux.Handle("/contact", secureMiddleware.ThenFunc(app.contact))
	mux.Handle("/login", secureMiddleware.ThenFunc(app.login))
	mux.Handle("/logout", secureMiddleware.ThenFunc(app.logout))
	mux.Handle("/submit", secureMiddleware.Append(app.requireAuth).ThenFunc(app.submit))
	mux.Handle("/register", secureMiddleware.ThenFunc(app.register))

	// h1 := app.logger(mux)  	// you can nest the middleware or keep adding one at a time
	// h2 := app.recover(h1)

	// While we could use alice to chain all these, its not much different than below chaining.
	// Instead lets use alice to separate our middleware and use based on different handler needs
	// chain := alice.New(app.recover, app.logger, app.session.Enable).Then(mux)

	// outermost (recover) runs first
	// handler := app.recover(app.logger(app.session.Enable(mux)))
	return defaultMiddleware.Then(mux)
}
