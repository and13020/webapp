package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredential = errors.New("invalid credential")

type UserRepository interface {
	CreateUser(name, email, password, avatar string) (int, error)
	CreateUsers(users []User) error
	GetUsersByName(name string) ([]User, error)
	GetUserByEmail(email string) (*User, error)
	Authenticate(email, plainPassword string) (int, error)
}

type SQLUserRepository struct {
	DB *sql.DB
}

func NewSQLUserRepository(db *sql.DB) *SQLUserRepository {
	return &SQLUserRepository{DB: db}
}

func (r *SQLUserRepository) CreateUser(name, email, plainPassword, avatar string) (int, error) {
	ctx := context.Background()

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	hashPass, err := HashPassword(plainPassword)
	if err != nil {
		return 0, err
	}

	// rollback here - if all goes well and we commit at the end, this will do nothing
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO users (name, email, hashed_password) VALUES (?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	// sanitize inputs to prevent SQL injection - using ? placeholders and passing args separately will handle this for us
	// Postgres will use #1, #2, etc instead of ? placeholders
	result, err := stmt.Exec(name, email, hashPass)
	if err != nil {
		fmt.Println("Error executing statement:", err)
		return 0, err
	}

	// lets also do other operations in same function to show atomicity of transactions - if any of these fail, the whole transaction will be rolled back and no partial data will be left in the database
	// e.g. we want to create a profile for the user we just created, but if that fails, we don't want the user to be created without a profile (orphaned user)
	pStmt, err := tx.PrepareContext(ctx, `INSERT INTO profile (user_id, avatar) VALUES (?, ?)`)
	if err != nil {
		return 0, err
	}
	defer pStmt.Close()

	userID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	_, err = pStmt.Exec(userID, "default_avatar.png")
	if err != nil {
		return 0, err
	}

	fmt.Println("User and profile created successfully with user ID:", userID)
	tx.Commit()
	return int(userID), nil
}

func (r *SQLUserRepository) CreateUsers(users []User) error {

	begin := `BEGIN TRANSACTION;`
	insert := `INSERT INTO users (name, email, hashed_password) VALUES (?, ?, ?);`
	commit := `COMMIT;`

	txn, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	stmt, err := txn.Prepare(insert)
	if err != nil {
		return fmt.Errorf("1: %w", err)
	}
	defer stmt.Close()

	fmt.Println(time.Now(), "--- Starting bulk insert of users...")

	for i, user := range users {

		// execute insert for each user
		_, err = txn.Stmt(stmt).Exec(user.Name, user.Email, user.HashedPassword)
		if err != nil {
			fmt.Printf("%s, %s, %s\n", user.Name, user.Email, user.HashedPassword)
			return fmt.Errorf("1. error creating user %s: %w", user.Email, err)
		}

		if i%100 == 99 { // commit every 100 users to avoid long transactions
			_, err = txn.Exec(commit)
			if err != nil {
				return fmt.Errorf("2. error committing transaction: %w", err)
			}
			_, err = txn.Exec(begin)
			if err != nil {
				return fmt.Errorf("3. error beginning new transaction: %w", err)
			}
			fmt.Println(time.Now(), "Inserted 100 users...")
		}

	}
	// Post loop commit any remaining users
	_, err = txn.Exec(commit)
	if err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}
	fmt.Println(time.Now(), "Finished bulk insert of users...")

	fmt.Println("All users added successfully!")
	return nil
}

func (r *SQLUserRepository) GetUsersByName(name string) ([]User, error) {
	pstmt, err := r.DB.Prepare(`SELECT id, name, email, hashed_password, created_at FROM users WHERE name LIKE ?`)
	if err != nil {
		return nil, err
	}
	defer pstmt.Close()

	rows, err := pstmt.Query("%" + name + "%") // use % wildcards to match any name containing the search term
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 64)
	var user User

	for rows.Next() {
		err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.HashedPassword, &user.CreatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return users, nil
}

func (r *SQLUserRepository) GetUserByEmail(email string) (*User, error) {
	stmt, err := r.DB.Prepare(`SELECT id, name, email, hashed_password, created_at FROM users WHERE email = ?`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(email)
	var user User
	err = row.Scan(&user.ID, &user.Name, &user.Email, &user.HashedPassword, &user.CreatedAt)
	if err != nil {
		fmt.Println("ERROR in row scan: ", err)
		return nil, err
	}
	return &user, nil
}

func (r *SQLUserRepository) Authenticate(email, plainPassword string) (int, error) {
	user, err := r.GetUserByEmail(email)
	if err != nil {
		return 0, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(plainPassword))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredential
		}
		return 0, err
	}
	return user.ID, nil
}
