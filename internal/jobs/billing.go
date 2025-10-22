package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Mekazstan/multi-tenant-saas-api/internal/database"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

// GenerateMonthlyBillingCycles creates billing cycles for all organizations
// Runs on the 1st of every month at 00:00 UTC
func GenerateMonthlyBillingCycles(db *database.Queries) error {
	ctx := context.Background()

	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

	log.Printf("Generating billing cycles for period: %s to %s", periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02"))

	offset := int32(0)
	limit := int32(100)
	totalProcessed := 0

	for {
		orgs, err := db.ListOrganizations(ctx, database.ListOrganizationsParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return fmt.Errorf("failed to list organizations: %w", err)
		}

		if len(orgs) == 0 {
			break
		}

		for _, org := range orgs {
			if err := createBillingCycleForOrg(ctx, db, org, periodStart, periodEnd); err != nil {
				log.Printf("Error creating billing cycle for org %s: %v", org.ID, err)
				continue
			}
			totalProcessed++
		}

		offset += limit
	}

	log.Printf("Successfully generated %d billing cycles", totalProcessed)
	return nil
}

func createBillingCycleForOrg(ctx context.Context, db *database.Queries, org database.Organization, periodStart, periodEnd time.Time) error {
	startPeriodPg := pgtype.Timestamp{Time: periodStart, Valid: true}
	endPeriodPg := pgtype.Timestamp{Time: periodEnd, Valid: true}
	totalRequests, err := db.CountOrganizationUsage(ctx, database.CountOrganizationUsageParams{
		OrganizationID: org.ID,
		CreatedAt:      startPeriodPg,
		CreatedAt_2:    endPeriodPg,
	})
	if err != nil {
		return fmt.Errorf("failed to count usage: %w", err)
	}

	totalAmount := calculateBillingAmount(totalRequests, org.Plan)

	totalAmountPg, err := decimalToPgNumeric(totalAmount)
	if err != nil {
		return fmt.Errorf("failed to convert amount: %w", err)
	}

	cycle, err := db.CreateBillingCycle(ctx, database.CreateBillingCycleParams{
		OrganizationID: org.ID,
		PeriodStart:    startPeriodPg,
		PeriodEnd:      endPeriodPg,
		TotalRequests:  int32(totalRequests),
		TotalAmount:    totalAmountPg,
		Status:         database.BillingStatusPending,
	})
	if err != nil {
		return fmt.Errorf("failed to create billing cycle: %w", err)
	}

	log.Printf("Created billing cycle %s for org %s: %d requests, $%s", cycle.ID, org.Name, totalRequests, totalAmount)

	// Send invoice email (implement this based on your email service)
	// sendInvoiceEmail(org, cycle)

	return nil
}

func decimalToPgNumeric(d decimal.Decimal) (pgtype.Numeric, error) {
	num := pgtype.Numeric{}
	err := num.Scan(d.String())
	return num, err
}

func calculateBillingAmount(requests int64, plan database.PlanType) decimal.Decimal {
	var baseCost, perRequestCost decimal.Decimal

	switch plan {
	case database.PlanTypeFree:
		// Free plan: First 1000 requests are free, then $0.01 per request
		baseCost = decimal.NewFromInt(0)
		perRequestCost = decimal.NewFromFloat(0.01)

		if requests <= 1000 {
			return decimal.NewFromInt(0)
		}
		billableRequests := requests - 1000
		return decimal.NewFromInt(billableRequests).Mul(perRequestCost)

	case database.PlanTypeStarter:
		// Starter: $29 base + $0.01 per request
		baseCost = decimal.NewFromInt(29)
		perRequestCost = decimal.NewFromFloat(0.01)
		usageCost := decimal.NewFromInt(requests).Mul(perRequestCost)
		return baseCost.Add(usageCost)

	case database.PlanTypePro:
		// Pro: $99 base + $0.005 per request (50% discount)
		baseCost = decimal.NewFromInt(99)
		perRequestCost = decimal.NewFromFloat(0.005)
		usageCost := decimal.NewFromInt(requests).Mul(perRequestCost)
		return baseCost.Add(usageCost)
	}

	return decimal.NewFromInt(0)
}

// CheckAndMarkOverdueBillings marks pending invoices as overdue
// Run daily at 02:00 UTC
func CheckAndMarkOverdueBillings(db *database.Queries) error {
	ctx := context.Background()

	log.Println("Checking for overdue billing cycles...")

	pendingCycles, err := db.GetPendingBillingCycles(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending cycles: %w", err)
	}

	now := time.Now()
	overdueCount := 0

	for _, cycle := range pendingCycles {
		dueDate := cycle.PeriodEnd.Time.Add(7 * 24 * time.Hour)

		if now.After(dueDate) {
			_, err := db.UpdateBillingCycleStatus(ctx, database.UpdateBillingCycleStatusParams{
				Status: database.BillingStatusOverdue,
				ID:     cycle.ID,
			})
			if err != nil {
				log.Printf("Failed to mark cycle %s as overdue: %v", cycle.ID, err)
				continue
			}

			overdueCount++
			daysPastDue := int(now.Sub(dueDate).Hours() / 24)

			log.Printf("Marked billing cycle %s as OVERDUE (org: %s, days past due: %d)",
				cycle.ID, cycle.OrganizationName, daysPastDue)

			// Send overdue notification
			// sendOverdueNotification(cycle)

			// Optional: Suspend organization if payment is very late (30+ days)
			if daysPastDue > 30 {
				log.Printf("WARNING: Organization %s is %d days overdue - consider suspension",
					cycle.OrganizationName, daysPastDue)
				// suspendOrganization(db, cycle.OrganizationID)
			}
		}
	}

	log.Printf("Marked %d billing cycles as overdue", overdueCount)
	return nil
}

// AutoPayFreePlanInvoices automatically marks free plan $0 invoices as paid
// Runs shortly after monthly billing generation
func AutoPayFreePlanInvoices(db *database.Queries) error {
	ctx := context.Background()

	log.Println("Auto-paying free plan invoices...")

	pendingCycles, err := db.GetPendingBillingCycles(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending cycles: %w", err)
	}

	paidCount := 0
	for _, cycle := range pendingCycles {
		floatVal, err := cycle.TotalAmount.Float64Value()
		if err == nil && floatVal.Valid && floatVal.Float64 == 0 {
			_, err := db.UpdateBillingCycleStatus(ctx, database.UpdateBillingCycleStatusParams{
				Status: database.BillingStatusPaid,
				ID:     cycle.ID,
			})
			if err != nil {
				log.Printf("Failed to auto-pay cycle %s: %v", cycle.ID, err)
				continue
			}
			paidCount++
			log.Printf("Auto-paid $0 invoice for org: %s", cycle.OrganizationName)
		}
	}

	log.Printf("Auto-paid %d free invoices", paidCount)
	return nil
}
