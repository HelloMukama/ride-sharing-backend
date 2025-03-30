package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/fatih/color"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

var (
	limiter *limiter.Limiter
)

func initRateLimiter() {
	rate := limiter.Rate{
		Period: 1 * time.Hour,
		Limit:  1000,
	}
	store := memory.NewStore()
	limiter = limiter.New(store, rate)
}

func main() {
	// Initialize colored output
	success := color.New(color.FgGreen).SprintFunc()
	highlight := color.New(color.FgCyan).SprintFunc()

	// Clear terminal screen
	clearScreen()

	// 1. Load environment variables
	if err := loadEnvFiles(); err != nil {
		log.Fatal(color.RedString("Error loading environment: %v", err))
	}

	// 2. Initialize auth system
	if err := initAuth(); err != nil {
		log.Fatal(color.RedString("Auth initialization failed: %v", err))
	}

	// 3. Initialize Redis connection
	if err := InitRedis(); err != nil {
		log.Fatal(color.RedString("Failed to connect to Redis: %v", err))
	}
	log.Println(success("✓ Redis connection established"))

	// 4. Initialize rate limiter
	initRateLimiter()

	// 5. Create and configure router
	r := configureRouter()

	// 6. Start server
	port := getPort()
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf(success("Server starting on port %s..."), highlight(port))
	log.Println(success("┌──────────────────────────────────────────────────────────────┐"))
	log.Println(success("│                      API Endpoints                           │"))
	log.Println(success("│──────────────────────────────────────────────────────────────│"))
	log.Println(success("│ Method  │ Endpoint         │ Description                     │"))
	log.Println(success("├─────────┼──────────────────┼─────────────────────────────────┤"))
	log.Println(success("│ POST    │ /auth/login      │ User authentication (JWT)       │"))
	log.Println(success("│ POST    │ /request-ride    │ Request a ride, match driver    │"))
	log.Println(success("│ GET     │ /drivers         │ List available drivers          │"))
	log.Println(success("│ GET     │ /ride-status/:id │ Track an ongoing ride           │"))
	log.Println(success("└─────────┴──────────────────┴─────────────────────────────────┘"))
		
	log.Fatal(server.ListenAndServe())
}

func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func loadEnvFiles() error {
	envFiles := []string{
		".env",
		"../.env",
		"config/.env",
	}

	for _, file := range envFiles {
		if err := godotenv.Load(file); err == nil {
			log.Printf("Loaded environment from %s", file)
			return nil
		}
	}
	return fmt.Errorf("no valid .env file found (tried: %v)", envFiles)
}

func configureRouter() *mux.Router {
	r := mux.NewRouter()

	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// API Documentation Route
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"service": "Ride Sharing Backend",
			"endpoints": map[string]string{
				"auth_login":    "POST /auth/login",
				"request_ride":  "POST /request-ride (requires auth)",
				"list_drivers":  "GET /drivers",
				"ride_status":   "GET /ride-status/:id",
				"metrics":       "GET /metrics",
			},
		})
	})

	// Authentication Routes
	r.HandleFunc("/auth/login", loginHandler).Methods("POST")

	// Ride Management Routes (protected)
	api := r.PathPrefix("/").Subrouter()
	api.Use(AuthMiddleware)
	{
		api.HandleFunc("/request-ride", requestRideHandler).Methods("POST")
		api.HandleFunc("/drivers", listDriversHandler).Methods("GET")
		api.HandleFunc("/ride-status/{id}", rideStatusHandler).Methods("GET")
	}

	return r
}

func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8080"
}
