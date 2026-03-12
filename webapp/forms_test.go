package main

import (
	"net/url"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewForm(t *testing.T) {
	v := make(url.Values)
	v.Add("email", "test@test.com")

	f := NewForm(v)

	assert.NotNil(t, f)
	assert.Equal(t, "test@test.com", f.Get("email"))
	assert.NotNil(t, f.Errors)
	assert.Len(t, f.Errors, 0)
}

func TestRequired(t *testing.T) {
	v := make(url.Values)
	v.Add("email", "test@test.com")
	v.Add("empty", "   ")

	f := NewForm(v)
	f.Required("email", "password", "empty")

	assert.NotNil(t, f)
	assert.Equal(t, "", f.Errors.Get("email"))
	assert.Contains(t, f.Errors.Get("password"), "password is required")
	assert.Contains(t, f.Errors.Get("empty"), "empty is required")
}

func TestValid(t *testing.T) {
	v := make(url.Values)
	v.Add("email", "test@test.com")

	f := NewForm(v)
	isValid := f.Valid()

	assert.True(t, isValid)
}

func TestValidFalse(t *testing.T) {
	v := make(url.Values)

	f := NewForm(v)
	f.Errors.Add("Username", "Testing - username isn't valid")
	isValid := f.Valid()

	assert.False(t, isValid)
}

func TestMaxLength(t *testing.T) {

	v := make(url.Values)
	v.Add("email", "thisIsTooLong")
	v.Add("password", "secret")
	v.Add("empty", "")

	f := NewForm(v)

	f.MaxLength("email", 10)
	f.MaxLength("password", 20)
	f.MaxLength("empty", 20)

	assert.Len(t, f.Errors, 1)
	assert.Contains(t, f.Errors.Get("email"), "This field email is too long.")
	assert.Contains(t, f.Errors.Get("password"), "")
	assert.Contains(t, f.Errors.Get("empty"), "")

}

func TestMinLength(t *testing.T) {
	v := make(url.Values)
	v.Add("email", "tooShort")
	v.Add("password", "longValidPassword")
	v.Add("empty", "")

	f := NewForm(v)

	f.MinLength("email", 20)
	f.MinLength("password", 5)
	f.MinLength("empty", 5)

	assert.Len(t, f.Errors, 1)
	assert.Contains(t, f.Errors.Get("email"), "email is too short")
	assert.Contains(t, f.Errors.Get("password"), "")
	assert.Contains(t, f.Errors.Get("empty"), "")
}

func TestMatches(t *testing.T) {

	pattern, _ := regexp.Compile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	v := make(url.Values)
	v.Add("apples", "delicious")
	v.Add("email", "Thomas@Tank.Engine")
	v.Add("empty", "")

	f := NewForm(v)

	f.Matches("apples", pattern)
	f.Matches("email", pattern)
	f.Matches("empty", pattern)

	assert.Len(t, f.Errors, 1)
	assert.Contains(t, f.Errors.Get("apples"), "field apples is invalid")
	assert.Contains(t, f.Errors.Get("email"), "")
	assert.Contains(t, f.Errors.Get("empty"), "")

}

func TestIsEmail(t *testing.T) {
	v := make(url.Values)
	v.Add("apples", "delicious")
	v.Add("email", "Thomas@Tank.Engine")
	v.Add("empty", "")

	f := NewForm(v)

	f.IsEmail("apples")
	f.IsEmail("email")
	f.IsEmail("empty")

	assert.Len(t, f.Errors, 1)
	assert.Contains(t, f.Errors.Get("apples"), "field apples is not a valid email address")
	assert.Contains(t, f.Errors.Get("email"), "")
	assert.Contains(t, f.Errors.Get("empty"), "")

}
