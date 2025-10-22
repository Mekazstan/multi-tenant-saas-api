package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (cfg *apiConfig) getCurrentUserHandler(w http.ResponseWriter, r *http.Request) {
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
		respondWithError(w, http.StatusNotFound, ApiError{
			Code:    "USER_NOT_FOUND",
			Message: "User not found",
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

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"id":              user.ID,
				"email":           user.Email,
				"role":            user.Role,
				"organization_id": user.OrganizationID,
				"created_at":      user.CreatedAt,
			},
			"organization": map[string]interface{}{
				"id":         org.ID,
				"name":       org.Name,
				"email":      org.Email,
				"plan":       org.Plan,
				"created_at": org.CreatedAt,
			},
		},
	})
}

func (cfg *apiConfig) createAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Name string `json:"name"`
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

	if params.Name == "" {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "API key name is required",
			Details: map[string]interface{}{
				"field":  "name",
				"reason": "This field cannot be empty",
			},
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

	if user.Role != database.UserRoleOwner && user.Role != database.UserRoleAdmin {
		respondWithError(w, http.StatusForbidden, ApiError{
			Code:    "PERMISSION_DENIED",
			Message: "Only owner and admin roles can create API keys",
		})
		return
	}

	keyString := generateAPIKey()

	apiKey, err := cfg.db.CreateAPIKey(r.Context(), database.CreateAPIKeyParams{
		OrganizationID: user.OrganizationID,
		Key:            keyString,
		Name:           params.Name,
		IsActive:       true,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to create API key",
		})
		return
	}

	respondWithJSON(w, http.StatusCreated, ApiResponse{
		Success: true,
		Message: "API key created successfully. ⚠️ Store this key securely. It won't be shown again.",
		Data: map[string]interface{}{
			"id":           apiKey.ID,
			"name":         apiKey.Name,
			"key":          apiKey.Key,
			"is_active":    apiKey.IsActive,
			"created_at":   apiKey.CreatedAt,
			"last_used_at": apiKey.LastUsedAt,
		},
	})
}

func (cfg *apiConfig) listAPIKeysHandler(w http.ResponseWriter, r *http.Request) {
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

	apiKeys, err := cfg.db.ListOrganizationAPIKeys(r.Context(), user.OrganizationID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve API keys",
		})
		return
	}

	keys := make([]map[string]interface{}, 0, len(apiKeys))
	for _, key := range apiKeys {
		keys = append(keys, map[string]interface{}{
			"id":           key.ID,
			"name":         key.Name,
			"key":          maskAPIKey(key.Key),
			"is_active":    key.IsActive,
			"created_at":   key.CreatedAt,
			"last_used_at": key.LastUsedAt,
		})
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"keys": keys,
			"pagination": map[string]interface{}{
				"total": len(keys),
			},
		},
	})
}

func (cfg *apiConfig) revokeAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, ApiError{
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	keyIDStr := r.PathValue("id")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_KEY_ID",
			Message: "Invalid API key ID format",
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

	if user.Role != database.UserRoleOwner && user.Role != database.UserRoleAdmin {
		respondWithError(w, http.StatusForbidden, ApiError{
			Code:    "PERMISSION_DENIED",
			Message: "Only owner and admin roles can revoke API keys",
		})
		return
	}

	apiKey, err := cfg.db.GetAPIKey(r.Context(), keyID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, ApiError{
			Code:    "KEY_NOT_FOUND",
			Message: "API key not found",
		})
		return
	}

	if apiKey.OrganizationID != user.OrganizationID {
		respondWithError(w, http.StatusForbidden, ApiError{
			Code:    "PERMISSION_DENIED",
			Message: "You don't have permission to revoke this API key",
		})
		return
	}

	_, err = cfg.db.DeactivateAPIKey(r.Context(), keyID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to revoke API key",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "API key revoked successfully",
	})
}

func (cfg *apiConfig) sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		To      string `json:"to"`
		Message string `json:"message"`
		Type    string `json:"type"`
	}

	var params parameters
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
		})
		return
	}

	if params.To == "" {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "Recipient is required",
			Details: map[string]interface{}{
				"field":  "to",
				"reason": "This field cannot be empty",
			},
		})
		return
	}

	if params.Message == "" {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "Message content is required",
			Details: map[string]interface{}{
				"field":  "message",
				"reason": "This field cannot be empty",
			},
		})
		return
	}

	if params.Type != "sms" && params.Type != "email" {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "VALIDATION_ERROR",
			Message: "Invalid message type",
			Details: map[string]interface{}{
				"field":    "type",
				"reason":   "Must be either 'sms' or 'email'",
				"provided": params.Type,
			},
		})
		return
	}

	orgID, _ := GetOrgID(r.Context())

	cost := 0.01
	if params.Type == "email" {
		cost = 0.001
	}

	messageID := fmt.Sprintf("msg_%s", generateRandomString(32))

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"message_id":      messageID,
			"to":              params.To,
			"status":          "queued",
			"type":            params.Type,
			"cost":            cost,
			"usage_recorded":  true,
			"organization_id": orgID,
			"created_at":      time.Now(),
		},
	})
}

