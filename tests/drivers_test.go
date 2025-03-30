package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListDrivers(t *testing.T) {
	// Cache a mock driver
	CacheDriverLocation("driver1", 0.3135, 32.5811)

	req, _ := http.NewRequest("GET", "/drivers", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(listDriversHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "driver1") {
		t.Errorf("Driver not listed")
	}
}
