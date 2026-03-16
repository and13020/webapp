package main

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMain to test our DB connection
// useful to run things prior to any test case
// Notice it takes "m *testing.M" rather than T
// M is the main test execution context for the package main
// It has a method m.Run() which runs every test
func TestMain(m *testing.M) {
	fmt.Println("prior to testing")
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}

	if err := db.Ping(); err != nil {
		panic(err)
	}

	// Create setup tables
	if err = setupTestSchema(db); err != nil {
		panic(err)
	}

	_ = m.Run()

	cleanupTestTables(db, &testing.T{})
	db.Close()

	fmt.Println("after testing")

	//os.Exit(code)
}

func setupTestSchema(db *sql.DB) error {

	schema := `CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	email TEXT NOT NULL UNIQUE,
	hashed_password TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE IF NOT EXISTS profile (
	user_id INTEGER PRIMARY KEY REFERENCES users(id),
	avatar TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE IF NOT EXISTS posts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	url TEXT NOT NULL,
	title TEXT NOT NULL UNIQUE,
    user_id INTEGER REFERENCES users(user_id) ON DELETE CASCADE,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS comments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	body TEXT NOT NULL,
    post_id INTEGER REFERENCES posts(posts_id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(users_id) ON DELETE CASCADE,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE IF NOT EXISTS votes (
    post_id INTEGER REFERENCES posts(posts_id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(users_id) ON DELETE CASCADE,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, post_id)
);`

	_, err := db.Exec(schema)
	return err
}

func cleanupTestTables(db *sql.DB, t *testing.T) {
	// 	schema := `SET foreign_key_checks = 0;
	// DROP TABLE IF EXISTS users, profile, posts, comments, votes;
	// SET foreign_key_checks = 1;`
	tableNames := []string{
		"users",
		"profile",
		"posts",
		"comments",
		"votes",
	}

	for _, name := range tableNames {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", name))
		assert.NoError(t, err)
	}
}
