// src/caching.go
package main

import (
	"context"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

var redisClient *redis.Client

func InitRedis() error {
	redisURL := os.Getenv("REDIS_URL")
    if redisURL == "" {
        // Fallback to Docker-compatible url if no .env variable is set
        redisURL = "redis:6379" 
    }

	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := redisClient.Ping(ctx).Result()
	return err
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

func FindNearbyDrivers(lat, lng float64, radius float64) ([]string, error) {
	ctx := context.Background()
	geoRadiusQuery := &redis.GeoRadiusQuery{
		Radius:    radius, // in kilometers
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
