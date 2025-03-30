package main

import (
	"log"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func runWebSocketTest(driverID string) {
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws", RawQuery: "driver_id=" + driverID}
	log.Printf("Connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("received: %s", message)
		}
	}()

	// Keep connection alive
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run ws_test_client.go <driver_id>")
	}
	runWebSocketTest(os.Args[1])
}
