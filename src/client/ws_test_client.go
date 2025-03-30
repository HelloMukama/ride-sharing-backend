package main

import (
	"log"
	"net/url"
	"os"
	"time"
	"github.com/gorilla/websocket"
)

func runWebSocketTest(driverID string) {
	u := url.URL{
		Scheme:   "ws",
		Host:     "app:8080",  // Changed from localhost to app (Docker service name)
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
		case <-time.After(5 * time.Second):
			log.Println("Connection timeout")
			return
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: /ws_test_client <driver_id>")
	}
	runWebSocketTest(os.Args[1])
}
