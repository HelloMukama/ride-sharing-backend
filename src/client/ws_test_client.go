package main

import (
	"encoding/json"
	"log"
	"net/url"
	"os"
	"time"
	"github.com/gorilla/websocket"
)

func runWebSocketTest(driverID string) {
	u := url.URL{
		Scheme:   "ws",
		Host:     "app:8080",
		Path:     "/ws",
		RawQuery: "driver_id=" + driverID,
	}
	log.Printf("Connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	// Message handler
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("Invalid message: %s", message)
				continue
			}
			
			switch msg["type"] {
			case "new_ride", "pending_ride":
				log.Printf("NEW RIDE ASSIGNED: %+v", msg)
				// Auto-accept for testing
				c.WriteJSON(map[string]string{
					"action":  "accept",
					"ride_id": msg["ride_id"].(string),
				})
			default:
				log.Printf("received: %s", message)
			}
		}
	}()

	// Heartbeat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(`{"type":"heartbeat","time":"`+t.String()+`"}`))
			if err != nil {
				log.Println("write:", err)
				return
			}
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: /ws_test_client <driver_id>")
	}
	runWebSocketTest(os.Args[1])
}
