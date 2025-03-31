package main

import (
	"os"
)

// Export test variables
var (
	TestRedisClient  = redisClient
	TestDBPool      = dbPool
	TestInitRedis   = InitRedis
	TestInitDB      = InitDB
	TestInitAuth    = initAuth
	TestLoginHandler = loginHandler
)

// TestSetup initializes dependencies for testing
func TestSetup() {
	// Use test configurations
	os.Setenv("REDIS_URL", "localhost:6379")
	os.Setenv("DATABASE_URL", "postgres://rideuser:ride123@localhost:5432/rides_test")
	
	TestInitRedis()
	TestInitDB()
	TestInitAuth()
}
