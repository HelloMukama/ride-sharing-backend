package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestRide(t *testing.T) {
	// Mock Redis and DB
	redisClient = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer redisClient.Close()

	reqBody := `{"lat":0.3135,"lng":32.5811}`
	req, _ := http.NewRequest("POST", "/request-ride", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+os.Getenv("TEST_TOKEN"))

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(requestRideHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}
}
