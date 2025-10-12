package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/database"
	"github.com/Mekazstan/multi-tenant-saas-api/internal/jobs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// Get database URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	log.Println("Connected to database successfully")

	// Initialize database queries
	db := database.New(pool)

	// Create cron scheduler with seconds precision
	c := cron.New(cron.WithSeconds())

	// ============================================
	// Job 1: Monthly Billing Cycle Generation
	// Runs on the 1st of every month at 00:00:00 UTC
	// ============================================
	_, err = c.AddFunc("0 0 0 1 * *", func() {
		log.Println("========================================")
		log.Println("Starting monthly billing cycle generation...")
		log.Println("========================================")
		
		if err := jobs.GenerateMonthlyBillingCycles(db); err != nil {
			log.Printf("ERROR: Failed to generate billing cycles: %v", err)
			return
		}
		
		log.Println("Monthly billing cycle generation completed successfully")
		
		// Auto-pay free plan invoices ($0 invoices)
		log.Println("Auto-paying free plan invoices...")
		if err := jobs.AutoPayFreePlanInvoices(db); err != nil {
			log.Printf("ERROR: Failed to auto-pay free invoices: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to schedule monthly billing job: %v", err)
	}

	// ============================================
	// Job 2: Daily Overdue Billing Check
	// Runs every day at 02:00:00 UTC
	// ============================================
	_, err = c.AddFunc("0 0 2 * * *", func() {
		log.Println("========================================")
		log.Println("Starting overdue billing check...")
		log.Println("========================================")
		
		if err := jobs.CheckAndMarkOverdueBillings(db); err != nil {
			log.Printf("ERROR: Failed to check overdue billings: %v", err)
			return
		}
		
		log.Println("Overdue billing check completed successfully")
	})
	if err != nil {
		log.Fatalf("Failed to schedule overdue check job: %v", err)
	}

	// ============================================
	// Optional: Test Job (runs every minute)
	// Comment out in production
	// ============================================
	// _, err = c.AddFunc("0 * * * * *", func() {
	// 	log.Println("Test job running - current time:", time.Now())
	// })

	// Start the scheduler
	c.Start()
	log.Println("========================================")
	log.Println("Cron scheduler started successfully")
	log.Println("========================================")
	log.Println("Scheduled jobs:")
	log.Println("1. Monthly Billing Generation: 1st of every month at 00:00 UTC")
	log.Println("2. Overdue Check: Every day at 02:00 UTC")
	log.Println("========================================")

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-quit
	log.Println("Shutting down cron scheduler...")
	
	// Stop the scheduler
	ctx = c.Stop()
	<-ctx.Done()
	
	log.Println("Cron scheduler stopped successfully")
}