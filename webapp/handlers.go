package main

import (
	"net/http"
	repo "webapp/repository"
)

const (
	loggedInUserKey = "logged_in_user_id"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {

	filter := repo.Filter{
		Page:     1,
		PageSize: 10,
	}

	posts, metadata, err := app.post.GetAll(filter)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	app.render(w, r, "index.html", &templateData{
		Posts:    posts,
		Metadata: metadata,
	})
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "about.html", nil)
}

func (app *application) login(w http.ResponseWriter, r *http.Request) {
	if app.isAuthenticated(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}

	// app.session.GetString(r, loggedInUserKey)
	app.infoLog.Printf("Logging in: %s", app.session.GetString(r, loggedInUserKey))

	// if user submitted POST request, we need to take data and log them in
	// app.session.Put(r, "userID", "Bobby")
	if r.Method == http.MethodPost {
		// pass form from the request
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// Option 1: Form.Get to get the values
		// r.Form.Get("email")

		// Our custom Form wrapper
		form := NewForm(r.PostForm)
		form.Required("email", "password").
			MaxLength("email", 255).
			MaxLength("password", 255).
			MinLength("email", 3).
			MinLength("password", 6).
			IsEmail("email")

		if !form.Valid() {
			form.Errors.Add("generic", "The data you submitted was not valid")
			app.errorLog.Printf("Invalid form: %+v", form.Errors)
			app.render(w, r, "login.html", &templateData{
				Form: form,
			})
			return
		}

		// Option 2: FormValue to get the values
		email := r.FormValue("email")
		password := r.FormValue("password")

		_, err := app.user.Authenticate(email, password)

		if err != nil {
			form.Errors.Add("generic", "Wrong email or password")
			app.render(w, r, "login.html", &templateData{
				Form: form,
			})
			return
		}

		// otherwise we are logged in
		app.session.Put(r, loggedInUserKey, email)
		app.session.Put(r, "flash", "You are logged in")

		app.infoLog.Println("Logged in ")

		http.Redirect(w, r, "/submit", http.StatusSeeOther)
		return
	}

	// If it was a GET request, just take to login page
	app.render(w, r, "login.html", &templateData{
		Form: NewForm(r.PostForm),
	})
}

func (app *application) logout(w http.ResponseWriter, r *http.Request) {

	app.session.Remove(r, loggedInUserKey)
	app.session.Put(r, "flash", "You are logged out")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) register(w http.ResponseWriter, r *http.Request) {
	if app.isAuthenticated(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}

	if r.Method == http.MethodPost {

		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		// Our custom Form wrapper
		form := NewForm(r.PostForm)
		form.Required("email", "password", "name").
			MaxLength("email", 255).
			MaxLength("password", 255).
			MinLength("email", 3).
			MinLength("password", 6).
			MinLength("name", 3).
			IsEmail("email")

		if !form.Valid() {
			form.Errors.Add("generic", "Could not register due to invalid form")
			app.errorLog.Printf("Invalid form: %+v", form.Errors)
			app.render(w, r, "register.html", &templateData{
				Form: form,
			})
			return
		}

		// Option 2: FormValue to get the values
		email := r.FormValue("email")
		password := r.FormValue("password")
		name := r.FormValue("name")
		avatar := r.FormValue("avatar")

		_, err := app.user.CreateUser(name, email, password, avatar)

		// if we fail to auth the user, render login page w/ empty form?
		if err != nil {
			//TODO: need to display error of wrong email/password
			form.Errors.Add("generic", "Could not register the user")
			app.render(w, r, "register.html", &templateData{
				Form: form,
			})
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	app.render(w, r, "register.html", &templateData{
		Form: NewForm(r.PostForm),
	})
}

func (app *application) contact(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "contact.html", nil)
}

func (app *application) submit(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodPost {
		form := NewForm(r.PostForm)
		form.Required("title", "url").
			MaxLength("title", 255).
			MaxLength("url", 255).
			MinLength("url", 3)

		if !form.Valid() {
			form.Errors.Add("generic", "The data you submitted was not valid")
			app.render(w, r, "submit.html", &templateData{
				Form: form,
			})
			return
		}

		// If form valid, get details
		title := r.FormValue("title")
		url := r.FormValue("url")
		user := app.getUserFromContext(r.Context())
		id, err := app.post.CreatePost(title, url, user.ID)
		if err != nil {
			form.Errors.Add("generic", "Could not submit when creating post")
			app.render(w, r, "submit.html", &templateData{
				Form: form,
			})
			return
		}

		app.session.Put(r, "flash", "post created")
		app.infoLog.Printf("post created with id: %d", id)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	app.render(w, r, "submit.html", &templateData{
		Form: NewForm(r.PostForm), // you can use r.Form to read url and body data, r.PostForm only reads body data, small difference
	})
}
