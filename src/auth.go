package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
)

// Custom claims with user ID, role and version
type Claims struct {
    UserID   int    `json:"user_id"`
    Username string `json:"username"`
    Email    string `json:"email"`
    Role     string `json:"role"`
    Version  int    `json:"version"`
    jwt.RegisteredClaims
}

const (
	tokenVersionPrefix = "token_version:"
	defaultJWTExpiry   = 12 * time.Hour
)

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
        jwtExpiration = 12 * time.Hour
    } else {
        var err error
        jwtExpiration, err = time.ParseDuration(expireStr)
        if err != nil {
            return errors.New("invalid JWT_EXPIRE format. Examples: 24h, 1h30m")
        }
    }

    // Verify Redis connection if needed
    if redisClient == nil {
        return errors.New("Redis client not initialized - token revocation will not work")
    } else {
        // Test Redis connection
        ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
        defer cancel()
        if _, err := redisClient.Ping(ctx).Result(); err != nil {
            return fmt.Errorf("Redis connection test failed: %w", err)
        }
        log.Println("Redis connection verified for auth system")
    }
    
    log.Printf("JWT initialized (secret length: %d, expiration: %v)", len(jwtSecret), jwtExpiration)
    return nil
}

func SetupAuthRoutes(r *mux.Router) {
	r.HandleFunc("/auth/login", loginHandler).Methods("POST")
	r.HandleFunc("/auth/validate", validateTokenHandler).Methods("GET")
	r.HandleFunc("/auth/logout", logoutHandler).Methods("POST")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// [Previous rate limiting code remains the same]
	
	var creds struct {
		Username string `json:"username"`
		UserID   int    `json:"user_id"`
		Role     string `json:"role"`
        Email    string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	tokenString, err := generateJWT(creds.Username, creds.UserID, creds.Role)
	if err != nil {
		log.Printf("Token generation error: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Token generation failed"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"token":      tokenString,
		"expires_in": jwtExpiration.String(),
	})
}

func generateJWT(username string, userID int, role string) (string, error) {
    version := 1 // Default version
    
    if redisClient != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
        defer cancel()
        
        // Actually use the ctx variable
        if ver, err := redisClient.Get(ctx, tokenVersionPrefix+strconv.Itoa(userID)).Int(); err == nil {
            version = ver
        }
    }

    claims := &Claims{
        UserID:   userID,
        Username: username,
        Role:     role,
        Version:  version,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtExpiration)),
            Issuer:    "ride-sharing-backend",
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(jwtSecret)
}

func validateToken(tokenString string) (*Claims, error) {
    if len(jwtSecret) == 0 {
        return nil, errors.New("JWT secret not initialized")
    }

    claims := &Claims{}
    token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return jwtSecret, nil
    })

    if err != nil || !token.Valid {
        return nil, errors.New("invalid token")
    }

    // Debug: Log Redis status during validation
    log.Printf("Validating token for user %d (version %d)", claims.UserID, claims.Version)

    currentVer, err := getTokenVersion(claims.UserID)
    if err != nil {
        log.Printf("Token version check failed: %v", err)
        // If Redis is down, we should fail closed (reject tokens)
        return nil, fmt.Errorf("token verification unavailable")
    }

    log.Printf("Current token version for user %d: %d (token version: %d)", 
        claims.UserID, currentVer, claims.Version)

    if claims.Version < currentVer {
        return nil, errors.New("token revoked - please login again")
    }

    return claims, nil
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
		"version":  claims.Version,
		"expires":  claims.ExpiresAt.Format(time.RFC3339),
	})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
    claims, ok := r.Context().Value("userClaims").(*Claims)
    if !ok {
        respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
        return
    }

    // Debug: Log current Redis status
    log.Printf("Redis client status: %v", redisClient != nil)

    // Get current version first
    currentVer, err := getTokenVersion(claims.UserID)
    if err != nil {
        log.Printf("Failed to get current token version: %v", err)
        respondJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Failed to get current token version",
            "details": err.Error(),
        })
        return
    }

    // Increment version
    newVer, err := incrementTokenVersion(claims.UserID)
    if err != nil {
        log.Printf("Failed to increment token version: %v", err)
        respondJSON(w, http.StatusInternalServerError, map[string]string{
            "error": "Logout failed - could not revoke token",
            "details": err.Error(),
        })
        return
    }

    log.Printf("User %d logged out (token version %d -> %d)", claims.UserID, currentVer, newVer)
    respondJSON(w, http.StatusOK, map[string]string{
        "status": "logged out",
        "version": strconv.Itoa(newVer),
        "message": fmt.Sprintf("Token version incremented from %d to %d", currentVer, newVer),
    })
}

func getTokenVersion(userID int) (int, error) {
    if redisClient == nil {
        return 0, errors.New("redis client not available")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    ver, err := redisClient.Get(ctx, tokenVersionPrefix+strconv.Itoa(userID)).Int()
    if err == redis.Nil {
        // If no version exists, return 1 (first valid version)
        return 1, nil
    }
    return ver, err
}

func incrementTokenVersion(userID int) (int, error) {
    if redisClient == nil {
        return 0, errors.New("redis client not available")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    key := tokenVersionPrefix + strconv.Itoa(userID)
    newVer, err := redisClient.Incr(ctx, key).Result()
    if err != nil {
        return 0, err
    }
    
    // Set expiration (30 days)
    if err := redisClient.Expire(ctx, key, 30*24*time.Hour).Err(); err != nil {
        return int(newVer), fmt.Errorf("failed to set expiration: %w", err)
    }
    
    return int(newVer), nil
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
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
