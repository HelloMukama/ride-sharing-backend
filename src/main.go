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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ululeLimiter "github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

var (
	appLimiter *ululeLimiter.Limiter
	// Prometheus metrics
	requestsCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ride_sharing_requests_total",
			Help: "Total API requests",
		},
		[]string{"path", "method", "status"},
	)
	responseTimeHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ride_sharing_response_time_seconds",
			Help:    "Response time distribution",
			Buckets: []float64{0.1, 0.5, 1, 2, 5},
		},
		[]string{"path", "method"},
	)
)

func initRateLimiter() {
	rate := ululeLimiter.Rate{
		Period: 1 * time.Hour,
		Limit:  1000,
	}
	store := memory.NewStore()
	appLimiter = ululeLimiter.New(store, rate)
}

func main() {
    // Initialize colored output
    success := color.New(color.FgGreen).SprintFunc()
    highlight := color.New(color.FgCyan).SprintFunc()

    // Clear terminal screen
    clearScreen()

    // 1. Load environment variables FIRST
    if err := loadEnvFiles(); err != nil {
        log.Fatal(color.RedString("Error loading environment: %v", err))
    }

    // 2. Initialize auth system IMMEDIATELY with forced validation
    if len(os.Getenv("JWT_SECRET")) < 32 {
        log.Fatal("JWT_SECRET must be at least 32 characters")
    }
    if err := initAuth(); err != nil {
        log.Fatal(color.RedString("Auth initialization failed: %v", err))
    }

    // 3. Initialize Redis
    if err := InitRedis(); err != nil {
        log.Fatal(color.RedString("Redis initialization failed: %v", err))
    }

    // 4. Initialize database
    if err := InitDB(); err != nil {
        log.Fatal(color.RedString("Database initialization failed: %v", err))
    }

    // 5. Initialize rate limiter
    initRateLimiter()

    // 6. Create and configure router
    r := configureRouter()

    // 7. Start server
    port := getPort()
    server := &http.Server{
        Addr:         ":" + port,
        Handler:      r,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 30 * time.Second,
    }

    log.Printf(success("Server starting on port %s..."), highlight(port))
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
		os.Getenv("ENV_FILE"),
		"/app/.env",
		".env",
	}
	
	for _, file := range envFiles {
		if file == "" {
			continue
		}
		err := godotenv.Load(file)
		if err == nil {
			log.Printf("Successfully loaded environment from: %s", file)
			return nil
		}
		log.Printf("Attempt failed for %s: %v", file, err)
	}
	return fmt.Errorf("failed to load .env from any location (tried: %v)", envFiles)
}

func configureRouter() *mux.Router {
	r := mux.NewRouter()
	SetupAuthRoutes(r)

	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// WebSocket endpoint
	r.HandleFunc("/ws", WSHandler)

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

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
				"websocket":     "GET /ws?driver_id=DRIVER_ID",
			},
		})
	})

	// Authentication Routes
	r.HandleFunc("/auth/login", loginHandler).Methods("POST")

	// Ride Management Routes (protected)
	api := r.PathPrefix("/").Subrouter()
	api.Use(AuthMiddleware)
	api.Use(metricsMiddleware)
	{
		api.HandleFunc("/request-ride", requestRideHandler).Methods("POST")
		api.HandleFunc("/drivers", listDriversHandler).Methods("GET")
		api.HandleFunc("/ride-status/{id}", rideStatusHandler).Methods("GET")
	}

	return r
}

// metricsMiddleware tracks request metrics
func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path
		method := r.Method

		// Wrap response writer to capture status code
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := fmt.Sprintf("%d", rw.status)

		requestsCounter.WithLabelValues(path, method, status).Inc()
		responseTimeHistogram.WithLabelValues(path, method).Observe(duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8080"
}
