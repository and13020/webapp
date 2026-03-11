package main

import (
	"fmt"
	"html/template"
	"net/http"
	"path"
	"path/filepath"
	"sync"

	postRepo "webapp/repository"
)

type TemplateRenderer struct {
	cache       map[string]*template.Template
	mutex       sync.RWMutex
	dev         bool
	templateDir string
}

// Template Data should consist of anything we want while creating templates to display on the html pages
// Including forms, auth status, flashes, user posts or any metadata
// Our html will refer to these template fields when rendering
type templateData struct {
	Form            *Form
	IsAuthenticated bool
	Flash           string
	Posts           []postRepo.Post
	Metadata        postRepo.Metadata
	Comments        []postRepo.Comment
	Post            *postRepo.Post
	NextLink        string
	PrevLink        string
}

func NewTemplateRenderer(templateDir string, isDev bool) *TemplateRenderer {
	return &TemplateRenderer{
		cache:       make(map[string]*template.Template),
		dev:         isDev,
		templateDir: templateDir,
	}
}

func (tr *TemplateRenderer) Render(w http.ResponseWriter, tmplName string, data any) {

	tmpl, err := tr.getTemplate(tmplName)
	if err != nil {
		fmt.Println("Encountered renderer issue: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "base.html", data)
	if err != nil {
		fmt.Println("Encountered renderer issue 2: ", err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (tr *TemplateRenderer) getTemplate(filename string) (*template.Template, error) {

	// check cache, if it exists return it
	tr.mutex.RLock()
	if tmpl, exists := tr.cache[filename]; exists {
		tr.mutex.RUnlock()
		return tmpl, nil
	}
	tr.mutex.RUnlock()

	// if template not in cache, store in cache
	tr.mutex.Lock()
	tmpl, err := tr.parseTemplate(filename)
	if err != nil {
		tr.mutex.Unlock()
		return nil, err
	}

	tr.cache[filename] = tmpl
	tr.mutex.Unlock()

	return tmpl, nil
}

func (tr *TemplateRenderer) parseTemplate(filename string) (*template.Template, error) {
	// normally we could just return the template for given filename
	// but in this case we are returning ALL the files within layouts and partials folders as well as given filename
	// so we load the cache with all the templates basically to avoid coming back to this
	p := path.Join(tr.templateDir, filename)

	files := []string{p}

	layoutPath := path.Join(tr.templateDir, "layouts", "*.html")
	layouts, err := filepath.Glob(layoutPath)
	if err == nil {
		files = append(files, layouts...)
	}

	partialPath := path.Join(tr.templateDir, "partials", "*.html")
	partials, err := filepath.Glob(partialPath)
	if err == nil {
		files = append(files, partials...)
	}

	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}
