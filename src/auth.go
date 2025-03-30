package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
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

	log.Printf("JWT initialized with expiration: %v", jwtExpiration)
	return nil
}

func SetupAuthRoutes(r *mux.Router) {
	r.HandleFunc("/auth/login", loginHandler).Methods("POST")
	r.HandleFunc("/auth/validate", validateTokenHandler).Methods("GET")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// Rate limiting check
	ctx, err := appLimiter.Get(r.Context(), r.RemoteAddr)
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
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	tokenString, err := generateJWT(creds.Username, creds.UserID, creds.Role)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Token generation failed"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"token":      tokenString,
		"expires_in": jwtExpiration.String(),
	})
}

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

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Rate limiting check
		ctx, err := appLimiter.Get(r.Context(), r.RemoteAddr)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "rate limit error"})
			return
		}
		
		if ctx.Reached {
			respondJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Authorization header required"})
			return
		}

		authParts := strings.Split(authHeader, " ")
		if len(authParts) != 2 || authParts[0] != "Bearer" {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid Authorization header format"})
			return
		}

		tokenString := authParts[1]
		claims, err := validateToken(tokenString)
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}

		ctxWithClaims := context.WithValue(r.Context(), "userClaims", claims)
		next.ServeHTTP(w, r.WithContext(ctxWithClaims))
	})
}

func validateTokenHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Missing authorization header"})
		return
	}

	authParts := strings.Split(authHeader, " ")
	if len(authParts) != 2 || authParts[0] != "Bearer" {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid Authorization header format"})
		return
	}

	claims, err := validateToken(authParts[1])
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
	log.Println("\n=== Token Validation ===")
	log.Printf("Token: %q", tokenString)
	log.Printf("Current time: %v", time.Now())

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		var jwtErr *jwt.ValidationError
		if errors.As(err, &jwtErr) {
			switch {
			case jwtErr.Errors&jwt.ValidationErrorMalformed != 0:
				log.Println("Malformed token")
			case jwtErr.Errors&jwt.ValidationErrorExpired != 0:
				log.Println("Expired token")
			case jwtErr.Errors&jwt.ValidationErrorNotValidYet != 0:
				log.Println("Token not valid yet")
			case jwtErr.Errors&jwt.ValidationErrorSignatureInvalid != 0:
				log.Println("Signature invalid")
			default:
				log.Printf("Validation error: %v", jwtErr)
			}
		}
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.Issuer != "ride-sharing-backend" {
		return nil, errors.New("invalid issuer")
	}

	log.Println("Token validated successfully")
	return claims, nil
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
