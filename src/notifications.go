package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "sync"
    "github.com/gorilla/websocket"
)

var (
    wsUpgrader = websocket.Upgrader{
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
    }
    
    driverConnections = struct {
        sync.RWMutex
        m map[string]*websocket.Conn
    }{m: make(map[string]*websocket.Conn)}
)

func NotifyDriver(driverID string, message interface{}) error {
    driverConnections.RLock()
    conn, ok := driverConnections.m[driverID]
    driverConnections.RUnlock()
    
    if !ok {
        return fmt.Errorf("driver not connected")
    }
    
    return conn.WriteJSON(message)
}

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

    // Register connection
    driverConnections.Lock()
    driverConnections.m[driverID] = conn
    driverConnections.Unlock()
    
    defer func() {
        driverConnections.Lock()
        delete(driverConnections.m, driverID)
        driverConnections.Unlock()
    }()

    // Check for pending notifications
    rows, err := dbPool.Query(r.Context(),
        `SELECT ride_id FROM driver_notifications 
         WHERE driver_id = $1 AND status = 'pending'`,
        driverID)
    if err == nil {
        defer rows.Close()
        for rows.Next() {
            var rideID string
            if err := rows.Scan(&rideID); err == nil {
                conn.WriteJSON(map[string]interface{}{
                    "type": "pending_ride",
                    "ride_id": rideID,
                })
            }
        }
    }

    // Heartbeat and message handling
    for {
        _, _, err := conn.ReadMessage()
        if err != nil {
            log.Printf("Driver %s disconnected: %v", driverID, err)
            break
        }
        // Keep connection alive
    }
}

func UpdateNotificationStatus(driverID, rideID, status string) error {
    _, err := dbPool.Exec(context.Background(),
        `UPDATE driver_notifications 
         SET status = $1, updated_at = NOW()
         WHERE driver_id = $2 AND ride_id = $3`,
        status, driverID, rideID)
    return err
}
