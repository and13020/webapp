package common

import (
	"log"

	repo "webapp/repository"

	"github.com/golangcollege/sessions"
)

type Application struct {
	ErrorLog   *log.Logger
	InfoLog    *log.Logger
	User       repo.UserRepository
	Post       repo.PostRepository
	TmplDir    string
	Tp         *TemplateRenderer
	PublicPath string
	Session    *sessions.Session
}
