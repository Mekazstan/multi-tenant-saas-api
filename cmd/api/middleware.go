package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/auth"
	"github.com/Mekazstan/multi-tenant-saas-api/internal/database"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type contextKey string

const (
	userIDKey   contextKey = "user_id"
	orgIDKey    contextKey = "org_id"
	apiKeyIDKey contextKey = "api_key_id"
	userRoleKey contextKey = "user_role"
)

func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondWithError(w, http.StatusUnauthorized, ApiError{
					Code:    "UNAUTHORIZED",
					Message: "Authentication required. Please provide a valid token.",
				})
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondWithError(w, http.StatusUnauthorized, ApiError{
					Code:    "INVALID_TOKEN_FORMAT",
					Message: "Authorization header must be in format: Bearer <token>",
				})
				return
			}

			tokenString := parts[1]

			userID, err := auth.ValidateJWT(tokenString, jwtSecret)
			if err != nil {
				respondWithError(w, http.StatusUnauthorized, ApiError{
					Code:    "INVALID_TOKEN",
					Message: "The provided token is invalid or has expired",
				})
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func APIKeyMiddleware(db *database.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				respondWithError(w, http.StatusUnauthorized, ApiError{
					Code:    "MISSING_API_KEY",
					Message: "API key is required. Please provide X-API-Key header.",
				})
				return
			}

			keyData, err := db.GetAPIKeyByKey(r.Context(), apiKey)
			if err != nil {
				respondWithError(w, http.StatusUnauthorized, ApiError{
					Code:    "INVALID_API_KEY",
					Message: "The provided API key is invalid or has been deactivated",
				})
				return
			}

			go func() {
				ctx := context.Background()
				db.UpdateAPIKeyLastUsed(ctx, keyData.ID)
			}()

			ctx := context.WithValue(r.Context(), apiKeyIDKey, keyData.ID)
			ctx = context.WithValue(ctx, orgIDKey, keyData.OrgID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RateLimitMiddleware(redisClient *redis.Client, limit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID, ok := r.Context().Value(orgIDKey).(uuid.UUID)
			if !ok {
				respondWithError(w, http.StatusInternalServerError, ApiError{
					Code:    "INTERNAL_ERROR",
					Message: "Failed to identify organization",
				})
				return
			}

			key := fmt.Sprintf("rate_limit:%s:%s", orgID.String(), time.Now().Format("2025-10-02-15-04"))

			ctx := r.Context()

			count, err := redisClient.Incr(ctx, key).Result()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				redisClient.Expire(ctx, key, time.Minute)
			}

			if count > int64(limit) {
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

				respondWithError(w, http.StatusTooManyRequests, ApiError{
					Code:    "RATE_LIMIT_EXCEEDED",
					Message: "Rate limit exceeded for your plan",
					Details: map[string]interface{}{
						"limit":       limit,
						"window":      "1 minute",
						"retry_after": 60,
					},
				})
				return
			}

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", limit-int(count)))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

			next.ServeHTTP(w, r)
		})
	}
}

func UsageTrackingMiddleware(db *database.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID, _ := r.Context().Value(orgIDKey).(uuid.UUID)
			apiKeyID, _ := r.Context().Value(apiKeyIDKey).(uuid.UUID)

			recorder := &statusRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(recorder, r)

			go func() {
				ctx := context.Background()
				_, err := db.CreateUsageRecord(ctx, database.CreateUsageRecordParams{
					OrganizationID: orgID,
					ApiKeyID:       apiKeyID,
					Endpoint:       r.URL.Path,
					Method:         r.Method,
					StatusCode:     int32(recorder.statusCode),
				})
				if err != nil {
					fmt.Printf("Failed to record usage: %v\n", err)
				}
			}()
		})
	}
}

func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		log.Printf("[%s] %s %s - Started", r.Method, r.URL.Path, r.RemoteAddr)

		next.ServeHTTP(recorder, r)

		duration := time.Since(start)
		log.Printf("[%s] %s %s - Completed %d in %v",
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			recorder.statusCode,
			duration,
		)
	})
}

func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Enforce HTTPS (only in production)
		// w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := generateSecureToken(16)
		w.Header().Set("X-Request-ID", requestID)

		ctx := context.WithValue(r.Context(), contextKey("request_id"), requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v", err)

				respondWithError(w, http.StatusInternalServerError, ApiError{
					Code:    "INTERNAL_ERROR",
					Message: "An unexpected error occurred",
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	return userID, ok
}

func GetOrgID(ctx context.Context) (uuid.UUID, bool) {
	orgID, ok := ctx.Value(orgIDKey).(uuid.UUID)
	return orgID, ok
}

func GetAPIKeyID(ctx context.Context) (uuid.UUID, bool) {
	apiKeyID, ok := ctx.Value(apiKeyIDKey).(uuid.UUID)
	return apiKeyID, ok
}
