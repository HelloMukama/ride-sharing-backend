package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"strings"
	
)

func TestLoginHandler(t *testing.T) {
	src.TestSetup()
	
	// Setup
	os.Setenv("JWT_SECRET", "testsecret")

	InitRedis()
	InitDB()
	initAuth()

	reqBody := `{"username":"testuser","user_id":123,"role":"rider"}`
	req, err := http.NewRequest("POST", "/auth/login", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(loginHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}

	if !strings.Contains(rr.Body.String(), "token") {
		t.Errorf("Response missing token")
	}
}
