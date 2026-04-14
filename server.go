package main

import (
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func (app *application) Serve() error {
	s := http.Server{
		Addr:         ":8080",
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		Handler:      app.routes(),
	}
	return s.ListenAndServe()
}
