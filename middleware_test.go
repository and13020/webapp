package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	r "webapp/repository"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	defer resetDB()
	testHandler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

	buf := new(bytes.Buffer) // create a buffer to use as our infolog. we can view and assert later
	testApp.infoLog = log.New(buf, "", 0)

	h := testApp.loggger(testHandler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder() // rr stands for response writer

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code) // should get OK status
	assert.Equal(t, "OK", rr.Body.String())
	assert.Contains(t, buf.String(), "HTTP/1.1 GET /test")

}

func TestRecover(t *testing.T) {
	defer resetDB()
	testHandler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	// recover is meant to accept the next handler as input
	// returning a handler
	// It attempts to serve HTTP but if it fails will attempt to recover
	h := testApp.recover(testHandler)
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "close", rr.Header().Get("Connection"))
}

func contextWithAuth(ctx context.Context, isAuth any) context.Context {
	return context.WithValue(ctx, contextAuthKey, isAuth)
}

func TestRequireAuth(t *testing.T) {
	defer resetDB()
	testHandler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(contextWithAuth(req.Context(), true))
	rr := httptest.NewRecorder()

	h := testApp.requireAuth(testHandler)
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())
	assert.Equal(t, "no-cache", rr.Header().Get("Cache-Control"))
}

func TestRequireAuth_Fail(t *testing.T) {
	defer resetDB()
	testHandler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	h := testApp.requireAuth(testHandler)
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Contains(t, rr.Body.String(), "redirect")
}

func TestAuthenticate(t *testing.T) {
	defer resetDB()
	userID, err := testApp.user.CreateUser(
		"session name",
		"session@test.com",
		"sessionpassword",
		"avatar",
	)
	assert.NoError(t, err)
	assert.Greater(t, userID, 0)

	setupHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		testApp.session.Put(r, loggedInUserKey, "session@test.com")
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, testApp.isAuthenticated(r))
		user := testApp.getUserFromContext(r.Context())

		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "session@test.com", user.Email)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	setupChain := testApp.session.Enable(setupHandler) // handler can save/update session data
	req1 := httptest.NewRequest(http.MethodGet, "/setup", nil)
	w1 := httptest.NewRecorder()
	setupChain.ServeHTTP(w1, req1)

	testChain := testApp.session.Enable(testApp.authenticate(testHandler))
	req2 := httptest.NewRequest(http.MethodGet, "/setup", nil)

	// Add any cookies from req1 to req2
	if cookies := w1.Result().Cookies(); len(cookies) > 0 {
		for _, cookie := range cookies {
			req2.AddCookie(cookie)
		}
	}

	w2 := httptest.NewRecorder()
	testChain.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "OK", w2.Body.String())

}

func TestAuthenticate_InvalidSession(t *testing.T) {
	defer resetDB()

	userID, err := testApp.user.CreateUser(
		"session name",
		"session@test.com",
		"sessionpassword",
		"avatar",
	)
	assert.NoError(t, err)
	assert.Greater(t, userID, 0)

	// SETUP HANDLER
	setupHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// TEST HANDLER
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.False(t, testApp.isAuthenticated(r))

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("no session data"))
	})

	req1 := httptest.NewRequest(http.MethodGet, "/setup", nil)
	w1 := httptest.NewRecorder()
	testApp.session.Enable(setupHandler).ServeHTTP(w1, req1) // handler can save/update session data

	testChain := testApp.session.Enable(testApp.authenticate(testHandler))
	req2 := httptest.NewRequest(http.MethodGet, "/setup", nil)

	// Add any cookies from req1 to req2
	if cookies := w1.Result().Cookies(); len(cookies) > 0 {
		for _, cookie := range cookies {
			req2.AddCookie(cookie)
		}
	}

	rr2 := httptest.NewRecorder()
	testChain.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusInternalServerError, rr2.Code)
	assert.Equal(t, "no session data", rr2.Body.String())

	assert.Panics(t, func() { testApp.session.Exists(req2, loggedInUserKey) })
}

func TestAuthenticate_InvalidGetUserByEmail_ErrNoRows(t *testing.T) {
	defer resetDB()

	userID, err := testApp.user.CreateUser(
		"session name",
		"session@test.com",
		"sessionpassword",
		"avatar",
	)
	assert.NoError(t, err)
	assert.Greater(t, userID, 0)

	// SETUP HANDLER
	setupHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		testApp.session.Put(r, loggedInUserKey, "wrong@test.com") // current signed in user is not valid email
	})

	// TEST HANDLER
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.False(t, testApp.isAuthenticated(r))
	})

	req1 := httptest.NewRequest(http.MethodGet, "/setup", nil)
	w1 := httptest.NewRecorder()
	testApp.session.Enable(setupHandler).ServeHTTP(w1, req1) // handler can save/update session data

	testChain := testApp.session.Enable(testApp.authenticate(testHandler))
	req2 := httptest.NewRequest(http.MethodGet, "/setup", nil)

	// Add any cookies from req1 to req2
	if cookies := w1.Result().Cookies(); len(cookies) > 0 {
		for _, cookie := range cookies {
			req2.AddCookie(cookie)
		}
	}

	rr2 := httptest.NewRecorder()
	testChain.ServeHTTP(rr2, req2)
	assert.Panics(t, func() { testApp.session.Exists(req2, loggedInUserKey) })
}

func TestAuthenticate_InvalidGetUserByEmail_ServerError(t *testing.T) {
	defer resetDB()
	userID, err := testApp.user.CreateUser(
		"session name",
		"session@test.com",
		"sessionpassword",
		"avatar",
	)
	assert.NoError(t, err)
	assert.Greater(t, userID, 0)

	// SETUP HANDLER
	setupHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		testApp.session.Put(r, loggedInUserKey, "session@test.com") // current signed in user is not valid email
	})

	// TEST HANDLER
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.False(t, testApp.isAuthenticated(r))
	})

	req1 := httptest.NewRequest(http.MethodGet, "/setup", nil)
	w1 := httptest.NewRecorder()
	testApp.session.Enable(setupHandler).ServeHTTP(w1, req1) // handler can save/update session data

	// DB down
	cleanupTestTables(t)

	testChain := testApp.session.Enable(testApp.authenticate(testHandler))
	req2 := httptest.NewRequest(http.MethodGet, "/setup", nil)

	// Add any cookies from req1 to req2
	if cookies := w1.Result().Cookies(); len(cookies) > 0 {
		for _, cookie := range cookies {
			req2.AddCookie(cookie)
		}
	}

	rr2 := httptest.NewRecorder()
	testChain.ServeHTTP(rr2, req2)

	// indicates serverError()'s http.Error() was triggered
	assert.Equal(t, "nosniff", rr2.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "text/plain; charset=utf-8", rr2.Header().Get("Content-Type"))

}

func TestGetUserFromContext(t *testing.T) {

	testUser := r.User{
		ID:             1,
		Name:           "session name",
		Email:          "session@test.com",
		HashedPassword: "password",
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, contextUserKey, &testUser)

	user := testApp.getUserFromContext(ctx)

	assert.Equal(t, user.ID, testUser.ID)
	assert.Equal(t, user.Email, testUser.Email)
}

func TestGetUserFromContext_WithEmptyContext(t *testing.T) {
	defer resetDB()

	userID, err := testApp.user.CreateUser(
		"session name",
		"session@test.com",
		"sessionpassword",
		"avatar",
	)
	assert.NoError(t, err)
	assert.Greater(t, userID, 0)

	ctx := context.Background()

	assert.PanicsWithValue(t, "User could not be retrieved from given context", func() { testApp.getUserFromContext(ctx) })
}
