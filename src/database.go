package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var dbPool *pgxpool.Pool

func InitDB() error {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://rideuser:ride123@db:5432/rides"
	}

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("failed to parse db config: %w", err)
	}

	config.MaxConns = 25
	config.MaxConnLifetime = 5 * time.Minute

	dbPool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	return nil
}

type RideStatus struct {
	ID         string    `json:"ride_id"`
	DriverID   string    `json:"driver_id"`
	RiderID    int       `json:"rider_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
