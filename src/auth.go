package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/ulule/limiter/v3"
)

// Custom claims with user ID and role
type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"` // e.g., "rider", "driver"
	jwt.RegisteredClaims
}

// Initialize JWT configuration
var (
	jwtSecret     []byte
	jwtExpiration time.Duration
)

func initAuth() error {
	// Load JWT secret
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return errors.New("JWT_SECRET not configured")
	}
	jwtSecret = []byte(secret)

	// Parse expiration duration
	expireStr := os.Getenv("JWT_EXPIRE")
	if expireStr == "" {
		expireStr = "24h" // Default fallback
	}
	
	var err error
	jwtExpiration, err = time.ParseDuration(expireStr)
	if err != nil {
		return errors.New("invalid JWT_EXPIRE format. Examples: 24h, 1h30m")
	}

	return nil
}

func SetupAuthRoutes(r *mux.Router) {
	r.HandleFunc("/auth/login", loginHandler).Methods("POST")
	r.HandleFunc("/auth/validate", validateTokenHandler).Methods("GET")
}

// Enhanced login handler with proper request parsing
func loginHandler(w http.ResponseWriter, r *http.Request) {
	// Rate limiting check
	ctx, err := limiter.Get(r.Context(), r.RemoteAddr)
	if err != nil || ctx.Reached {
		respondJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many login attempts"})
		return
	}

	var creds struct {
		Username string `json:"username"`
		UserID   int    `json:"user_id"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, `{"error":"Invalid request"}`, http.StatusBadRequest)
		return
	}

	tokenString, err := generateJWT(creds.Username, creds.UserID, creds.Role)
	if err != nil {
		http.Error(w, `{"error":"Token generation failed"}`, http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"token":      tokenString,
		"expires_in": jwtExpiration.String(),
	})
}

// Generate token with custom claims
func generateJWT(username string, userID int, role string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtExpiration)),
			Issuer:    "ride-sharing-backend",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Middleware for protected routes
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Rate limiting check
		ctx, err := limiter.Get(r.Context(), r.RemoteAddr)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "rate limit error"})
			return
		}
		
		if ctx.Reached {
			respondJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			return
		}

		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Authorization header required"})
			return
		}

		claims, err := validateToken(tokenString)
		if err != nil {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}

		// Add claims to request context
		ctxWithClaims := context.WithValue(r.Context(), "userClaims", claims)
		next.ServeHTTP(w, r.WithContext(ctxWithClaims))
	})
}

func validateTokenHandler(w http.ResponseWriter, r *http.Request) {
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Missing authorization header"})
		return
	}

	claims, err := validateToken(tokenString)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":  claims.UserID,
		"username": claims.Username,
		"role":     claims.Role,
		"expires":  claims.ExpiresAt.Format(time.RFC3339),
	})
}

func validateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// Helper for JSON responses
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
