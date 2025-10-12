package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type apiConfig struct {
	db          *database.Queries
	jwtSecret   string
	redisClient *redis.Client
}

func main() {
	// Loading environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	rateLimitStr := os.Getenv("RATE_LIMIT_PER_MINUTE")
	rateLimit := 60
	if rateLimitStr != "" {
		if limit, err := strconv.Atoi(rateLimitStr); err == nil {
			rateLimit = limit
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	log.Println("Connected to database successfully")

	// Connect to Redis
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Unable to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(opt)

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Unable to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis successfully")

	// Initialize database queries
	dbQueries := database.New(pool)

	cfg := apiConfig{
		db:          dbQueries,
		jwtSecret:   jwtSecret,
		redisClient: redisClient,
	}

	// Setup router
	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("POST /api/v1/auth/register", cfg.registerHandler)
	mux.HandleFunc("POST /api/v1/auth/login", cfg.loginHandler)

	// Protected user routes (require JWT)
	authMiddleware := AuthMiddleware(cfg.jwtSecret)
	mux.Handle("GET /api/v1/auth/me", authMiddleware(http.HandlerFunc(cfg.getCurrentUserHandler)))
	mux.Handle("POST /api/v1/keys", authMiddleware(http.HandlerFunc(cfg.createAPIKeyHandler)))
	mux.Handle("GET /api/v1/keys", authMiddleware(http.HandlerFunc(cfg.listAPIKeysHandler)))
	mux.Handle("DELETE /api/v1/keys/{id}", authMiddleware(http.HandlerFunc(cfg.revokeAPIKeyHandler)))

	// Protected billing routes (require JWT)
	mux.Handle("GET /api/v1/billing/usage", authMiddleware(http.HandlerFunc(cfg.getBillingUsageHandler)))
	mux.Handle("GET /api/v1/billing/history", authMiddleware(http.HandlerFunc(cfg.getBillingHistoryHandler)))
	mux.Handle("GET /api/v1/billing/calculate", authMiddleware(http.HandlerFunc(cfg.calculateCurrentBillHandler)))
	mux.Handle("POST /api/v1/billing/upgrade", authMiddleware(http.HandlerFunc(cfg.upgradePlanHandler)))
	mux.Handle("POST /api/v1/billing/initiate-payment", authMiddleware(http.HandlerFunc(cfg.initiatePaymentHandler)))

	// Protected dashboard routes (require JWT)
	mux.Handle("GET /api/v1/dashboard/stats", authMiddleware(http.HandlerFunc(cfg.getDashboardStatsHandler)))
	mux.Handle("GET /api/v1/dashboard/usage-graph", authMiddleware(http.HandlerFunc(cfg.getDashboardUsageGraphHandler)))
	mux.Handle("GET /api/v1/dashboard/api-keys", authMiddleware(http.HandlerFunc(cfg.getDashboardAPIKeysHandler)))

	// Webhook routes (no auth - verified by signature)
	mux.HandleFunc("POST /api/v1/webhooks/payment", cfg.paymentWebhookHandler)

	// API Key protected routes (with rate limiting and usage tracking)
	apiKeyMiddleware := APIKeyMiddleware(cfg.db)
	rateLimitMiddleware := RateLimitMiddleware(cfg.redisClient, rateLimit)
	usageTrackingMiddleware := UsageTrackingMiddleware(cfg.db)

	messageHandler := usageTrackingMiddleware(
		rateLimitMiddleware(
			apiKeyMiddleware(http.HandlerFunc(cfg.sendMessageHandler)),
		),
	)
	mux.Handle("POST /api/v1/messages/send", messageHandler)

	messageStatusHandler := apiKeyMiddleware(http.HandlerFunc(cfg.getMessageStatusHandler))
	mux.Handle("GET /api/v1/messages/{id}", messageStatusHandler)

	handler := middlewareCors(mux)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	log.Printf("Server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