func (cfg *apiConfig) getMessageStatusHandler(w http.ResponseWriter, r *http.Request) {
	messageID := r.PathValue("id")

	if messageID == "" {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_MESSAGE_ID",
			Message: "Message ID is required",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"message_id":   messageID,
			"status":       "delivered",
			"sent_at":      time.Now().Add(-5 * time.Minute),
			"delivered_at": time.Now().Add(-3 * time.Minute),
		},
	})
}

func generateAPIKey() string {
	prefix := "sk_live_"
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return prefix + hex.EncodeToString(bytes)
}

func maskAPIKey(key string) string {
	if len(key) <= 12 {
		return key
	}
	return key[:12] + strings.Repeat("*", len(key)-16) + key[len(key)-4:]
}

func generateRandomString(length int) string {
	bytes := make([]byte, length/2)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (cfg *apiConfig) getBillingUsageHandler(w http.ResponseWriter, r *http.Request) {
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

	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	var startDate, endDate time.Time
	if startDateStr != "" {
		startDate, err = time.Parse("2025-10-02", startDateStr)
		if err != nil {
			startDate = time.Now().AddDate(0, 0, -30)
		}
	} else {
		startDate = time.Now().AddDate(0, 0, -30)
	}

	if endDateStr != "" {
		endDate, err = time.Parse("2025-10-02", endDateStr)
		if err != nil {
			endDate = time.Now()
		}
	} else {
		endDate = time.Now()
	}

	startDatePg := pgtype.Timestamp{Time: startDate, Valid: true}
	endDatePg := pgtype.Timestamp{Time: endDate, Valid: true}

	totalRequests, err := cfg.db.CountOrganizationUsage(r.Context(), database.CountOrganizationUsageParams{
		OrganizationID: user.OrganizationID,
		CreatedAt:      startDatePg,
		CreatedAt_2:    endDatePg,
	})
	if err != nil {
		totalRequests = 0
	}

	usageByEndpoint, err := cfg.db.GetUsageByEndpoint(r.Context(), database.GetUsageByEndpointParams{
		OrganizationID: user.OrganizationID,
		CreatedAt:      startDatePg,
		CreatedAt_2:    endDatePg,
	})
	if err != nil {
		usageByEndpoint = []database.GetUsageByEndpointRow{}
	}

	usageByAPIKey, err := cfg.db.GetUsageByAPIKey(r.Context(), database.GetUsageByAPIKeyParams{
		OrganizationID: user.OrganizationID,
		CreatedAt:      startDatePg,
		CreatedAt_2:    endDatePg,
	})
	if err != nil {
		usageByAPIKey = []database.GetUsageByAPIKeyRow{}
	}

	dailyStats, err := cfg.db.GetDailyUsageStats(r.Context(), database.GetDailyUsageStatsParams{
		OrganizationID: user.OrganizationID,
		CreatedAt:      startDatePg,
		CreatedAt_2:    endDatePg,
	})
	if err != nil {
		dailyStats = []database.GetDailyUsageStatsRow{}
	}

	var successCount, errorCount int64
	totalCost := 0.0
	endpointData := make([]map[string]interface{}, 0)

	for _, endpoint := range usageByEndpoint {
		cost := float64(endpoint.RequestCount) * 0.01 // $0.01 per request
		totalCost += cost
		successCount += endpoint.SuccessCount
		errorCount += endpoint.ErrorCount

		endpointData = append(endpointData, map[string]interface{}{
			"endpoint":      endpoint.Endpoint,
			"requests":      endpoint.RequestCount,
			"success_count": endpoint.SuccessCount,
			"error_count":   endpoint.ErrorCount,
			"cost":          cost,
		})
	}

	successRate := 0.0
	if totalRequests > 0 {
		successRate = (float64(successCount) / float64(totalRequests)) * 100
	}

	apiKeyData := make([]map[string]interface{}, 0)
	for _, key := range usageByAPIKey {
		apiKeyData = append(apiKeyData, map[string]interface{}{
			"id":       key.ID,
			"name":     key.Name,
			"key":      maskAPIKey(key.Key),
			"requests": key.RequestCount,
			"cost":     float64(key.RequestCount) * 0.01,
		})
	}

	dailyData := make([]map[string]interface{}, 0)
	for _, day := range dailyStats {
		dailyData = append(dailyData, map[string]interface{}{
			"date":          day.Date,
			"requests":      day.RequestCount,
			"success_count": day.SuccessCount,
			"error_count":   day.ErrorCount,
			"cost":          float64(day.RequestCount) * 0.01,
		})
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"summary": map[string]interface{}{
				"total_requests":      totalRequests,
				"successful_requests": successCount,
				"failed_requests":     errorCount,
				"success_rate":        successRate,
				"total_cost":          totalCost,
				"period": map[string]interface{}{
					"start": startDate,
					"end":   endDate,
				},
			},
			"by_endpoint":     endpointData,
			"by_api_key":      apiKeyData,
			"daily_breakdown": dailyData,
		},
	})
}

