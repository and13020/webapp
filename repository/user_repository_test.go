package repository_test

import (
	"errors"
	"testing"

	r "webapp/repository"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var stdmail = "email@email.com"
var stdpass = "password"
var stdname = "John Doe"

var user = r.User{
	Name:           stdname,
	Email:          stdmail,
	HashedPassword: stdpass,
	Profile: r.Profile{
		Avatar: "default_avatar.png",
	},
}

func TestSQLUserRepository_CreateUser(t *testing.T) {

	repo := r.NewSQLUserRepository(db)

	// expect rollback if failed

	uID, err := repo.CreateUser(stdname, stdmail, stdpass, "avatar")

	assert.NoError(t, err)
	assert.Greater(t, uID, 0)

}

func TestSQLUserRepository_CreateUser_DuplicateEmail(t *testing.T) {

	repo := r.NewSQLUserRepository(db)

	uID, err := repo.CreateUser(stdname, stdmail, stdpass, "avatar")
	assert.NoError(t, err)
	assert.Greater(t, uID, 0)

	_, err = repo.CreateUser("John Doe2", stdmail, "password2", "avatar2")
	assert.Error(t, err)

}

func TestSQLUserRepository_CreateUser_Tx(t *testing.T) {

	repo := r.NewSQLUserRepository(mockDB)

	result := sqlmock.NewResult(1, 1)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO users`)
	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.Name, user.Email, user.HashedPassword).
		WillReturnResult(result)

	mock.ExpectPrepare(`INSERT INTO profile`)
	mock.ExpectExec("INSERT INTO profile").
		WithArgs(1, user.Profile.Avatar).
		WillReturnResult(result)
	mock.ExpectCommit()

	_, err := repo.CreateUser(user.Name, user.Email, user.HashedPassword, "avatar")
	if err != nil {
		t.Errorf("\nERROR creating user: %s: ERROR: %s", user.Name, err)
	}

}

func TestSQLUserRepository_CreateUser_TxFail(t *testing.T) {

	repo := r.NewSQLUserRepository(mockDB)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO users`) // accepts regex
	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.Name, user.Email, user.HashedPassword).
		WillReturnError(errors.New("duplicate user detected"))

	_, err := repo.CreateUser(user.Name, user.Email, user.HashedPassword, "avatar")
	assert.Error(t, err)
}

func TestSQLUserRepository_CreateUser_TxUserPrepareFail(t *testing.T) {

	repo := r.NewSQLUserRepository(mockDB)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO users`).
		WillReturnError(errors.New("Table does not exist"))

	_, err := repo.CreateUser(user.Name, user.Email, user.HashedPassword, "avatar")
	assert.Error(t, err)
}

func TestSQLUserRepository_CreateUser_TxBeginFail(t *testing.T) {

	repo := r.NewSQLUserRepository(mockDB)

	mock.ExpectBegin().
		WillReturnError(errors.New("Table does not exist"))

	_, err := repo.CreateUser(user.Name, user.Email, user.HashedPassword, "avatar")
	assert.Error(t, err)
}

func TestSQLUserRepository_CreateUser_TxProfPrepareFail(t *testing.T) {

	repo := r.NewSQLUserRepository(mockDB)

	result := sqlmock.NewResult(1, 1)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO users`)
	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.Name, user.Email, user.HashedPassword).
		WillReturnResult(result)
	mock.ExpectPrepare(`INSERT INTO profile`).
		WillReturnError(errors.New("Table does not exist"))

	_, err := repo.CreateUser(user.Name, user.Email, user.HashedPassword, "avatar")
	assert.Error(t, err)
}

// TODO: make tests for error statements
func TestSQLUserRepository_CreateUser_Result(t *testing.T) {

	repo := r.NewSQLUserRepository(db)

	result := sqlmock.NewResult(1, 1)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO users`)
	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.Name, user.Email, "").
		WillReturnResult(result)

	mock.ExpectPrepare(`INSERT INTO profile`)
	mock.ExpectExec("INSERT INTO profile").
		WithArgs(1, user.Profile.Avatar).
		WillReturnResult(result)
	mock.ExpectCommit()

	_, err := repo.CreateUser(user.Name, user.Email, "", "avatar")
	assert.NoError(t, err)

	userID, err := result.LastInsertId()
	assert.Equal(t, userID, int64(1))
}

// Err is not covered for result.LastInsertId()
func TestSQLUserRepository_CreateUser_LastInsertIdFail(t *testing.T) {

	repo := r.NewSQLUserRepository(mockDB)

	result := sqlmock.NewResult(-1, 1)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO users`)
	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.Name, user.Email, user.HashedPassword).
		WillReturnResult(result)

	_, err := repo.CreateUser(user.Name, user.Email, user.HashedPassword, "avatar")
	assert.Error(t, err)

	userID, err := result.LastInsertId()
	assert.NotEqual(t, 1, int(userID))
}

func TestSQLUserRepository_CreateUser_TxProfExecFail(t *testing.T) {

	repo := r.NewSQLUserRepository(mockDB)

	result := sqlmock.NewResult(1, 1)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO users`)
	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.Name, user.Email, user.HashedPassword).
		WillReturnResult(result)

	mock.ExpectPrepare(`INSERT INTO profile`)
	mock.ExpectExec("INSERT INTO profile").
		WithArgs(1, user.Profile.Avatar).
		WillReturnError(errors.New("duplicate profile id"))

	_, err := repo.CreateUser(user.Name, user.Email, user.HashedPassword, "avatar")
	assert.Error(t, err)
}

func TestSQLUserRepository_Authenticate_Success(t *testing.T) {

	repo := r.NewSQLUserRepository(db)

	uID, err := repo.CreateUser(stdname, stdmail, stdpass, "avatar")

	assert.NoError(t, err)
	assert.Greater(t, uID, 0)

	authUserID, err := repo.Authenticate(stdmail, stdpass)
	assert.NoError(t, err)
	assert.Equal(t, uID, authUserID)
}

func TestSQLUserRepository_Authenticate_WrongPassword(t *testing.T) {

	repo := r.NewSQLUserRepository(db)

	uID, err := repo.CreateUser(stdname, stdmail, stdpass, "avatar")

	assert.NoError(t, err)
	assert.Greater(t, uID, 0)

	_, err = repo.Authenticate(stdmail, "wrongpassword")
	assert.Error(t, err)
	assert.Equal(t, r.ErrInvalidCredential, err)
}
