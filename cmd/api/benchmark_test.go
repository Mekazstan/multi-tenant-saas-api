package main

import (
	"testing"
	"time"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/auth"
	"github.com/google/uuid"
)

func BenchmarkGenerateAPIKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateAPIKey()
	}
}

func BenchmarkHashPassword(b *testing.B) {
	password := "testpassword123"
	for i := 0; i < b.N; i++ {
		auth.HashPassword(password)
	}
}

func BenchmarkCheckPasswordHash(b *testing.B) {
	password := "testpassword123"
	hash, _ := auth.HashPassword(password)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.CheckPasswordHash(password, hash)
	}
}

func BenchmarkMakeJWT(b *testing.B) {
	userID := uuid.New()
	secret := "test-secret"
	duration := time.Hour
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.MakeJWT(userID, secret, duration)
	}
}

func BenchmarkValidateJWT(b *testing.B) {
	userID := uuid.New()
	secret := "test-secret"
	token, _ := auth.MakeJWT(userID, secret, time.Hour)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.ValidateJWT(token, secret)
	}
}