package main

import (
    "context"
    "encoding/json"
    "log"
    "math"
    "net/http"
    "strings"
    "time"
    "errors"

    "github.com/gorilla/mux"
    "github.com/redis/go-redis/v9"
    "github.com/jackc/pgx/v5"
)

type RideRequest struct {
    PickupLat  float64 `json:"lat"`
    PickupLng  float64 `json:"lng"`
    DropoffLat float64 `json:"dropoff_lat,omitempty"`
    DropoffLng float64 `json:"dropoff_lng,omitempty"`
    VehicleType string `json:"vehicle_type,omitempty"`
}

type RideResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Message string      `json:"message,omitempty"`
    Error   string      `json:"error,omitempty"`
}

type Driver struct {
    ID       string  `json:"id"`
    Lat      float64 `json:"lat"`
    Lng      float64 `json:"lng"`
    Dist     float64 `json:"dist,omitempty"`
    Name     string  `json:"name,omitempty"`
    Rating   float64 `json:"rating,omitempty"`
    Vehicle  string  `json:"vehicle,omitempty"`
    ETA      int     `json:"eta,omitempty"` // in minutes
}

const (
    maxMatchingAttempts = 3
    searchRadiusKm      = 5.0
    baseFare           = 5.0
    pricePerKm         = 1.5
)

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
        `SELECT id, driver_id, rider_id, status, price_estimate, estimated_eta, created_at, updated_at
         FROM rides WHERE id = $1 AND (rider_id = $2 OR driver_id = $3)`,
        rideID, claims.UserID, claims.Username).Scan(
        &status.ID, &status.DriverID, &status.RiderID,
        &status.Status, &status.Price, &status.ETA,
        &status.CreatedAt, &status.UpdatedAt)

    if err != nil {
        respondJSON(w, http.StatusNotFound, map[string]string{"error": "ride not found"})
        return
    }

    respondJSON(w, http.StatusOK, status)
}

func requestRideHandler(w http.ResponseWriter, r *http.Request) {
    // Authentication
    claims, err := validateRequest(r)
    if err != nil {
        respondJSON(w, http.StatusUnauthorized, errorResponse(err.Error()))
        return
    }

    // Request parsing
    var req RideRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondJSON(w, http.StatusBadRequest, errorResponse("Invalid request format"))
        return
    }

    // Validate coordinates
    if !validCoordinates(req.PickupLat, req.PickupLng) {
        respondJSON(w, http.StatusBadRequest, errorResponse("Invalid coordinates"))
        return
    }

    // Find and assign driver
    result, err := matchDriver(claims.UserID, req)
    if err != nil {
        log.Printf("Ride matching failed: %v", err)
        respondJSON(w, http.StatusServiceUnavailable, errorResponse(err.Error()))
        return
    }

    // Cache driver location
    go cacheDriverLocation(result.DriverID, req.PickupLat, req.PickupLng)

    respondJSON(w, http.StatusOK, successResponse(result))
}

func matchDriver(riderID int, req RideRequest) (*RideStatus, error) {
    ctx := context.Background()
    tx, err := dbPool.Begin(ctx)
    if err != nil {
        return nil, errors.New("failed to start transaction")
    }
    defer tx.Rollback(ctx)

    var match *RideStatus
    var lastErr error

    // Try multiple times to find a driver
    for attempt := 0; attempt < maxMatchingAttempts; attempt++ {
        match, lastErr = findNearestDriver(ctx, tx.(pgx.Tx), riderID, req)
        if lastErr == nil {
            break
        }
        time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
    }

    if err := tx.Commit(ctx); err != nil {
        return nil, errors.New("failed to commit transaction")
    }

    return match, nil
}

