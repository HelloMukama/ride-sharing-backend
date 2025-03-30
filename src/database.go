package main

import (
	"context"
	"fmt"
	"log"
	"os"
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbPool, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := dbPool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Run migrations
	if err := verifyAndMigrateDB(); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}

	log.Println("Database connection and migrations verified")
	return nil
}

func verifyAndMigrateDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var tableExists bool
	err := dbPool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'rides'
		)`).Scan(&tableExists)

	if err != nil {
		return fmt.Errorf("failed to check for rides table: %w", err)
	}

	if !tableExists {
		log.Println("Initializing database schema...")
		if err := runMigrations(ctx); err != nil {
			return err
		}
	}
	return nil
}

func runMigrations(ctx context.Context) error {
	tx, err := dbPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create rides table
	if _, err := tx.Exec(ctx, `
		CREATE TABLE rides (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			driver_id VARCHAR(255) NOT NULL,
			rider_id INTEGER NOT NULL,
			status VARCHAR(50) NOT NULL CHECK (status IN ('requested', 'accepted', 'in_progress', 'completed', 'cancelled')),
			start_location GEOGRAPHY(POINT) NOT NULL,
			end_location GEOGRAPHY(POINT),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("failed to create rides table: %w", err)
	}

	// Create indexes
	if _, err := tx.Exec(ctx, `CREATE INDEX idx_rides_status ON rides(status)`); err != nil {
		return fmt.Errorf("failed to create status index: %w", err)
	}

	if _, err := tx.Exec(ctx, `CREATE INDEX idx_rides_rider_id ON rides(rider_id)`); err != nil {
		return fmt.Errorf("failed to create rider_id index: %w", err)
	}

	if _, err := tx.Exec(ctx, `CREATE INDEX idx_rides_driver_id ON rides(driver_id)`); err != nil {
		return fmt.Errorf("failed to create driver_id index: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
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
