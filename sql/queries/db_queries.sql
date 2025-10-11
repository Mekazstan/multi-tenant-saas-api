-- name: CreateOrganization :one
INSERT INTO organizations (name, email, plan)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetOrganization :one
SELECT * FROM organizations
WHERE id = $1;

-- name: GetOrganizationByEmail :one
SELECT * FROM organizations
WHERE email = $1;

-- name: UpdateOrganizationPlan :one
UPDATE organizations
SET plan = $1, updated_at = NOW()
WHERE id = $2
RETURNING *;

-- name: ListOrganizations :many
SELECT * FROM organizations
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: DeleteOrganization :exec
DELETE FROM organizations
WHERE id = $1;

-- ============================================
-- USER QUERIES
-- ============================================

-- name: CreateUser :one
INSERT INTO users (organization_id, email, password_hash, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: GetUserWithOrganization :one
SELECT 
    u.*,
    o.name as organization_name,
    o.plan as organization_plan
FROM users u
JOIN organizations o ON u.organization_id = o.id
WHERE u.id = $1;

-- name: ListOrganizationUsers :many
SELECT * FROM users
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: UpdateUserRole :one
UPDATE users
SET role = $1
WHERE id = $2
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- ============================================
-- API KEY QUERIES
-- ============================================

-- name: CreateAPIKey :one
INSERT INTO api_keys (organization_id, key, name, is_active)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAPIKey :one
SELECT * FROM api_keys
WHERE id = $1;

-- name: GetAPIKeyByKey :one
SELECT 
    ak.*,
    o.id as org_id,
    o.name as org_name,
    o.plan as org_plan
FROM api_keys ak
JOIN organizations o ON ak.organization_id = o.id
WHERE ak.key = $1 AND ak.is_active = true;

-- name: ListOrganizationAPIKeys :many
SELECT * FROM api_keys
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys
SET last_used_at = NOW()
WHERE id = $1;

-- name: DeactivateAPIKey :one
UPDATE api_keys
SET is_active = false
WHERE id = $1
RETURNING *;

-- name: ActivateAPIKey :one
UPDATE api_keys
SET is_active = true
WHERE id = $1
RETURNING *;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys
WHERE id = $1;

-- ============================================
-- USAGE RECORD QUERIES
-- ============================================

-- name: CreateUsageRecord :one
INSERT INTO usage_records (organization_id, api_key_id, endpoint, method, status_code)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUsageRecord :one
SELECT * FROM usage_records
WHERE id = $1;

-- name: ListOrganizationUsage :many
SELECT * FROM usage_records
WHERE organization_id = $1
    AND created_at >= $2
    AND created_at <= $3
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: CountOrganizationUsage :one
SELECT COUNT(*) FROM usage_records
WHERE organization_id = $1
    AND created_at >= $2
    AND created_at <= $3;

-- name: GetUsageByEndpoint :many
SELECT 
    endpoint,
    COUNT(*) as request_count,
    COUNT(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 END) as success_count,
    COUNT(CASE WHEN status_code >= 400 THEN 1 END) as error_count
FROM usage_records
WHERE organization_id = $1
    AND created_at >= $2
    AND created_at <= $3
GROUP BY endpoint
ORDER BY request_count DESC;

-- name: GetUsageByAPIKey :many
SELECT 
    ak.id,
    ak.name,
    ak.key,
    COUNT(ur.id) as request_count
FROM api_keys ak
LEFT JOIN usage_records ur ON ak.id = ur.api_key_id
    AND ur.created_at >= $2
    AND ur.created_at <= $3
WHERE ak.organization_id = $1
GROUP BY ak.id, ak.name, ak.key
ORDER BY request_count DESC;

-- name: GetDailyUsageStats :many
SELECT 
    DATE(created_at) as date,
    COUNT(*) as request_count,
    COUNT(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 END) as success_count,
    COUNT(CASE WHEN status_code >= 400 THEN 1 END) as error_count
FROM usage_records
WHERE organization_id = $1
    AND created_at >= $2
    AND created_at <= $3
GROUP BY DATE(created_at)
ORDER BY date DESC;

-- ============================================
-- BILLING CYCLE QUERIES
-- ============================================

-- name: CreateBillingCycle :one
INSERT INTO billing_cycles (
    organization_id,
    period_start,
    period_end,
    total_requests,
    total_amount,
    status
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetBillingCycle :one
SELECT * FROM billing_cycles
WHERE id = $1;

-- name: GetCurrentBillingCycle :one
SELECT * FROM billing_cycles
WHERE organization_id = $1
    AND period_start <= NOW()
    AND period_end >= NOW()
ORDER BY period_start DESC
LIMIT 1;

-- name: ListOrganizationBillingCycles :many
SELECT * FROM billing_cycles
WHERE organization_id = $1
ORDER BY period_start DESC
LIMIT $2 OFFSET $3;

-- name: UpdateBillingCycleStatus :one
UPDATE billing_cycles
SET status = $1
WHERE id = $2
RETURNING *;

-- name: UpdateBillingCycleTotals :one
UPDATE billing_cycles
SET 
    total_requests = $1,
    total_amount = $2
WHERE id = $3
RETURNING *;

-- name: GetPendingBillingCycles :many
SELECT 
    bc.*,
    o.name as organization_name,
    o.email as organization_email
FROM billing_cycles bc
JOIN organizations o ON bc.organization_id = o.id
WHERE bc.status = 'pending'
    AND bc.period_end < NOW()
ORDER BY bc.period_end ASC;

-- name: GetOverdueBillingCycles :many
SELECT 
    bc.*,
    o.name as organization_name,
    o.email as organization_email
FROM billing_cycles bc
JOIN organizations o ON bc.organization_id = o.id
WHERE bc.status = 'overdue'
ORDER BY bc.period_end ASC;