package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/auth"
	"github.com/Mekazstan/multi-tenant-saas-api/internal/database"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
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

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "An unexpected error occurred. Please try again later.",
		})
		return
	}

	org, err := cfg.db.CreateOrganization(r.Context(), database.CreateOrganizationParams{
		Name:  params.OrganizationName,
		Email: params.Email,
		Plan:  database.PlanTypeFree,
	})

	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" {
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

	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		OrganizationID: org.ID,
		Email:          params.Email,
		PasswordHash:   hashedPassword,
		Role:           database.UserRoleOwner,
	})

	if err != nil {
		cfg.db.DeleteOrganization(r.Context(), org.ID)

		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to create user account",
		})
		return
	}

	accessToken, err := auth.MakeJWT(
		user.ID,
		cfg.jwtSecret,
		time.Hour*24*7,
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to generate authentication token",
		})
		return
	}

	go func() {
		cfg.emailService.SendWelcomeEmail(params.Email, params.FullName, params.OrganizationName)
	}()

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

	org, err := cfg.db.GetOrganization(r.Context(), user.OrganizationID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve organization details",
		})
		return
	}

	expiresIn := time.Hour * 24 * 7
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

func (cfg *apiConfig) verifyEmailHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Token string `json:"token"`
	}

	var params parameters
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
		})
		return
	}

	if params.Token == "" {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "Token is required",
		})
		return
	}

	authToken, err := cfg.db.GetAuthToken(r.Context(), params.Token)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_TOKEN",
			Message: "Invalid or expired verification token",
		})
		return
	}

	if authToken.Type != database.TokenTypeEmailVerification {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_TOKEN_TYPE",
			Message: "This token is not for email verification",
		})
		return
	}

	_, err = cfg.db.MarkTokenAsUsed(r.Context(), authToken.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to process verification",
		})
		return
	}

	if !authToken.UserID.Valid {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_TOKEN",
			Message: "Invalid token",
		})
		return
	}

	user, err := cfg.db.VerifyUserEmail(r.Context(), authToken.UserID.Bytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to verify email",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "Email verified successfully",
		Data: map[string]interface{}{
			"email":         user.Email,
			"verified_at":   user.EmailVerifiedAt,
			"email_verified": user.EmailVerified,
		},
	})
}

func (cfg *apiConfig) requestEmailVerificationHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, ApiError{
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	user, err := cfg.db.GetUser(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve user details",
		})
		return
	}

	if user.EmailVerified {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "ALREADY_VERIFIED",
			Message: "Email is already verified",
		})
		return
	}

	token := generateSecureToken(32)
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err = cfg.db.CreateAuthToken(r.Context(), database.CreateAuthTokenParams{
		UserID: pgtype.UUID{
			Bytes: userID,
			Valid: true,
		},
		Token:     token,
		Type:      database.TokenTypeEmailVerification,
		ExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to generate verification token",
		})
		return
	}

	go func() {
		cfg.emailService.SendEmailVerification(user.Email, user.Email, token)
	}()

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "Verification email sent successfully",
	})
}

func (cfg *apiConfig) requestPasswordResetHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	var params parameters
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
		})
		return
	}

	if params.Email == "" {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "Email is required",
		})
		return
	}

	user, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		respondWithJSON(w, http.StatusOK, ApiResponse{
			Success: true,
			Message: "If an account exists with this email, you will receive a password reset link",
		})
		return
	}

	token := generateSecureToken(32)
	expiresAt := time.Now().Add(1 * time.Hour)

	_, err = cfg.db.CreateAuthToken(r.Context(), database.CreateAuthTokenParams{
		UserID: pgtype.UUID{
			Bytes: user.ID,
			Valid: true,
		},
		Token:     token,
		Type:      database.TokenTypePasswordReset,
		ExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to generate reset token",
		})
		return
	}

	go func() {
		cfg.emailService.SendPasswordReset(user.Email, user.Email, token)
	}()

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "If an account exists with this email, you will receive a password reset link",
	})
}

func (cfg *apiConfig) resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}

	var params parameters
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
		})
		return
	}

	if params.Token == "" || params.NewPassword == "" {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "Token and new password are required",
		})
		return
	}

	if len(params.NewPassword) < 8 {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "Password must be at least 8 characters",
		})
		return
	}

	authToken, err := cfg.db.GetAuthToken(r.Context(), params.Token)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_TOKEN",
			Message: "Invalid or expired reset token",
		})
		return
	}

	if authToken.Type != database.TokenTypePasswordReset {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_TOKEN_TYPE",
			Message: "This token is not for password reset",
		})
		return
	}

	hashedPassword, err := auth.HashPassword(params.NewPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to process password",
		})
		return
	}

	_, err = cfg.db.MarkTokenAsUsed(r.Context(), authToken.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to process reset",
		})
		return
	}

	if !authToken.UserID.Valid {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_TOKEN",
			Message: "Invalid token",
		})
		return
	}

	_, err = cfg.db.UpdateUserPassword(r.Context(), database.UpdateUserPasswordParams{
		PasswordHash: hashedPassword,
		ID:           authToken.UserID.Bytes,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to update password",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "Password reset successfully",
	})
}

func generateSecureToken(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}