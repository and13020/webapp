package common

import (
	"html/template"
	"sync"
)

type TemplateRenderer struct {
	Cache       map[string]*template.Template
	Mutex       sync.RWMutex
	Dev         bool
	TemplateDir string
}