func (cfg *apiConfig) getBillingHistoryHandler(w http.ResponseWriter, r *http.Request) {
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

	page := int32(1)
	limit := int32(20)

	cycles, err := cfg.db.ListOrganizationBillingCycles(r.Context(), database.ListOrganizationBillingCyclesParams{
		OrganizationID: user.OrganizationID,
		Limit:          limit,
		Offset:         (page - 1) * limit,
	})
	if err != nil {
		cycles = []database.BillingCycle{}
	}

	invoices := make([]map[string]interface{}, 0)
	var totalBilled, totalPaid, outstanding float64

	for _, cycle := range cycles {
		amount := numericToFloat64(cycle.TotalAmount)
		totalBilled += amount

		switch cycle.Status {
		case database.BillingStatusPaid:
			totalPaid += amount
		case database.BillingStatusPending, database.BillingStatusOverdue:
			outstanding += amount
		}

		invoice := map[string]interface{}{
			"id": cycle.ID,
			"period": map[string]interface{}{
				"start": cycle.PeriodStart,
				"end":   cycle.PeriodEnd,
			},
			"total_requests": cycle.TotalRequests,
			"total_amount":   amount,
			"status":         cycle.Status,
			"created_at":     cycle.CreatedAt,
		}

		if cycle.Status == database.BillingStatusPaid {
			invoice["paid_at"] = cycle.CreatedAt
		}

		dueDate := cycle.PeriodEnd.Time.Add(7 * 24 * time.Hour)
		invoice["due_date"] = dueDate

		invoices = append(invoices, invoice)
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"invoices": invoices,
			"pagination": map[string]interface{}{
				"page":        page,
				"limit":       limit,
				"total":       len(invoices),
				"total_pages": 1,
			},
			"summary": map[string]interface{}{
				"total_billed": totalBilled,
				"total_paid":   totalPaid,
				"outstanding":  outstanding,
			},
		},
	})
}

func (cfg *apiConfig) calculateCurrentBillHandler(w http.ResponseWriter, r *http.Request) {
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

	currentCycle, err := cfg.db.GetCurrentBillingCycle(r.Context(), user.OrganizationID)
	if err != nil {
		now := time.Now()
		periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

		startPeriodPg := pgtype.Timestamp{Time: periodStart, Valid: true}
		endPeriodPg := pgtype.Timestamp{Time: periodEnd, Valid: true}

		totalRequests, _ := cfg.db.CountOrganizationUsage(r.Context(), database.CountOrganizationUsageParams{
			OrganizationID: user.OrganizationID,
			CreatedAt:      startPeriodPg,
			CreatedAt_2:    endPeriodPg,
		})

		totalAmount := float64(totalRequests) * 0.01

		respondWithJSON(w, http.StatusOK, ApiResponse{
			Success: true,
			Data: map[string]interface{}{
				"period": map[string]interface{}{
					"start": periodStart,
					"end":   periodEnd,
				},
				"total_requests": totalRequests,
				"total_amount":   totalAmount,
				"status":         "calculating",
				"note":           "Current billing period - invoice not yet generated",
			},
		})
		return
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"id": currentCycle.ID,
			"period": map[string]interface{}{
				"start": currentCycle.PeriodStart,
				"end":   currentCycle.PeriodEnd,
			},
			"total_requests": currentCycle.TotalRequests,
			"total_amount":   numericToFloat64(currentCycle.TotalAmount),
			"status":         currentCycle.Status,
			"created_at":     currentCycle.CreatedAt,
		},
	})
}

