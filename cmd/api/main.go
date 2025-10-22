package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/config"
	"github.com/Mekazstan/multi-tenant-saas-api/internal/database"
	"github.com/Mekazstan/multi-tenant-saas-api/internal/email"
	"github.com/Mekazstan/multi-tenant-saas-api/internal/payment"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type apiConfig struct {
	db          *database.Queries
	jwtSecret   string
	redisClient *redis.Client
	emailService *email.EmailService
	paymentService *payment.PaymentService
	config       *config.Config
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	log.Println("Connected to database successfully")

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Unable to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(opt)

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Unable to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis successfully")

	emailService, err := email.NewEmailService()
	if err != nil {
		log.Fatalf("Failed to initialize email service: %v", err)
	}
	log.Println("Email service initialized successfully")

	paymentService := payment.NewPaymentService(
		cfg.StripeSecretKey,
		cfg.StripeWebhookSecret,
		cfg.PaystackSecretKey,
		cfg.PaystackWebhookSecret,
	)
	log.Println("Payment service initialized successfully")

	dbQueries := database.New(pool)

	apiCfg := apiConfig{
		db:          dbQueries,
		jwtSecret:   cfg.JWTSecret,
		redisClient: redisClient,
		emailService: emailService,
		paymentService: paymentService,
		config:       cfg,
	}

	mux := http.NewServeMux()

	// ============================================
	// Public Auth Routes (No authentication)
	// ============================================
	mux.HandleFunc("POST /api/v1/auth/register", apiCfg.registerHandler)
	mux.HandleFunc("POST /api/v1/auth/login", apiCfg.loginHandler)
	mux.HandleFunc("POST /api/v1/auth/verify-email", apiCfg.verifyEmailHandler)
	mux.HandleFunc("POST /api/v1/auth/request-password-reset", apiCfg.requestPasswordResetHandler)
	mux.HandleFunc("POST /api/v1/auth/reset-password", apiCfg.resetPasswordHandler)

	// Team invitation acceptance (public - for new users)
	mux.HandleFunc("POST /api/v1/team/accept-invitation", apiCfg.acceptTeamInvitationHandler)
	mux.HandleFunc("POST /api/v1/team/decline-invitation", apiCfg.declineTeamInvitationHandler)

	// ============================================
	// Protected User Routes (JWT Authentication)
	// ============================================
	authMiddleware := AuthMiddleware(apiCfg.jwtSecret)
	
	// User profile
	mux.Handle("GET /api/v1/auth/me", authMiddleware(http.HandlerFunc(apiCfg.getCurrentUserHandler)))
	mux.Handle("POST /api/v1/auth/request-email-verification", authMiddleware(http.HandlerFunc(apiCfg.requestEmailVerificationHandler)))
	
	// API Keys
	mux.Handle("POST /api/v1/keys", authMiddleware(http.HandlerFunc(apiCfg.createAPIKeyHandler)))
	mux.Handle("GET /api/v1/keys", authMiddleware(http.HandlerFunc(apiCfg.listAPIKeysHandler)))
	mux.Handle("DELETE /api/v1/keys/{id}", authMiddleware(http.HandlerFunc(apiCfg.revokeAPIKeyHandler)))

	// Billing
	mux.Handle("GET /api/v1/billing/usage", authMiddleware(http.HandlerFunc(apiCfg.getBillingUsageHandler)))
	mux.Handle("GET /api/v1/billing/history", authMiddleware(http.HandlerFunc(apiCfg.getBillingHistoryHandler)))
	mux.Handle("GET /api/v1/billing/calculate", authMiddleware(http.HandlerFunc(apiCfg.calculateCurrentBillHandler)))
	mux.Handle("POST /api/v1/billing/upgrade", authMiddleware(http.HandlerFunc(apiCfg.upgradePlanHandler)))
	mux.Handle("POST /api/v1/billing/initiate-payment", authMiddleware(http.HandlerFunc(apiCfg.initiatePaymentHandler)))

	// Dashboard
	mux.Handle("GET /api/v1/dashboard/stats", authMiddleware(http.HandlerFunc(apiCfg.getDashboardStatsHandler)))
	mux.Handle("GET /api/v1/dashboard/usage-graph", authMiddleware(http.HandlerFunc(apiCfg.getDashboardUsageGraphHandler)))
	mux.Handle("GET /api/v1/dashboard/api-keys", authMiddleware(http.HandlerFunc(apiCfg.getDashboardAPIKeysHandler)))

	// Team Management
	mux.Handle("POST /api/v1/team/invite", authMiddleware(http.HandlerFunc(apiCfg.inviteTeamMemberHandler)))
	mux.Handle("GET /api/v1/team/members", authMiddleware(http.HandlerFunc(apiCfg.listOrganizationMembersHandler)))
	mux.Handle("DELETE /api/v1/team/members/{id}", authMiddleware(http.HandlerFunc(apiCfg.removeTeamMemberHandler)))
	mux.Handle("PUT /api/v1/team/members/{id}/role", authMiddleware(http.HandlerFunc(apiCfg.updateUserRoleHandler)))
	mux.Handle("DELETE /api/v1/team/invitations/{id}", authMiddleware(http.HandlerFunc(apiCfg.cancelInvitationHandler)))

	// ============================================
	// Webhook Routes (No auth - verified by signature)
	// ============================================
	mux.HandleFunc("POST /api/v1/webhooks/stripe", apiCfg.stripeWebhookHandler)
	mux.HandleFunc("POST /api/v1/webhooks/paystack", apiCfg.paystackWebhookHandler)

	// ============================================
	// API Key Protected Routes (with rate limiting)
	// ============================================
	apiKeyMiddleware := APIKeyMiddleware(apiCfg.db)
	rateLimitMiddleware := RateLimitMiddleware(apiCfg.redisClient, cfg.RateLimit)
	usageTrackingMiddleware := UsageTrackingMiddleware(apiCfg.db)

	messageHandler := usageTrackingMiddleware(
		rateLimitMiddleware(
			apiKeyMiddleware(http.HandlerFunc(apiCfg.sendMessageHandler)),
		),
	)
	mux.Handle("POST /api/v1/messages/send", messageHandler)

	messageStatusHandler := apiKeyMiddleware(http.HandlerFunc(apiCfg.getMessageStatusHandler))
	mux.Handle("GET /api/v1/messages/{id}", messageStatusHandler)

	// Apply global middleware
	handler := middlewareCors(mux)
	handler = LoggingMiddleware(handler)
	handler = SecurityHeadersMiddleware(handler)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
	}

	log.Printf("Server starting on port %s", cfg.Port)
	log.Printf("Environment: %s", cfg.Environment)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}