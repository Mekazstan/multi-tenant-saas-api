package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/database"
	"github.com/Mekazstan/multi-tenant-saas-api/internal/email"
	"github.com/Mekazstan/multi-tenant-saas-api/internal/payment"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
)

func (cfg *apiConfig) upgradePlanHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Plan string `json:"plan"`
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

	user, err := cfg.db.GetUser(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve user details",
		})
		return
	}

	if user.Role != database.UserRoleOwner {
		respondWithError(w, http.StatusForbidden, ApiError{
			Code:    "PERMISSION_DENIED",
			Message: "Only organization owner can upgrade plan",
		})
		return
	}

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

	org, err := cfg.db.GetOrganization(r.Context(), user.OrganizationID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve organization",
		})
		return
	}

	if newPlan == org.Plan {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "SAME_PLAN",
			Message: "Organization is already on this plan",
		})
		return
	}

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
		Provider       string `json:"provider"`
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

	if params.Provider == "" {
		params.Provider = "stripe"
	}

	cycleID, err := uuid.Parse(params.BillingCycleID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_CYCLE_ID",
			Message: "Invalid billing cycle ID",
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

	cycle, err := cfg.db.GetBillingCycle(r.Context(), cycleID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, ApiError{
			Code:    "CYCLE_NOT_FOUND",
			Message: "Billing cycle not found",
		})
		return
	}

	if cycle.OrganizationID != user.OrganizationID {
		respondWithError(w, http.StatusForbidden, ApiError{
			Code:    "PERMISSION_DENIED",
			Message: "You don't have permission to pay this invoice",
		})
		return
	}

	if cycle.Status == database.BillingStatusPaid {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "ALREADY_PAID",
			Message: "This invoice has already been paid",
		})
		return
	}

	org, err := cfg.db.GetOrganization(r.Context(), user.OrganizationID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Failed to retrieve organization",
		})
		return
	}

	amountFloat, err := cycle.TotalAmount.Float64Value()
	if err != nil || !amountFloat.Valid {
		respondWithError(w, http.StatusInternalServerError, ApiError{
			Code:    "INTERNAL_ERROR",
			Message: "Invalid amount",
		})
		return
	}

	var paymentURL string
	var providerResponse interface{}

	switch params.Provider {
	case "stripe":
		amountCents := int64(amountFloat.Float64 * 100)

		session, err := cfg.paymentService.Stripe.CreateCheckoutSession(payment.CheckoutSessionParams{
			OrganizationID: org.ID.String(),
			BillingCycleID: cycle.ID.String(),
			Amount:         amountCents,
			Currency:       "usd",
			SuccessURL:     cfg.config.AppURL + "/billing/success?session_id={CHECKOUT_SESSION_ID}",
			CancelURL:      cfg.config.AppURL + "/billing/cancel",
			CustomerEmail:  org.Email,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, ApiError{
				Code:    "PAYMENT_ERROR",
				Message: "Failed to create payment session",
				Details: err.Error(),
			})
			return
		}
		paymentURL = session.URL
		providerResponse = session

	case "paystack":
		amountKobo := int64(amountFloat.Float64 * 100)

		reference := fmt.Sprintf("INV_%s_%s", org.ID.String()[:8], cycle.ID.String()[:8])

		response, err := cfg.paymentService.Paystack.InitializeTransaction(payment.PaystackInitializeParams{
			Email:       org.Email,
			Amount:      amountKobo,
			Currency:    "NGN",
			Reference:   reference,
			CallbackURL: cfg.config.AppURL + "/billing/paystack/callback",
			Metadata: map[string]string{
				"organization_id":  org.ID.String(),
				"billing_cycle_id": cycle.ID.String(),
			},
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, ApiError{
				Code:    "PAYMENT_ERROR",
				Message: "Failed to create payment session",
				Details: err.Error(),
			})
			return
		}
		paymentURL = response.Data.AuthorizationURL
		providerResponse = response

	default:
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_PROVIDER",
			Message: "Payment provider must be 'stripe' or 'paystack'",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"payment_url":      paymentURL,
			"amount":           amountFloat.Float64,
			"billing_cycle_id": cycle.ID,
			"provider":         params.Provider,
			"status":           cycle.Status,
			"provider_data":    providerResponse,
		},
	})
}

