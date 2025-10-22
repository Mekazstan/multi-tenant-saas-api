package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/auth"
	"github.com/google/uuid"
)

func TestAuthMiddleware(t *testing.T) {
	secret := "test-secret"
	userID := uuid.New()
	validToken, _ := auth.MakeJWT(userID, secret, time.Hour)

	middleware := AuthMiddleware(secret)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "Valid token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing auth header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid format",
			authHeader:     "InvalidFormat",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	rr := httptest.NewRecorder()

	handler := middlewareCors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS header not set correctly")
	}

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}