package main

import (
	"net/http"
)

// render is a helper method to render HTML templates. It takes the response writer, the filename of the template, and any data needed for dynamic content. It constructs the full path to the template file, parses it, and executes it, writing the output to the response writer.
func (app *application) render(w http.ResponseWriter, r *http.Request, filename string, data *templateData) {
	if app.tp == nil {
		http.Error(w, "template rendering engine is not set", http.StatusInternalServerError)
	}
	app.tp.Render(w, filename, app.defaultTemplateData(data, r))
}

func (app *application) defaultTemplateData(data *templateData, r *http.Request) *templateData {
	if data == nil {
		data = &templateData{}
	}
	data.Flash = app.session.PopString(r, "flash")
	data.IsAuthenticated = app.isAuthenticated(r)

	return data
}