func (cfg *apiConfig) stripeWebhookHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_PAYLOAD",
			Message: "Failed to read request body",
		})
		return
	}

	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		respondWithError(w, http.StatusUnauthorized, ApiError{
			Code:    "MISSING_SIGNATURE",
			Message: "Missing Stripe signature",
		})
		return
	}

	event, err := cfg.paymentService.Stripe.VerifyWebhookSignature(payload, signature)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, ApiError{
			Code:    "INVALID_SIGNATURE",
			Message: "Invalid webhook signature",
		})
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			log.Printf("Failed to parse checkout session: %v", err)
			w.WriteHeader(http.StatusOK)
			return
		}

		billingCycleID := session.Metadata["billing_cycle_id"]
		organizationID := session.Metadata["organization_id"]

		if billingCycleID == "" {
			log.Printf("Missing billing_cycle_id in metadata")
			w.WriteHeader(http.StatusOK)
			return
		}

		cycleID, err := uuid.Parse(billingCycleID)
		if err != nil {
			log.Printf("Invalid billing cycle ID: %s", billingCycleID)
			w.WriteHeader(http.StatusOK)
			return
		}

		cycle, err := cfg.db.GetBillingCycle(r.Context(), cycleID)
		if err != nil {
			log.Printf("Billing cycle not found: %s", cycleID)
			w.WriteHeader(http.StatusOK)
			return
		}

		_, err = cfg.db.UpdateBillingCycleStatus(r.Context(), database.UpdateBillingCycleStatusParams{
			Status: database.BillingStatusPaid,
			ID:     cycleID,
		})
		if err != nil {
			log.Printf("Failed to update billing cycle status: %v", err)
			w.WriteHeader(http.StatusOK)
			return
		}

		orgID, _ := uuid.Parse(organizationID)
		org, err := cfg.db.GetOrganization(r.Context(), orgID)
		if err != nil {
			log.Printf("Failed to get organization: %v", err)
		} else {
			go func() {
				amountFloat, _ := cycle.TotalAmount.Float64Value()
				cfg.emailService.SendPaymentSuccess(org.Email, email.PaymentSuccessData{
					OrganizationName: org.Name,
					Amount:           amountFloat.Float64,
					InvoiceNumber:    cycle.ID.String()[:8],
					ReceiptURL:       cfg.config.AppURL + "/billing/receipt/" + cycle.ID.String(),
				})
			}()
		}

		log.Printf("Successfully processed Stripe payment for cycle %s", cycleID)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

func (cfg *apiConfig) paystackWebhookHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ApiError{
			Code:    "INVALID_PAYLOAD",
			Message: "Failed to read request body",
		})
		return
	}

	signature := r.Header.Get("X-Paystack-Signature")
	if signature == "" {
		respondWithError(w, http.StatusUnauthorized, ApiError{
			Code:    "MISSING_SIGNATURE",
			Message: "Missing Paystack signature",
		})
		return
	}

	if !cfg.paymentService.Paystack.VerifyWebhookSignature(payload, signature) {
		respondWithError(w, http.StatusUnauthorized, ApiError{
			Code:    "INVALID_SIGNATURE",
			Message: "Invalid webhook signature",
		})
		return
	}

	event, err := cfg.paymentService.Paystack.ParseWebhookEvent(payload)
	if err != nil {
		log.Printf("Failed to parse Paystack webhook: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	switch event.Event {
	case "charge.success":
		metadata, ok := event.Data["metadata"].(map[string]interface{})
		if !ok {
			log.Printf("Missing metadata in Paystack webhook")
			w.WriteHeader(http.StatusOK)
			return
		}

		billingCycleID, ok := metadata["billing_cycle_id"].(string)
		if !ok || billingCycleID == "" {
			log.Printf("Missing billing_cycle_id in metadata")
			w.WriteHeader(http.StatusOK)
			return
		}

		organizationID, _ := metadata["organization_id"].(string)

		cycleID, err := uuid.Parse(billingCycleID)
		if err != nil {
			log.Printf("Invalid billing cycle ID: %s", billingCycleID)
			w.WriteHeader(http.StatusOK)
			return
		}

		cycle, err := cfg.db.GetBillingCycle(r.Context(), cycleID)
		if err != nil {
			log.Printf("Billing cycle not found: %s", cycleID)
			w.WriteHeader(http.StatusOK)
			return
		}

		_, err = cfg.db.UpdateBillingCycleStatus(r.Context(), database.UpdateBillingCycleStatusParams{
			Status: database.BillingStatusPaid,
			ID:     cycleID,
		})
		if err != nil {
			log.Printf("Failed to update billing cycle status: %v", err)
			w.WriteHeader(http.StatusOK)
			return
		}

		if organizationID != "" {
			orgID, _ := uuid.Parse(organizationID)
			org, err := cfg.db.GetOrganization(r.Context(), orgID)
			if err != nil {
				log.Printf("Failed to get organization: %v", err)
			} else {
				go func() {
					amountFloat, _ := cycle.TotalAmount.Float64Value()
					cfg.emailService.SendPaymentSuccess(org.Email, email.PaymentSuccessData{
						OrganizationName: org.Name,
						Amount:           amountFloat.Float64,
						InvoiceNumber:    cycle.ID.String()[:8],
						ReceiptURL:       cfg.config.AppURL + "/billing/receipt/" + cycle.ID.String(),
					})
				}()
			}
		}

		log.Printf("Successfully processed Paystack payment for cycle %s", cycleID)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}
