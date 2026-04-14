package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogin_GET_NotAuthenticated(t *testing.T) {
	defer resetDB()

	// Wrap with session.Enable whenever we need session data
	// Wrap w/ authenticate for login
	handler := testApp.session.Enable(testApp.authenticate(http.HandlerFunc(testApp.login)))

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestLogin_GET_AlreadyAuthenticated(t *testing.T) {
	defer resetDB()

	_, err := testApp.user.CreateUser(
		"John Doe",
		"test@login.com",
		"testpassword",
		"avatar",
	)
	assert.NoError(t, err)

	// SETUP HANDLER
	setupHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		testApp.session.Put(r, loggedInUserKey, "test@login.com") // current signed in user is not valid email
	})

	// Prepare then serve the SETUP handler w/ session enabled
	setupChain := testApp.session.Enable(setupHandler)
	req1 := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr1 := httptest.NewRecorder()
	setupChain.ServeHTTP(rr1, req1)

	// TEST HANDLER
	// Session middleware, auth middleware then login handler
	testHandler := testApp.session.Enable(testApp.authenticate(http.HandlerFunc(testApp.login)))
	req2 := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr2 := httptest.NewRecorder()
	// if we have any cookies, copy them into new request
	if cookies := rr1.Result().Cookies(); len(cookies) > 0 {
		for _, cookie := range cookies {
			req2.AddCookie(cookie)
		}
	}

	// Serve TEST handler
	testHandler.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusSeeOther, rr2.Code)
	assert.Equal(t, "/", rr2.Header().Get("Location"))
}

func TestLogin_POST_ValidCredentials(t *testing.T) {
	defer resetDB()

	_, err := testApp.user.CreateUser(
		"John Doe",
		"test@login.com",
		"testpassword",
		"avatar",
	)
	assert.NoError(t, err)

	// Create handler w/ session enabled, and auth middleware
	handler := testApp.session.Enable(testApp.authenticate(http.HandlerFunc(testApp.login)))

	// Create form data
	formData := "email=test@login.com&password=testpassword"

	// req body should accept a form, set header to change content type to form
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/submit", rr.Header().Get("location"))

}

func TestLogin_POST_InvalidForm(t *testing.T) {
	defer resetDB()

	// Create handler w/ session enabled, and auth middleware
	handler := testApp.session.Enable(testApp.authenticate(http.HandlerFunc(testApp.login)))

	// Create form data
	formData := "email=&password="

	// req body should accept a form, set header to change content type to form
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	b := rr.Body.String()

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, b, "data you submitted was not valid")
	assert.Contains(t, b, "email is required")
	assert.Contains(t, b, "password is required")
}

func TestLogin_POST_InvalidAuthenticationData(t *testing.T) {
	defer resetDB()

	// Create handler w/ session enabled, and auth middleware
	handler := testApp.session.Enable(testApp.authenticate(http.HandlerFunc(testApp.login)))

	// Create form data
	formData := "email=test@login.com&password=testpassword"

	// req body should accept a form, set header to change content type to form
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	b := rr.Body.String()

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, b, "Wrong email or password")
}
