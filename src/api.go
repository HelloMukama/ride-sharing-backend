// src/api.go
package main

import (
	"fmt"
	"time"
)

type GeoLocationAPIResponse struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"address"`
}

func FetchDriverLocation(driverID string) (*GeoLocationAPIResponse, error) {
	// In a real app, this would call an external API
	// For now, we'll mock the response based on driverID
	time.Sleep(100 * time.Millisecond) // Simulate API delay

	// Mock locations for different drivers
	locations := map[string]GeoLocationAPIResponse{
		"driver1": {Lat: 0.3135, Lng: 32.5811, Address: "Kampala Road"},
		"driver2": {Lat: 0.3167, Lng: 32.5825, Address: "Nakasero"},
		"driver3": {Lat: 0.3150, Lng: 32.5800, Address: "Kololo"},
	}

	if loc, ok := locations[driverID]; ok {
		return &loc, nil
	}

	return nil, fmt.Errorf("driver not found")
}

func ReverseGeocode(lat, lng float64) (string, error) {
	// Mock implementation
	return fmt.Sprintf("Near %.4f, %.4f", lat, lng), nil
}
