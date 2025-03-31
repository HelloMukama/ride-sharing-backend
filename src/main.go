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
    warning := color.New(color.FgYellow).SprintFunc()

    // Clear terminal screen
    clearScreen()

    log.Println(warning("Starting initialization sequence..."))

    // 1. Load environment variables FIRST
    if err := loadEnvFiles(); err != nil {
        log.Fatal(color.RedString("Error loading environment: %v", err))
    }
    log.Println(success("Environment variables loaded"))

    // 2. Initialize Redis with retries
    log.Println("Initializing Redis connection...")
    if err := InitRedis(); err != nil {
        log.Fatal(color.RedString("Redis initialization failed: %v", err))
    }
    log.Println(success("Redis connection established and verified"))

    // 3. Initialize database with retries
    log.Println("Initializing database connection...")
    if err := InitDB(); err != nil {
        log.Fatal(color.RedString("Database initialization failed: %v", err))
    }
    log.Println(success("Database connection and migrations verified"))

    // 4. Initialize auth system (requires Redis)
    log.Println("Initializing authentication system...")
    if err := initAuth(); err != nil {
        log.Fatal(color.RedString("Auth initialization failed: %v", err))
    }
    log.Println(success("Authentication system ready"))

    // 5. Initialize rate limiter
    initRateLimiter()
    log.Println(success("Rate limiter initialized"))

    // 6. Create and configure router
    r := configureRouter()
    log.Println(success("Router configured"))

    // 7. Start server
    port := getPort()
    server := &http.Server{
        Addr:         ":" + port,
        Handler:      r,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    log.Printf(success("\nServer starting on port %s...\n"), highlight(port))
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatal(color.RedString("Server failed: %v", err))
    }
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
    
    // Public routes
    r.HandleFunc("/auth/login", loginHandler).Methods("POST")
    r.HandleFunc("/auth/validate", validateTokenHandler).Methods("GET")
    r.Handle("/metrics", promhttp.Handler())
    r.HandleFunc("/ws", WSHandler)
    r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("OK"))
    })

    // Protected routes
    api := r.PathPrefix("/").Subrouter()
    api.Use(AuthMiddleware)
    api.Use(metricsMiddleware)
    {
        api.HandleFunc("/request-ride", requestRideHandler).Methods("POST")
        api.HandleFunc("/drivers", listDriversHandler).Methods("GET")
        api.HandleFunc("/ride-status/{id}", rideStatusHandler).Methods("GET")

        r.HandleFunc("/payment/initiate", initiatePaymentHandler).Methods("POST")
		r.HandleFunc("/payment/verify", verifyPaymentHandler).Methods("POST")

        api.HandleFunc("/auth/logout", logoutHandler).Methods("POST")
    }

    // API Documentation Route
    r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "service": "Ride Sharing Backend",
            "endpoints": map[string]string{
                "auth_login":    "POST /auth/login",
                "auth_validate": "GET /auth/validate",
                "auth_logout":   "POST /auth/logout (protected)",
                "request_ride":  "POST /request-ride (protected)",
                "list_drivers":  "GET /drivers (protected)",
                "ride_status":   "GET /ride-status/:id (protected)",
                "metrics":       "GET /metrics",
                "websocket":     "GET /ws?driver_id=DRIVER_ID",
            },
        })
    })

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

func initiatePaymentHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := validateRequest(r)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, errorResponse(err.Error()))
		return
	}

	var req struct {
		RideID string  `json:"ride_id"`
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse("Invalid request"))
		return
	}

	paymentLink, err := ProcessPayment(req.RideID, req.Amount, claims.Email)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, errorResponse(err.Error()))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"payment_link": paymentLink,
	})
}

func verifyPaymentHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TxRef string `json:"tx_ref"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse("Invalid request"))
		return
	}

	verified, err := VerifyPayment(req.TxRef)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, errorResponse(err.Error()))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"verified": verified,
	})
}
