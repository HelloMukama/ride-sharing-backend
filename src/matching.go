// src/matching.go
package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type RideRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Driver struct {
	ID    string  `json:"id"`
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	Dist  float64 `json:"dist"`
}

func rideStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ride_id": vars["id"],
		"status":  "in_progress",
	})
}

func requestRideHandler(w http.ResponseWriter, r *http.Request) {
	// Validate token
	_, err := validateToken(r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req RideRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find nearby drivers (within 5km)
	driverIDs, err := FindNearbyDrivers(req.Lat, req.Lng, 5)
	if err != nil {
		http.Error(w, "Error finding drivers", http.StatusInternalServerError)
		return
	}

	if len(driverIDs) == 0 {
		http.Error(w, "No drivers available", http.StatusNotFound)
		return
	}

	// In a real app, we'd implement more sophisticated matching logic
	matchedDriver := driverIDs[0]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"driver_id": matchedDriver,
		"status":    "matched",
	})
}

func listDriversHandler(w http.ResponseWriter, r *http.Request) {
	// This is a mock implementation - in a real app, you'd get real driver data
	mockDrivers := []Driver{
		{ID: "driver1", Lat: 0.3135, Lng: 32.5811},
		{ID: "driver2", Lat: 0.3167, Lng: 32.5825},
		{ID: "driver3", Lat: 0.3150, Lng: 32.5800},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mockDrivers)
}
