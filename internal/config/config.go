package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Environment string

const (
	Development Environment = "development"
	Staging     Environment = "staging"
	Production  Environment = "production"
)

type Config struct {
	Environment Environment
	Port        string
	AppURL      string
	DatabaseURL string
	RedisURL string
	JWTSecret string
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
	RateLimit int
	StripeSecretKey      string
	StripeWebhookSecret  string
	PaystackSecretKey    string
	PaystackWebhookSecret string
	EnableEmailVerification bool
	EnableTeamInvitations   bool
}

func Load() (*Config, error) {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	if env != "production" {
		if err := godotenv.Load(); err != nil {
			if err := godotenv.Load("../../.env"); err != nil {
				if env == "development" {
					return nil, fmt.Errorf("failed to load .env file: %w", err)
				}
			}
		}
	}

	cfg := &Config{
		Environment: Environment(env),
		Port:        getEnv("PORT", "8080"),
		AppURL:      getEnv("APP_URL", "http://localhost:3000"),
		
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		
		JWTSecret: getEnv("JWT_SECRET", ""),
		
		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		FromEmail:    getEnv("FROM_EMAIL", "noreply@mts.com"),
		FromName:     getEnv("FROM_NAME", "MTS"),
		
		RateLimit: getEnvAsInt("RATE_LIMIT_PER_MINUTE", 60),
		
		StripeSecretKey:       getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret:   getEnv("STRIPE_WEBHOOK_SECRET", ""),
		PaystackSecretKey:     getEnv("PAYSTACK_SECRET_KEY", ""),
		PaystackWebhookSecret: getEnv("PAYSTACK_WEBHOOK_SECRET", ""),
		
		EnableEmailVerification: getEnvAsBool("ENABLE_EMAIL_VERIFICATION", true),
		EnableTeamInvitations:   getEnvAsBool("ENABLE_TEAM_INVITATIONS", true),
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if c.SMTPHost != "" || c.SMTPUsername != "" || c.SMTPPassword != "" {
		if c.SMTPHost == "" || c.SMTPUsername == "" || c.SMTPPassword == "" {
			return fmt.Errorf("incomplete SMTP configuration: all SMTP fields must be set")
		}
	}

	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == Development
}

func (c *Config) IsStaging() bool {
	return c.Environment == Staging
}

func (c *Config) IsProduction() bool {
	return c.Environment == Production
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	
	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	
	return value
}