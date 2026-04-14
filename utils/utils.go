package utils

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func ConnectDB(name string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", name)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil

}
