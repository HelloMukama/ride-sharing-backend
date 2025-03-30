package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

    "context" // Add this
    "fmt" // Add this
    "time"

    "github.com/google/uuid" // Add this
	"github.com/gorilla/mux"
)

type RideRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type RideResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Driver struct {
	ID    string  `json:"id"`
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	Dist  float64 `json:"dist"`
}

func rideStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rideID := vars["id"]

	claims, ok := r.Context().Value("userClaims").(*Claims)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid auth claims"})
		return
	}

	var status RideStatus
	err := dbPool.QueryRow(context.Background(),
		`SELECT id, driver_id, rider_id, status, created_at, updated_at
		 FROM rides WHERE id = $1 AND (rider_id = $2 OR driver_id = $3)`,
		rideID, claims.UserID, claims.Username).Scan(
		&status.ID, &status.DriverID, &status.RiderID,
		&status.Status, &status.CreatedAt, &status.UpdatedAt)

	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "ride not found"})
		return
	}

	respondJSON(w, http.StatusOK, status)
}

func requestRideHandler(w http.ResponseWriter, r *http.Request) {
    // Authentication and validation (unchanged)
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        respondJSON(w, http.StatusUnauthorized, RideResponse{
            Success: false,
            Error:   "Authorization header required",
        })
        return
    }

    authParts := strings.Split(authHeader, " ")
    if len(authParts) != 2 || authParts[0] != "Bearer" {
        respondJSON(w, http.StatusUnauthorized, RideResponse{
            Success: false,
            Error:   "Invalid Authorization header format",
        })
        return
    }

    claims, err := validateToken(authParts[1])
    if err != nil {
        log.Printf("Token validation failed: %v", err)
        respondJSON(w, http.StatusUnauthorized, RideResponse{
            Success: false,
            Error:   "Invalid authentication token",
        })
        return
    }

    var req RideRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondJSON(w, http.StatusBadRequest, RideResponse{
            Success: false,
            Error:   "Invalid request format",
        })
        return
    }

    // Driver matching with Redis GEO
    driverIDs, err := FindNearbyDrivers(req.Lat, req.Lng, 5)
    if err != nil {
        log.Printf("Driver search error: %v", err)
        respondJSON(w, http.StatusInternalServerError, RideResponse{
            Success: false,
            Error:   "Driver search temporarily unavailable",
        })
        return
    }

    if len(driverIDs) == 0 {
        respondJSON(w, http.StatusOK, RideResponse{
            Success: false,
            Message: "No drivers available. Please try again later.",
        })
        return
    }

    // Create minimal ride record in Redis (per instructions)
    rideID := uuid.New().String()
    err = redisClient.SetEX(context.Background(),
        fmt.Sprintf("ride:%s", rideID),
        fmt.Sprintf(`{"driver_id":"%s","rider_id":%d,"status":"accepted"}`, 
            driverIDs[0], claims.UserID),
        2*time.Hour).Err()

    if err != nil {
        log.Printf("Failed to create ride record: %v", err)
        respondJSON(w, http.StatusInternalServerError, RideResponse{
            Success: false,
            Error:   "Failed to create ride",
        })
        return
    }

    // Real-time notification via WebSocket (Bonus feature)
    if err := NotifyDriver(driverIDs[0], rideID); err != nil {
        log.Printf("WebSocket notification failed: %v", err)
        // Continue since this is a bonus feature
    }

    respondJSON(w, http.StatusOK, RideResponse{
        Success: true,
        Data: map[string]interface{}{
            "ride_id":   rideID,
            "driver_id": driverIDs[0],
            "status":    "accepted",
        },
    })
}

func listDriversHandler(w http.ResponseWriter, r *http.Request) {
	// Mock driver data
	mockDrivers := []Driver{
		{ID: "driver1", Lat: 0.3135, Lng: 32.5811},
		{ID: "driver2", Lat: 0.3167, Lng: 32.5825},
		{ID: "driver3", Lat: 0.3150, Lng: 32.5800},
	}

	respondJSON(w, http.StatusOK, mockDrivers)
}
