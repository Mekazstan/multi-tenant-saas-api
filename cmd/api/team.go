package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/auth"
	"github.com/Mekazstan/multi-tenant-saas-api/internal/database"
	"github.com/Mekazstan/multi-tenant-saas-api/internal/email"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (cfg *apiConfig) inviteTeamMemberHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, ApiError{
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
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

	if params.Role != "admin" && params.Role != "member" {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "Role must be 'admin' or 'member'",
		})
		return
	}

	inviter, err := cfg.db.GetUser(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve user details",
		})
		return
	}

	if inviter.Role != database.UserRoleOwner && inviter.Role != database.UserRoleAdmin {
		respondWithError(w, http.StatusForbidden, ApiError{
			Code:    "PERMISSION_DENIED",
			Message: "Only owners and admins can invite team members",
		})
		return
	}

	org, err := cfg.db.GetOrganization(r.Context(), inviter.OrganizationID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve organization",
		})
		return
	}

	existingUser, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if err == nil && existingUser.OrganizationID == inviter.OrganizationID {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "USER_EXISTS",
			Message: "This user is already a member of your organization",
		})
		return
	}

	pendingInvite, err := cfg.db.GetPendingInvitationByEmail(r.Context(), 
		database.GetPendingInvitationByEmailParams{
			OrganizationID: inviter.OrganizationID,
			Email:          params.Email,
		})
	if err == nil && pendingInvite.ID != uuid.Nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVITATION_EXISTS",
			Message: "There is already a pending invitation for this email",
		})
		return
	}

	var role database.UserRole
	switch params.Role {
	case "admin":
		role = database.UserRoleAdmin
	case "member":
		role = database.UserRoleMember
	default:
		role = database.UserRoleMember
	}

	token := generateSecureToken(32)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	invitation, err := cfg.db.CreateTeamInvitation(r.Context(), database.CreateTeamInvitationParams{
		OrganizationID: inviter.OrganizationID,
		Email:          params.Email,
		Role:           role,
		InvitedBy:      userID,
		Token:          token,
		ExpiresAt:      pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to create invitation",
		})
		return
	}

	go func() {
		cfg.emailService.SendTeamInvitation(params.Email, email.TeamInvitationData{
			InviterName:      inviter.Email,
			OrganizationName: org.Name,
			Role:             params.Role,
			InvitationURL:    cfg.config.AppURL + "/accept-invitation?token=" + token,
			ExpiresIn:        "7 days",
		})
	}()

	respondWithJSON(w, http.StatusCreated, ApiResponse{
		Success: true,
		Message: "Invitation sent successfully",
		Data: map[string]interface{}{
			"invitation_id": invitation.ID,
			"email":         invitation.Email,
			"role":          invitation.Role,
			"expires_at":    invitation.ExpiresAt,
		},
	})
}

func (cfg *apiConfig) acceptTeamInvitationHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Token    string `json:"token"`
		Password string `json:"password"`
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

	invitation, err := cfg.db.GetTeamInvitationByToken(r.Context(), params.Token)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_TOKEN",
			Message: "Invalid or expired invitation token",
		})
		return
	}

	existingUser, err := cfg.db.GetUserByEmail(r.Context(), invitation.Email)
	
	var user database.User
	
	if err != nil {
		if params.Password == "" || len(params.Password) < 8 {
			respondWithError(w, http.StatusBadRequest, ApiError{
				Code:    "VALIDATION_ERROR",
				Message: "Password is required and must be at least 8 characters",
			})
			return
		}

		hashedPassword, err := auth.HashPassword(params.Password)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, ApiError{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to process password",
			})
			return
		}

		user, err = cfg.db.CreateUser(r.Context(), database.CreateUserParams{
			OrganizationID: invitation.OrganizationID,
			Email:          invitation.Email,
			PasswordHash:   hashedPassword,
			Role:           invitation.Role,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, ApiError{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create user account",
			})
			return
		}

		go func() {
			cfg.emailService.SendWelcomeEmail(user.Email, user.Email, invitation.OrganizationName)
		}()
	} else {
		if existingUser.OrganizationID != invitation.OrganizationID {
			respondWithError(w, http.StatusBadRequest, ApiError{
				Code:    "USER_IN_DIFFERENT_ORG",
				Message: "This email is already associated with a different organization",
			})
			return
		}
		user = existingUser
	}

	_, err = cfg.db.AcceptTeamInvitation(r.Context(), invitation.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to accept invitation",
		})
		return
	}

	accessToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Hour*24*7)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to generate authentication token",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "Invitation accepted successfully",
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"id":              user.ID,
				"email":           user.Email,
				"role":            user.Role,
				"organization_id": user.OrganizationID,
			},
			"token": accessToken,
		},
	})
}