func (cfg *apiConfig) getDashboardStatsHandler(w http.ResponseWriter, r *http.Request) {
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

	org, err := cfg.db.GetOrganization(r.Context(), user.OrganizationID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve organization details",
		})
		return
	}

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	startDatePg := pgtype.Timestamp{Time: startDate, Valid: true}
	endDatePg := pgtype.Timestamp{Time: endDate, Valid: true}

	totalRequests, _ := cfg.db.CountOrganizationUsage(r.Context(), database.CountOrganizationUsageParams{
		OrganizationID: user.OrganizationID,
		CreatedAt:      startDatePg,
		CreatedAt_2:    endDatePg,
	})

	apiKeys, _ := cfg.db.ListOrganizationAPIKeys(r.Context(), user.OrganizationID)
	activeKeys := 0
	for _, key := range apiKeys {
		if key.IsActive {
			activeKeys++
		}
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := now

	startMonth := pgtype.Timestamp{Time: monthStart, Valid: true}
	endMonth := pgtype.Timestamp{Time: monthEnd, Valid: true}

	monthRequests, _ := cfg.db.CountOrganizationUsage(r.Context(), database.CountOrganizationUsageParams{
		OrganizationID: user.OrganizationID,
		CreatedAt:      startMonth,
		CreatedAt_2:    endMonth,
	})

	currentMonthCost := float64(monthRequests) * 0.01

	usageByEndpoint, _ := cfg.db.GetUsageByEndpoint(r.Context(), database.GetUsageByEndpointParams{
		OrganizationID: user.OrganizationID,
		CreatedAt:      startDatePg,
		CreatedAt_2:    endDatePg,
	})

	var successCount, errorCount int64
	for _, endpoint := range usageByEndpoint {
		successCount += endpoint.SuccessCount
		errorCount += endpoint.ErrorCount
	}

	successRate := 0.0
	if totalRequests > 0 {
		successRate = (float64(successCount) / float64(totalRequests)) * 100
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"organization": map[string]interface{}{
				"name": org.Name,
				"plan": org.Plan,
			},
			"stats": map[string]interface{}{
				"total_requests_30d":     totalRequests,
				"current_month_requests": monthRequests,
				"current_month_cost":     currentMonthCost,
				"success_rate":           successRate,
				"active_api_keys":        activeKeys,
				"total_api_keys":         len(apiKeys),
			},
		},
	})
}

func (cfg *apiConfig) getDashboardUsageGraphHandler(w http.ResponseWriter, r *http.Request) {
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

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	startDatePg := pgtype.Timestamp{Time: startDate, Valid: true}
	endDatePg := pgtype.Timestamp{Time: endDate, Valid: true}

	dailyStats, err := cfg.db.GetDailyUsageStats(r.Context(), database.GetDailyUsageStatsParams{
		OrganizationID: user.OrganizationID,
		CreatedAt:      startDatePg,
		CreatedAt_2:    endDatePg,
	})
	if err != nil {
		dailyStats = []database.GetDailyUsageStatsRow{}
	}

	graphData := make([]map[string]interface{}, 0)
	for _, day := range dailyStats {
		graphData = append(graphData, map[string]interface{}{
			"date":          day.Date,
			"requests":      day.RequestCount,
			"success_count": day.SuccessCount,
			"error_count":   day.ErrorCount,
		})
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"period": map[string]interface{}{
				"start": startDate,
				"end":   endDate,
			},
			"data": graphData,
		},
	})
}

func (cfg *apiConfig) getDashboardAPIKeysHandler(w http.ResponseWriter, r *http.Request) {
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

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	startDatePg := pgtype.Timestamp{Time: startDate, Valid: true}
	endDatePg := pgtype.Timestamp{Time: endDate, Valid: true}

	usageByAPIKey, err := cfg.db.GetUsageByAPIKey(r.Context(), database.GetUsageByAPIKeyParams{
		OrganizationID: user.OrganizationID,
		CreatedAt:      startDatePg,
		CreatedAt_2:    endDatePg,
	})
	if err != nil {
		usageByAPIKey = []database.GetUsageByAPIKeyRow{}
	}

	keysData := make([]map[string]interface{}, 0)
	for _, key := range usageByAPIKey {
		keysData = append(keysData, map[string]interface{}{
			"id":           key.ID,
			"name":         key.Name,
			"key":          maskAPIKey(key.Key),
			"requests_30d": key.RequestCount,
			"cost_30d":     float64(key.RequestCount) * 0.01,
		})
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"api_keys": keysData,
			"period": map[string]interface{}{
				"start": startDate,
				"end":   endDate,
			},
		},
	})
}

func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0.0
	}
	val, _ := n.Int64Value()
	return float64(val.Int64)
}
