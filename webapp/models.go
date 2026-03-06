package main

import "time"

var UserSchema = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	email TEXT NOT NULL UNIQUE,
	hashed_password TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

var ProfileSchema = `
CREATE TABLE IF NOT EXISTS profile (
	user_id INTEGER PRIMARY KEY references users(id),
	avatar TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

var PostsSchema = `CREATE TABLE posts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	url TEXT NOT NULL,
	title TEXT NOT NULL UNIQUE,
    user_id INTEGER REFERENCES users(user_id) ON DELETE CASCADE,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

var CommentsSchema = `CREATE TABLE comments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	body TEXT NOT NULL,
	title TEXT NOT NULL UNIQUE,
    post_id INTEGER REFERENCES posts(posts_id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(users_id) ON DELETE CASCADE,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

var VotesSchema = `CREATE TABLE votes (
    post_id INTEGER REFERENCES posts(posts_id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(users_id) ON DELETE CASCADE,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, post_id)
);`

type User struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	HashedPassword string    `json:"-"`
	Profile        Profile   `json:"profile"`
	CreatedAt      time.Time `json:"created"`
}

type Profile struct {
	ProfileID int64     `json:"id"`
	Avatar    string    `json:"avatar"`
	Created   time.Time `json:"created"`
}
