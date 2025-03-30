package main

import (
    "fmt"
    "log"
    "net/http"
    "time"
    
    "github.com/gorilla/websocket"
)

var (
    wsUpgrader = websocket.Upgrader{
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
    }
    driverConnections = make(map[string]*websocket.Conn)
)

// NotifyDriver sends real-time update to driver
func NotifyDriver(driverID, rideID string) error {
    conn, ok := driverConnections[driverID]
    if !ok {
        return fmt.Errorf("driver not connected")
    }

    msg := map[string]interface{}{
        "type":    "new_ride",
        "ride_id": rideID,
        "time":    time.Now().Unix(),
    }

    return conn.WriteJSON(msg)
}

// WSHandler handles WebSocket connections
func WSHandler(w http.ResponseWriter, r *http.Request) {
    driverID := r.URL.Query().Get("driver_id")
    if driverID == "" {
        http.Error(w, "driver_id required", http.StatusBadRequest)
        return
    }

    conn, err := wsUpgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("WebSocket upgrade failed: %v", err)
        return
    }
    defer conn.Close()

    driverConnections[driverID] = conn

    // Keep connection alive
    for {
        if _, _, err := conn.ReadMessage(); err != nil {
            delete(driverConnections, driverID)
            break
        }
    }
}
