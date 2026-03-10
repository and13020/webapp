package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	repo "webapp/repository"
	utils "webapp/utils"

	"github.com/golangcollege/sessions"
)

type application struct {
	errorLog   *log.Logger
	infoLog    *log.Logger
	user       repo.UserRepository
	post       repo.PostRepository
	tmplDir    string
	tp         *TemplateRenderer
	publicPath string
	session    *sessions.Session
}

func main() {

	// start DB
	db, err := utils.ConnectDB("users_database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// initialize session
	ses := sessions.New([]byte("u46IpCV9y5Vlur8YvODJEhgOY8m9JVE4XXX"))
	ses.Lifetime = 24 * time.Hour
	ses.Secure = true
	ses.SameSite = http.SameSiteLaxMode

	app := &application{
		errorLog:   log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile),
		infoLog:    log.New(os.Stderr, "INFO\t", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile),
		user:       repo.NewSQLUserRepository(db),
		post:       repo.NewSQLPostRepository(db),
		tmplDir:    "./templates",
		publicPath: "./public/",
		session:    ses, // secret goes here
	}

	setupDB(db, app.user)

	app.tp = NewTemplateRenderer(app.tmplDir, true)

	err = app.Serve()
	if err != nil {
		log.Fatal(err)
	}

}

func setupDB(db *sql.DB, u repo.UserRepository) {
	go func() {
		tableNames := []string{UserSchema, ProfileSchema, PostsSchema, CommentsSchema, VotesSchema}
		generateTables(db, tableNames)
		err := generateUsers(u)
		if err != nil {
			fmt.Println(err)
		}
	}()
}

func generateUsers(u repo.UserRepository) error {
	fmt.Println("generating users")
	_, err := u.CreateUser("Jonas", "jonas123@gmail.com", "secret001", "a1.png")
	if err != nil {
		return err
	}
	_, err = u.CreateUser("Mindy", "starlightqueen@gmail.com", "secret002", "a2.png")
	if err != nil {
		return err
	}
	_, err = u.CreateUser("Thomas", "ttt@my.com", "secret003", "a3.png")
	if err != nil {
		return err
	}
	_, err = u.CreateUser("Sammy", "samster@gmail.com", "secret004", "a4.png")
	if err != nil {
		return err
	}
	_, err = u.CreateUser("Brittney", "brittneyrocks@gmail.com", "secret005", "a5.png")
	if err != nil {
		return err
	}
	fmt.Println("Finished generating users")
	return nil
}

func generateTables(db *sql.DB, tableNames []string) {
	for _, query := range tableNames {
		_, err := db.Exec(query)
		if err != nil {
			log.Fatal("Could not generate table")
		}

	}

	// _, err := db.Exec(UserSchema)
	// if err != nil {
	// 	log.Fatal("Could not generate table")
	// }

	// _, err = db.Exec(ProfileSchema)
	// if err != nil {
	// 	log.Fatal("Could not generate table")
	// }
}
