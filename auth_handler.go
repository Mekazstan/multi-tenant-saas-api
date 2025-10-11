package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/auth"
	"github.com/Mekazstan/multi-tenant-saas-api/internal/database"
	"github.com/jackc/pgx/v5/pgconn"
)

func (cfg *apiConfig) registerHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		OrganizationName string `json:"organization_name"`
		Email            string `json:"email"`
		Password         string `json:"password"`
		FullName         string `json:"full_name"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
		})
		return
	}

	// Validation
	if params.OrganizationName == "" {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "Organization name is required",
			Details: map[string]interface{}{
				"field":  "organization_name",
				"reason": "This field cannot be empty",
			},
		})
		return
	}

	if params.Email == "" {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "Email is required",
			Details: map[string]interface{}{
				"field":  "email",
				"reason": "This field cannot be empty",
			},
		})
		return
	}

	if params.Password == "" || len(params.Password) < 8 {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "Password must be at least 8 characters",
			Details: map[string]interface{}{
				"field":  "password",
				"reason": "Password must be at least 8 characters long",
			},
		})
		return
	}

	// Hash Password
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "An unexpected error occurred. Please try again later.",
		})
		return
	}

	// Create Organization
	org, err := cfg.db.CreateOrganization(r.Context(), database.CreateOrganizationParams{
		Name:  params.OrganizationName,
		Email: params.Email,
		Plan:  database.PlanTypeFree,
	})

	if err != nil {
		// Check for duplicate email
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" { // unique_violation
				respondWithError(w, http.StatusBadRequest, ApiError{
					Code:    "VALIDATION_ERROR",
					Message: "Email already exists",
					Details: map[string]interface{}{
						"field":  "email",
						"reason": "An organization with this email already exists",
					},
				})
				return
			}
		}
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to create organization",
		})
		return
	}

	// Create User
	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		OrganizationID: org.ID,
		Email:          params.Email,
		PasswordHash:   hashedPassword,
		Role:           database.UserRoleOwner,
	})

	if err != nil {
		// Try to clean up organization if user creation fails
		cfg.db.DeleteOrganization(r.Context(), org.ID)

		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to create user account",
		})
		return
	}

	// Generate JWT token
	accessToken, err := auth.MakeJWT(
		user.ID,
		cfg.jwtSecret,
		time.Hour*24*7, // 7 days
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to generate authentication token",
		})
		return
	}

	respondWithJSON(w, http.StatusCreated, ApiResponse{
		Success: true,
		Message: "Organization created successfully",
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"id":         user.ID,
				"email":      user.Email,
				"role":       user.Role,
				"created_at": user.CreatedAt,
			},
			"organization": map[string]interface{}{
				"id":         org.ID,
				"name":       org.Name,
				"email":      org.Email,
				"plan":       org.Plan,
				"created_at": org.CreatedAt,
			},
			"token": accessToken,
		},
	})
}

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
		})
		return
	}

	// Validation
	if params.Email == "" || params.Password == "" {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "Email and password are required",
		})
		return
	}

	user, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, ApiError{
			Code:    "INVALID_CREDENTIALS",
			Message: "Invalid email or password",
		})
		return
	}

	match, err := auth.CheckPasswordHash(params.Password, user.PasswordHash)
	if err != nil || !match {
		respondWithError(w, http.StatusUnauthorized, ApiError{
			Code:    "INVALID_CREDENTIALS",
			Message: "Invalid email or password",
		})
		return
	}

	// Get organization details
	org, err := cfg.db.GetOrganization(r.Context(), user.OrganizationID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve organization details",
		})
		return
	}

	expiresIn := time.Hour * 24 * 7 // 7 days
	accessToken, err := auth.MakeJWT(
		user.ID,
		cfg.jwtSecret,
		expiresIn,
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to generate authentication token",
		})
		return
	}

	expiresAt := time.Now().Add(expiresIn)

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"id":              user.ID,
				"email":           user.Email,
				"role":            user.Role,
				"organization_id": user.OrganizationID,
			},
			"organization": map[string]interface{}{
				"id":   org.ID,
				"name": org.Name,
				"plan": org.Plan,
			},
			"token":      accessToken,
			"expires_at": expiresAt,
		},
	})
}
