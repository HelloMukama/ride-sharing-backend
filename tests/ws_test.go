package main

import (
	"net/http/httptest"
	"testing"
	"golang.org/x/net/websocket"
	"strings"

	"ride-sharing-backend/src"
)

func TestWebSocket(t *testing.T) {

    InitRedis()
    InitDB()
    initAuth()
    
	server := httptest.NewServer(http.HandlerFunc(WSHandler))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?driver_id=driver1"
	_, err := websocket.Dial(wsURL, "", "http://localhost")
	if err != nil {
		t.Errorf("WebSocket connection failed: %v", err)
	}
}
