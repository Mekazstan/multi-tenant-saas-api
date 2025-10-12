package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) upgradePlanHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Plan string `json:"plan"` // "starter" or "pro"
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

	// Get user
	user, err := cfg.db.GetUser(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve user details",
		})
		return
	}

	// Only owner can upgrade plan
	if user.Role != database.UserRoleOwner {
		respondWithError(w, http.StatusForbidden, ApiError{
			Code:    "PERMISSION_DENIED",
			Message: "Only organization owner can upgrade plan",
		})
		return
	}

	// Validate and convert plan
	var newPlan database.PlanType
	switch params.Plan {
	case "starter":
		newPlan = database.PlanTypeStarter
	case "pro":
		newPlan = database.PlanTypePro
	case "free":
		newPlan = database.PlanTypeFree
	default:
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_PLAN",
			Message: "Plan must be 'free', 'starter', or 'pro'",
		})
		return
	}

	// Get current organization
	org, err := cfg.db.GetOrganization(r.Context(), user.OrganizationID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve organization",
		})
		return
	}

	// Check if downgrading
	if newPlan == org.Plan {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "SAME_PLAN",
			Message: "Organization is already on this plan",
		})
		return
	}

	// Update organization plan
	updatedOrg, err := cfg.db.UpdateOrganizationPlan(r.Context(), database.UpdateOrganizationPlanParams{
		Plan: newPlan,
		ID:   user.OrganizationID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to update plan",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Message: "Plan upgraded successfully",
		Data: map[string]interface{}{
			"organization": map[string]interface{}{
				"id":         updatedOrg.ID,
				"name":       updatedOrg.Name,
				"plan":       updatedOrg.Plan,
				"updated_at": updatedOrg.UpdatedAt,
			},
		},
	})
}

func (cfg *apiConfig) initiatePaymentHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		BillingCycleID string `json:"billing_cycle_id"`
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

	// Parse cycle ID
	cycleID, err := uuid.Parse(params.BillingCycleID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_CYCLE_ID",
			Message: "Invalid billing cycle ID",
		})
		return
	}

	// Get user
	user, err := cfg.db.GetUser(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve user details",
		})
		return
	}

	// Get billing cycle
	cycle, err := cfg.db.GetBillingCycle(r.Context(), cycleID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, ApiError{
			Code:    "CYCLE_NOT_FOUND",
			Message: "Billing cycle not found",
		})
		return
	}

	// Verify cycle belongs to user's organization
	if cycle.OrganizationID != user.OrganizationID {
		respondWithError(w, http.StatusForbidden, ApiError{
			Code:    "PERMISSION_DENIED",
			Message: "You don't have permission to pay this invoice",
		})
		return
	}

	// Check if already paid
	if cycle.Status == database.BillingStatusPaid {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "ALREADY_PAID",
			Message: "This invoice has already been paid",
		})
		return
	}

	// Get organization
	org, err := cfg.db.GetOrganization(r.Context(), user.OrganizationID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve organization",
		})
		return
	}

	// In production, integrate with Stripe/Paystack here
	// For now, create a mock payment URL
	paymentURL := generatePaymentURL(cycle, org)

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"payment_url":     paymentURL,
			"amount":          cycle.TotalAmount,
			"billing_cycle_id": cycle.ID,
			"status":          cycle.Status,
		},
	})
}

func (cfg *apiConfig) paymentWebhookHandler(w http.ResponseWriter, r *http.Request) {
	// This is a simplified version. In production, verify webhook signature
	
	type WebhookPayload struct {
		Event          string `json:"event"`
		BillingCycleID string `json:"billing_cycle_id"`
		Amount         string `json:"amount"`
		Status         string `json:"status"`
		Reference      string `json:"reference"`
	}

	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_WEBHOOK",
			Message: "Invalid webhook payload",
		})
		return
	}

	// Verify webhook signature (implement based on payment provider)
	// if !verifyWebhookSignature(r) {
	//     respondWithError(w, http.StatusUnauthorized, ApiError{
	//         Code:    "INVALID_SIGNATURE",
	//         Message: "Invalid webhook signature",
	//     })
	//     return
	// }

	// Only process successful payments
	if payload.Event != "payment.success" || payload.Status != "success" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse billing cycle ID
	cycleID, err := uuid.Parse(payload.BillingCycleID)
	if err != nil {
		log.Printf("Invalid billing cycle ID in webhook: %s", payload.BillingCycleID)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get billing cycle
	cycle, err := cfg.db.GetBillingCycle(r.Context(), cycleID)
	if err != nil {
		log.Printf("Billing cycle not found: %s", cycleID)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Verify amount matches using Float64Value
	expectedFloat, err := cycle.TotalAmount.Float64Value()
	if err != nil {
		log.Printf("Failed to get expected amount for cycle %s: %v", cycleID, err)
		w.WriteHeader(http.StatusOK)
		return
	}

	if !expectedFloat.Valid {
		log.Printf("Invalid amount in billing cycle %s", cycleID)
		w.WriteHeader(http.StatusOK)
		return
	}

	paidAmount, err := strconv.ParseFloat(payload.Amount, 64)
	if err != nil {
		log.Printf("Invalid paid amount in webhook: %s", payload.Amount)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Allow small floating point differences (1 cent tolerance)
	if math.Abs(expectedFloat.Float64 - paidAmount) > 0.01 {
		log.Printf("Amount mismatch for cycle %s: expected %.2f, paid %.2f", 
			cycleID, expectedFloat.Float64, paidAmount)
		// Continue processing despite mismatch - you might want to handle this differently
	}

	// Update billing cycle status to PAID
	_, err = cfg.db.UpdateBillingCycleStatus(r.Context(), database.UpdateBillingCycleStatusParams{
		Status: database.BillingStatusPaid,
		ID:     cycleID,
	})
	if err != nil {
		log.Printf("Failed to update billing cycle status for %s: %v", cycleID, err)
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("Successfully processed payment for billing cycle %s, amount: %.2f", cycleID, paidAmount)

	// Send payment confirmation email
	// sendPaymentConfirmationEmail(cycle)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "received",
	})
}

func generatePaymentURL(cycle database.BillingCycle, org database.Organization) string {
	// In production, integrate with Stripe/Paystack
	// Example with Stripe:
	// session, _ := stripe.CheckoutSession.New(&stripe.CheckoutSessionParams{
	//     PaymentMethodTypes: ["card"],
	//     LineItems: [...],
	//     Mode: "payment",
	//     SuccessURL: "https://yourapp.com/billing/success",
	//     CancelURL: "https://yourapp.com/billing/cancel",
	// })
	// return session.URL
	
	return "https://payment-gateway.example.com/pay/" + cycle.ID.String()
}