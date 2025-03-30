// src/api.go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

type GeoLocationAPIResponse struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"address"`
}

// Cache duration for API responses
const locationCacheTTL = 5 * time.Minute

func FetchDriverLocation(driverID string) (*GeoLocationAPIResponse, error) {
	// First check Redis cache
	cachedLoc, err := getCachedLocation(driverID)
	if err == nil && cachedLoc != nil {
		return cachedLoc, nil
	}

	// Check rate limits before calling external API
	if isRateLimited("geo-api") {
		return nil, errors.New("API rate limit exceeded")
	}

	// Use real API if configured, otherwise fallback to mock
	apiKey := os.Getenv("API_KEY")
	if apiKey != "" && apiKey != "your_api_key_here" {
		loc, err := fetchFromGoogleMaps(driverID, apiKey)
		if err == nil {
			// Cache successful responses
			cacheLocation(driverID, loc)
			return loc, nil
		}
		log.Printf("Google Maps API failed, falling back to mock: %v", err)
	}

	// Fallback to mock implementation
	return mockDriverLocation(driverID)
}

func ReverseGeocode(lat, lng float64) (string, error) {
	apiKey := os.Getenv("API_KEY")
	if apiKey != "" && apiKey != "your_api_key_here" {
		url := fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?latlng=%f,%f&key=%s", 
			lat, lng, apiKey)
		
		resp, err := http.Get(url)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		var result struct {
			Results []struct {
				FormattedAddress string `json:"formatted_address"`
			} `json:"results"`
		}
		
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", err
		}

		if len(result.Results) > 0 {
			return result.Results[0].FormattedAddress, nil
		}
		return "", errors.New("no address found")
	}

	// Mock implementation fallback
	return fmt.Sprintf("Near %.4f, %.4f", lat, lng), nil
}

// --- Helper Functions ---

func fetchFromGoogleMaps(driverID, apiKey string) (*GeoLocationAPIResponse, error) {
	url := fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?address=%s&key=%s", 
		driverID, apiKey)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			Geometry struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
			FormattedAddress string `json:"formatted_address"`
		} `json:"results"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Results) == 0 {
		return nil, errors.New("no results found")
	}

	return &GeoLocationAPIResponse{
		Lat:     result.Results[0].Geometry.Location.Lat,
		Lng:     result.Results[0].Geometry.Location.Lng,
		Address: result.Results[0].FormattedAddress,
	}, nil
}

func isRateLimited(resource string) bool {
	ctx := context.Background()
	key := fmt.Sprintf("rate_limit:%s", resource)
	
	count, err := redisClient.Incr(ctx, key).Result()
	if err != nil {
		return true // Fail closed on error
	}
	
	if count == 1 {
		redisClient.Expire(ctx, key, time.Hour)
	}
	
	return count > 100 // Limit to 100 requests/hour
}

func cacheLocation(driverID string, loc *GeoLocationAPIResponse) error {
	ctx := context.Background()
	data, err := json.Marshal(loc)
	if err != nil {
		return err
	}
	
	return redisClient.SetEX(ctx, fmt.Sprintf("ride:%s", rideID), data, locationCacheTTL).Err()
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
