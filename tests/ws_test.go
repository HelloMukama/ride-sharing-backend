package main

import (
	"net/http/httptest"
	"testing"
	"golang.org/x/net/websocket"
)

func TestWebSocket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(WSHandler))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?driver_id=driver1"
	_, err := websocket.Dial(wsURL, "", "http://localhost")
	if err != nil {
		t.Errorf("WebSocket connection failed: %v", err)
	}
}
