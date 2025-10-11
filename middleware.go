package main

import (
	"context"
	"fmt"
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

// AuthMiddleware validates JWT tokens and attaches user info to context
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondWithError(w, http.StatusUnauthorized, ApiError{
					Code:    "UNAUTHORIZED",
					Message: "Authentication required. Please provide a valid token.",
				})
				return
			}

			// Check for Bearer token format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondWithError(w, http.StatusUnauthorized, ApiError{
					Code:    "INVALID_TOKEN_FORMAT",
					Message: "Authorization header must be in format: Bearer <token>",
				})
				return
			}

			tokenString := parts[1]

			// Validate JWT
			userID, err := auth.ValidateJWT(tokenString, jwtSecret)
			if err != nil {
				respondWithError(w, http.StatusUnauthorized, ApiError{
					Code:    "INVALID_TOKEN",
					Message: "The provided token is invalid or has expired",
				})
				return
			}

			// Attach user_id to request context
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// APIKeyMiddleware validates API keys and attaches org/key info to context
func APIKeyMiddleware(db *database.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract API key from X-API-Key header
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				respondWithError(w, http.StatusUnauthorized, ApiError{
					Code:    "MISSING_API_KEY",
					Message: "API key is required. Please provide X-API-Key header.",
				})
				return
			}

			// Validate and get API key details
			keyData, err := db.GetAPIKeyByKey(r.Context(), apiKey)
			if err != nil {
				respondWithError(w, http.StatusUnauthorized, ApiError{
					Code:    "INVALID_API_KEY",
					Message: "The provided API key is invalid or has been deactivated",
				})
				return
			}

			// Update last_used_at timestamp
			go func() {
				ctx := context.Background()
				db.UpdateAPIKeyLastUsed(ctx, keyData.ID)
			}()

			// Attach info to context
			ctx := context.WithValue(r.Context(), apiKeyIDKey, keyData.ID)
			ctx = context.WithValue(ctx, orgIDKey, keyData.OrgID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RateLimitMiddleware limits requests per organization using Redis
func RateLimitMiddleware(redisClient *redis.Client, limit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get org_id from context (set by APIKeyMiddleware)
			orgID, ok := r.Context().Value(orgIDKey).(uuid.UUID)
			if !ok {
				respondWithError(w, http.StatusInternalServerError, ApiError{
					Code:    "INTERNAL_ERROR",
					Message: "Failed to identify organization",
				})
				return
			}

			// Create rate limit key
			key := fmt.Sprintf("rate_limit:%s:%s", orgID.String(), time.Now().Format("2006-01-02-15-04"))

			ctx := r.Context()

			// Increment counter
			count, err := redisClient.Incr(ctx, key).Result()
			if err != nil {
				// Log error but don't block request
				next.ServeHTTP(w, r)
				return
			}

			// Set expiration on first request
			if count == 1 {
				redisClient.Expire(ctx, key, time.Minute)
			}

			// Check if limit exceeded
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

			// Add rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", limit-int(count)))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

			next.ServeHTTP(w, r)
		})
	}
}

// UsageTrackingMiddleware records API usage after request completes
func UsageTrackingMiddleware(db *database.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get context values
			orgID, _ := r.Context().Value(orgIDKey).(uuid.UUID)
			apiKeyID, _ := r.Context().Value(apiKeyIDKey).(uuid.UUID)

			// Create a response recorder to capture status code
			recorder := &statusRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Call next handler
			next.ServeHTTP(recorder, r)

			// Record usage asynchronously
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
					// Log error but don't fail request
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

// statusRecorder wraps ResponseWriter to capture status code
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// Helper function to get user ID from context
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	return userID, ok
}

// Helper function to get org ID from context
func GetOrgID(ctx context.Context) (uuid.UUID, bool) {
	orgID, ok := ctx.Value(orgIDKey).(uuid.UUID)
	return orgID, ok
}

// Helper function to get API key ID from context
func GetAPIKeyID(ctx context.Context) (uuid.UUID, bool) {
	apiKeyID, ok := ctx.Value(apiKeyIDKey).(uuid.UUID)
	return apiKeyID, ok
}