func findNearestDriver(ctx context.Context, tx pgx.Tx, riderID int, req RideRequest) (*RideStatus, error) {
    var driver struct {
        ID       string
        Name     string
        Rating   float64
        Vehicle  string
        Distance float64 // in meters
    }

    // Find nearest available driver within radius
    err := tx.QueryRow(ctx,
        `SELECT 
            d.driver_id,
            d.name,
            d.rating,
            d.vehicle_model,
            ST_DistanceSphere(
                d.current_location, 
                ST_SetSRID(ST_MakePoint($1, $2), 4326)
            ) AS distance
        FROM drivers d
        WHERE d.available = true
        AND ST_DWithin(
            d.current_location,
            ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
            $3 * 1000)  -- Convert km to meters
        ORDER BY distance
        LIMIT 1
        FOR UPDATE SKIP LOCKED`,
        req.PickupLng, req.PickupLat, searchRadiusKm).Scan(
        &driver.ID, &driver.Name, &driver.Rating, &driver.Vehicle, &driver.Distance)

    if err != nil {
        return nil, errors.New("no available drivers nearby")
    }

    // Calculate price and ETA
    distanceKm := driver.Distance / 1000
    price := calculatePrice(distanceKm)
    eta := calculateETA(distanceKm)

    // Create ride record
    var rideID string
    err = tx.QueryRow(ctx,
        `INSERT INTO rides (
            driver_id, rider_id, status, 
            start_location, end_location,
            estimated_eta, price_estimate
        ) VALUES ($1, $2, 'requested',
            ST_SetSRID(ST_MakePoint($3, $4), 4326),
            ST_SetSRID(ST_MakePoint($5, $6), 4326),
            $7, $8)
        RETURNING id`,
        driver.ID, riderID,
        req.PickupLng, req.PickupLat,
        req.DropoffLng, req.DropoffLat,
        eta, price).Scan(&rideID)

    if err != nil {
        return nil, errors.New("failed to create ride record")
    }

    // Mark driver as unavailable
    _, err = tx.Exec(ctx,
        `UPDATE drivers SET available = false WHERE driver_id = $1`,
        driver.ID)
    if err != nil {
        return nil, errors.New("failed to update driver status")
    }

    // Create notification
    _, err = tx.Exec(ctx,
        `INSERT INTO driver_notifications (driver_id, ride_id, status)
         VALUES ($1, $2, 'pending')`,
        driver.ID, rideID)
    if err != nil {
        log.Printf("Failed to store notification: %v", err)
    }

    return &RideStatus{
        ID:        rideID,
        DriverID:  driver.ID,
        RiderID:   riderID,
        Status:    "requested",
        Price:     price,
        ETA:       eta,
        CreatedAt: time.Now(),
    }, nil
}

func calculatePrice(distanceKm float64) float64 {
    price := baseFare + (distanceKm * pricePerKm)
    
    // Apply surge pricing during rush hours
    hour := time.Now().Hour()
    if (hour >= 7 && hour <= 9) || (hour >= 17 && hour <= 19) {
        price *= 1.2 // 20% surge
    }

    // Round to 2 decimal places
    return math.Round(price*100) / 100
}

func calculateETA(distanceKm float64) int {
    // Base 5 minutes + 1 minute per 0.5km
    return 5 + int(distanceKm/0.5)
}

func listDriversHandler(w http.ResponseWriter, r *http.Request) {
    rows, err := dbPool.Query(r.Context(),
        `SELECT 
            driver_id, 
            ST_X(current_location::geometry) as lat, 
            ST_Y(current_location::geometry) as lng,
            name,
            rating,
            vehicle_model
        FROM drivers 
        WHERE available = true`)
    if err != nil {
        respondJSON(w, http.StatusInternalServerError, errorResponse("Database error"))
        return
    }
    defer rows.Close()

    var drivers []Driver
    for rows.Next() {
        var d Driver
        if err := rows.Scan(&d.ID, &d.Lat, &d.Lng, &d.Name, &d.Rating, &d.Vehicle); err != nil {
            respondJSON(w, http.StatusInternalServerError, errorResponse("Data parsing error"))
            return
        }
        drivers = append(drivers, d)
    }

    respondJSON(w, http.StatusOK, successResponse(drivers))
}

func validateRequest(r *http.Request) (*Claims, error) {
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        return nil, errors.New("authorization header required")
    }

    authParts := strings.Split(authHeader, " ")
    if len(authParts) != 2 || authParts[0] != "Bearer" {
        return nil, errors.New("invalid authorization header format")
    }

    return validateToken(authParts[1])
}

func validCoordinates(lat, lng float64) bool {
    return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

func errorResponse(msg string) RideResponse {
    return RideResponse{
        Success: false,
        Error:   msg,
    }
}

func successResponse(data interface{}) RideResponse {
    return RideResponse{
        Success: true,
        Data:    data,
    }
}

func cacheDriverLocation(driverID string, lat, lng float64) error {
    return redisClient.GeoAdd(context.Background(), "drivers", &redis.GeoLocation{
        Name:      driverID,
        Longitude: lng,
        Latitude:  lat,
    }).Err()
}
