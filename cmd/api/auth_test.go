package main

import (
	"testing"
	"time"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/auth"
	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "Valid password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "Empty password",
			password: "",
			wantErr:  false,
		},
		{
			name:     "Long password",
			password: string(make([]byte, 100)),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := auth.HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && hash == "" {
				t.Error("HashPassword() returned empty hash")
			}
		})
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "test123"
	hash, _ := auth.HashPassword(password)

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{
			name:     "Correct password",
			password: password,
			hash:     hash,
			want:     true,
		},
		{
			name:     "Wrong password",
			password: "wrongpassword",
			hash:     hash,
			want:     false,
		},
		{
			name:     "Empty password",
			password: "",
			hash:     hash,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, _ := auth.CheckPasswordHash(tt.password, tt.hash)
			if match != tt.want {
				t.Errorf("CheckPasswordHash() = %v, want %v", match, tt.want)
			}
		})
	}
}

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"
	duration := time.Hour

	token, err := auth.MakeJWT(userID, secret, duration)
	if err != nil {
		t.Fatalf("MakeJWT() error = %v", err)
	}

	if token == "" {
		t.Error("MakeJWT() returned empty token")
	}

	parsedUserID, err := auth.ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT() error = %v", err)
	}

	if parsedUserID != userID {
		t.Errorf("ValidateJWT() = %v, want %v", parsedUserID, userID)
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	tests := []struct {
		name    string
		token   string
		secret  string
		wantErr bool
	}{
		{
			name:    "Valid token",
			token:   mustMakeJWT(userID, secret, time.Hour),
			secret:  secret,
			wantErr: false,
		},
		{
			name:    "Invalid secret",
			token:   mustMakeJWT(userID, secret, time.Hour),
			secret:  "wrong-secret",
			wantErr: true,
		},
		{
			name:    "Expired token",
			token:   mustMakeJWT(userID, secret, -time.Hour),
			secret:  secret,
			wantErr: true,
		},
		{
			name:    "Malformed token",
			token:   "not.a.valid.token",
			secret:  secret,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.ValidateJWT(tt.token, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJWT() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func mustMakeJWT(userID uuid.UUID, secret string, duration time.Duration) string {
	token, err := auth.MakeJWT(userID, secret, duration)
	if err != nil {
		panic(err)
	}
	return token
}
