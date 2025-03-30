package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
    "context" // Add this

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

    // Start database transaction
    ctx := context.Background()
    tx, err := dbPool.Begin(ctx)
    if err != nil {
        log.Printf("Failed to begin transaction: %v", err)
        respondJSON(w, http.StatusInternalServerError, RideResponse{
            Success: false,
            Error:   "Internal server error",
        })
        return
    }
    defer tx.Rollback(ctx)

    // Find nearest available driver using PostGIS
    var driverID string
    err = tx.QueryRow(ctx,
        `UPDATE drivers 
         SET available = false 
         WHERE driver_id = (
             SELECT driver_id 
             FROM drivers 
             WHERE available = true
             ORDER BY current_location <-> ST_SetSRID(ST_MakePoint($1, $2), 4326)
             LIMIT 1
         )
         RETURNING driver_id`,
        req.Lng, req.Lat).Scan(&driverID)

    if err != nil {
        log.Printf("Driver search failed: %v", err)
        respondJSON(w, http.StatusNotFound, RideResponse{
            Success: false,
            Message: "No available drivers found",
        })
        return
    }

    // Create ride record in PostgreSQL
    var rideID string
    err = tx.QueryRow(ctx,
        `INSERT INTO rides (driver_id, rider_id, status, start_location)
         VALUES ($1, $2, 'requested', ST_SetSRID(ST_MakePoint($3, $4), 4326))
         RETURNING id`,
        driverID, claims.UserID, req.Lng, req.Lat).Scan(&rideID)

    if err != nil {
        log.Printf("Failed to create ride: %v", err)
        respondJSON(w, http.StatusInternalServerError, RideResponse{
            Success: false,
            Error:   "Failed to create ride record",
        })
        return
    }

    // Commit transaction
    if err := tx.Commit(ctx); err != nil {
        log.Printf("Transaction commit failed: %v", err)
        respondJSON(w, http.StatusInternalServerError, RideResponse{
            Success: false,
            Error:   "Failed to complete ride request",
        })
        return
    }

    // Real-time notification via WebSocket (Bonus feature)
    if err := NotifyDriver(driverID, rideID); err != nil {
        log.Printf("WebSocket notification failed: %v", err)
        // Continue since this is a bonus feature
    }

    respondJSON(w, http.StatusOK, RideResponse{
        Success: true,
        Data: map[string]interface{}{
            "ride_id":   rideID,
            "driver_id": driverID,
            "status":    "requested",
        },
    })
}

func listDriversHandler(w http.ResponseWriter, r *http.Request) {
    rows, err := dbPool.Query(r.Context(),
        `SELECT driver_id, 
        ST_X(current_location::geometry) as lat, 
        ST_Y(current_location::geometry) as lng 
        FROM drivers WHERE available = true`)
    if err != nil {
        respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Database error"})
        return
    }
    defer rows.Close()

    var drivers []Driver
    for rows.Next() {
        var d Driver
        if err := rows.Scan(&d.ID, &d.Lat, &d.Lng); err != nil {
            respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Data parsing error"})
            return
        }
        drivers = append(drivers, d)
    }

    respondJSON(w, http.StatusOK, drivers)
}
