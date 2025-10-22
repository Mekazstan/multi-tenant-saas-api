package main

import (
	"context"
	"os"
	"testing"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestDatabaseIntegration tests database operations
// Run with: go test -tags=integration
func TestDatabaseIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Connect to test database
	ctx := context.Background()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	db := database.New(pool)

	// Test organization creation
	t.Run("CreateOrganization", func(t *testing.T) {
		org, err := db.CreateOrganization(ctx, database.CreateOrganizationParams{
			Name:  "Test Org",
			Email: "test@example.com",
			Plan:  database.PlanTypeFree,
		})
		if err != nil {
			t.Fatalf("CreateOrganization() error = %v", err)
		}

		if org.Name != "Test Org" {
			t.Errorf("Expected name 'Test Org', got '%s'", org.Name)
		}

		// Cleanup
		defer db.DeleteOrganization(ctx, org.ID)
	})

	// Test user creation
	t.Run("CreateUser", func(t *testing.T) {
		// Create organization first
		org, _ := db.CreateOrganization(ctx, database.CreateOrganizationParams{
			Name:  "User Test Org",
			Email: "usertest@example.com",
			Plan:  database.PlanTypeFree,
		})
		defer db.DeleteOrganization(ctx, org.ID)

		user, err := db.CreateUser(ctx, database.CreateUserParams{
			OrganizationID: org.ID,
			Email:          "user@example.com",
			PasswordHash:   "hashedpassword",
			Role:           database.UserRoleOwner,
		})
		if err != nil {
			t.Fatalf("CreateUser() error = %v", err)
		}

		if user.Email != "user@example.com" {
			t.Errorf("Expected email 'user@example.com', got '%s'", user.Email)
		}
	})

	// Test API key creation
	t.Run("CreateAPIKey", func(t *testing.T) {
		// Setup
		org, _ := db.CreateOrganization(ctx, database.CreateOrganizationParams{
			Name:  "API Key Test Org",
			Email: "apitest@example.com",
			Plan:  database.PlanTypeFree,
		})
		defer db.DeleteOrganization(ctx, org.ID)

		apiKey, err := db.CreateAPIKey(ctx, database.CreateAPIKeyParams{
			OrganizationID: org.ID,
			Key:            "sk_live_test123",
			Name:           "Test Key",
			IsActive:       true,
		})
		if err != nil {
			t.Fatalf("CreateAPIKey() error = %v", err)
		}

		if apiKey.Name != "Test Key" {
			t.Errorf("Expected name 'Test Key', got '%s'", apiKey.Name)
		}

		// Test retrieval
		retrieved, err := db.GetAPIKeyByKey(ctx, "sk_live_test123")
		if err != nil {
			t.Fatalf("GetAPIKeyByKey() error = %v", err)
		}

		if retrieved.ID != apiKey.ID {
			t.Error("Retrieved API key doesn't match created key")
		}
	})
}
