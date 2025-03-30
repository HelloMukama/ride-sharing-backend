package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type GeoLocationAPIResponse struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"address"`
}

const locationCacheTTL = 5 * time.Minute

func FetchDriverLocation(driverID string) (*GeoLocationAPIResponse, error) {
	cachedLoc, err := getCachedLocation(driverID)
	if err == nil && cachedLoc != nil {
		return cachedLoc, nil
	}

	if isRateLimited("geo-api") {
		return nil, errors.New("API rate limit exceeded")
	}

	// Only use mock data (OpenStreetMap would be called via ReverseGeocode in caching.go)
	return mockDriverLocation(driverID)
}

func isRateLimited(resource string) bool {
	ctx := context.Background()
	key := fmt.Sprintf("rate_limit:%s", resource)
	
	count, err := redisClient.Incr(ctx, key).Result()
	if err != nil {
		return true
	}
	
	if count == 1 {
		redisClient.Expire(ctx, key, time.Hour)
	}
	
	return count > 100
}

func cacheLocation(driverID string, loc *GeoLocationAPIResponse) error {
	ctx := context.Background()
	data, err := json.Marshal(loc)
	if err != nil {
		return err
	}
	
	return redisClient.SetEX(ctx, fmt.Sprintf("driver_loc:%s", driverID), data, locationCacheTTL).Err()
}

func getCachedLocation(driverID string) (*GeoLocationAPIResponse, error) {
	ctx := context.Background()
	data, err := redisClient.Get(ctx, fmt.Sprintf("driver_loc:%s", driverID)).Bytes()
	if err != nil {
		return nil, err
	}
	
	var loc GeoLocationAPIResponse
	if err := json.Unmarshal(data, &loc); err != nil {
		return nil, err
	}
	return &loc, nil
}

func mockDriverLocation(driverID string) (*GeoLocationAPIResponse, error) {
	time.Sleep(100 * time.Millisecond) // Simulate API delay

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