func (cfg *apiConfig) declineTeamInvitationHandler(w http.ResponseWriter, r *http.Request) {
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

	invitation, err := cfg.db.GetTeamInvitationByToken(r.Context(), params.Token)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_TOKEN",
			Message: "Invalid or expired invitation token",
		})
		return
	}

	_, err = cfg.db.DeclineTeamInvitation(r.Context(), invitation.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to decline invitation",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "Invitation declined",
	})
}

func (cfg *apiConfig) listOrganizationMembersHandler(w http.ResponseWriter, r *http.Request) {
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

	members, err := cfg.db.ListOrganizationMembers(r.Context(), user.OrganizationID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve team members",
		})
		return
	}

	invitations, err := cfg.db.ListOrganizationInvitations(r.Context(), user.OrganizationID)
	if err != nil {
		invitations = []database.ListOrganizationInvitationsRow{}
	}

	pendingInvites := make([]map[string]interface{}, 0)
	for _, inv := range invitations {
		if !inv.AcceptedAt.Valid && !inv.DeclinedAt.Valid {
			pendingInvites = append(pendingInvites, map[string]interface{}{
				"id":         inv.ID,
				"email":      inv.Email,
				"role":       inv.Role,
				"invited_by": inv.InviterEmail,
				"expires_at": inv.ExpiresAt,
				"created_at": inv.CreatedAt,
			})
		}
	}

	membersList := make([]map[string]interface{}, 0)
	for _, member := range members {
		membersList = append(membersList, map[string]interface{}{
			"id":             member.ID,
			"email":          member.Email,
			"role":           member.Role,
			"email_verified": member.EmailVerified,
			"created_at":     member.CreatedAt,
		})
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"members":            membersList,
			"pending_invitations": pendingInvites,
			"total_members":      len(membersList),
			"total_pending":      len(pendingInvites),
		},
	})
}

func (cfg *apiConfig) removeTeamMemberHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, ApiError{
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	memberIDStr := r.PathValue("id")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_MEMBER_ID",
			Message: "Invalid member ID format",
		})
		return
	}

	currentUser, err := cfg.db.GetUser(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve user details",
		})
		return
	}

	if currentUser.Role != database.UserRoleOwner && currentUser.Role != database.UserRoleAdmin {
		respondWithError(w, http.StatusForbidden, ApiError{
			Code:    "PERMISSION_DENIED",
			Message: "Only owners and admins can remove team members",
		})
		return
	}

	if memberID == userID {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "CANNOT_REMOVE_SELF",
			Message: "You cannot remove yourself from the organization",
		})
		return
	}

	err = cfg.db.RemoveTeamMember(r.Context(), database.RemoveTeamMemberParams{
		ID:             memberID,
		OrganizationID: currentUser.OrganizationID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to remove team member. Cannot remove organization owner.",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "Team member removed successfully",
	})
}

func (cfg *apiConfig) updateUserRoleHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Role string `json:"role"`
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, ApiError{
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	memberIDStr := r.PathValue("id")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_MEMBER_ID",
			Message: "Invalid member ID format",
		})
		return
	}

	var params parameters
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
		})
		return
	}

	if params.Role != "admin" && params.Role != "member" {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "Role must be 'admin' or 'member'",
		})
		return
	}

	currentUser, err := cfg.db.GetUser(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve user details",
		})
		return
	}

	if currentUser.Role != database.UserRoleOwner {
		respondWithError(w, http.StatusForbidden, ApiError{
			Code:    "PERMISSION_DENIED",
			Message: "Only organization owner can change user roles",
		})
		return
	}

	var newRole database.UserRole
	switch params.Role {
	case "admin":
		newRole = database.UserRoleAdmin
	case "member":
		newRole = database.UserRoleMember
	default:
		newRole = database.UserRoleMember
	}

	updatedUser, err := cfg.db.UpdateUserRole(r.Context(), database.UpdateUserRoleParams{
		Role:           newRole,
		ID:             memberID,
		OrganizationID: currentUser.OrganizationID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to update user role. Cannot change owner role.",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "User role updated successfully",
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"id":    updatedUser.ID,
				"email": updatedUser.Email,
				"role":  updatedUser.Role,
			},
		},
	})
}

func (cfg *apiConfig) cancelInvitationHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, ApiError{
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	invitationIDStr := r.PathValue("id")
	invitationID, err := uuid.Parse(invitationIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_INVITATION_ID",
			Message: "Invalid invitation ID format",
		})
		return
	}

	currentUser, err := cfg.db.GetUser(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve user details",
		})
		return
	}

	if currentUser.Role != database.UserRoleOwner && currentUser.Role != database.UserRoleAdmin {
		respondWithError(w, http.StatusForbidden, ApiError{
			Code:    "PERMISSION_DENIED",
			Message: "Only owners and admins can cancel invitations",
		})
		return
	}

	_, err = cfg.db.CancelInvitation(r.Context(), database.CancelInvitationParams{
		ID:             invitationID,
		OrganizationID: currentUser.OrganizationID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to cancel invitation",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "Invitation cancelled successfully",
	})
}