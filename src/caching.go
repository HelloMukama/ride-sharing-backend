package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"log"

	"github.com/go-redis/redis/v8"
)

var redisClient *redis.Client

func InitRedis() error {
    redisURL := os.Getenv("REDIS_URL")
    if redisURL == "" {
        redisURL = "redis:6379"
    }

    redisClient = redis.NewClient(&redis.Options{
        Addr:     redisURL,
        Password: "",
        DB:       0,
    })

    // Retry connection up to 5 times
    var err error
    for i := 0; i < 5; i++ {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        _, err = redisClient.Ping(ctx).Result()
        cancel()
        
        if err == nil {
            break
        }
        
        log.Printf("Redis connection attempt %d failed: %v", i+1, err)
        time.Sleep(2 * time.Second)
    }
    
    if err != nil {
        return fmt.Errorf("failed to connect to Redis after retries: %w", err)
    }
    
    log.Println("Redis connection established")
    return nil
}

func CacheDriverLocation(driverID string, lat, lng float64) error {
	ctx := context.Background()
	geoLocation := &redis.GeoLocation{
		Name:      driverID,
		Longitude: lng,
		Latitude:  lat,
	}
	return redisClient.GeoAdd(ctx, "drivers", geoLocation).Err()
}

func FindNearbyDrivers(lat, lng, radius float64) ([]string, error) {
	ctx := context.Background()
	geoRadiusQuery := &redis.GeoRadiusQuery{
		Radius:    radius,
		Unit:      "km",
		WithDist:  true,
		Sort:      "ASC",
		Count:     10,
	}

	locations, err := redisClient.GeoRadius(ctx, "drivers", lng, lat, geoRadiusQuery).Result()
	if err != nil {
		return nil, err
	}

	var driverIDs []string
	for _, loc := range locations {
		driverIDs = append(driverIDs, loc.Name)
	}
	return driverIDs, nil
}

// OpenStreetMap implementation
func ReverseGeocode(lat, lng float64) (string, error) {
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?format=json&lat=%f&lon=%f", lat, lng)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.DisplayName == "" {
		return fmt.Sprintf("Near %.4f, %.4f", lat, lng), nil
	}
	return result.DisplayName, nil
}
