-- Fix #2: Scope idempotency_key uniqueness to per-user instead of global.
-- Fix #3: Add client_event_id unique constraint for idempotency.
-- Fix #4: Prevent duplicate weekly plans via DB constraint.

-- Drop the global idempotency_key unique index and replace with per-user scope.
DROP INDEX IF EXISTS idx_review_logs_idempotency;
CREATE UNIQUE INDEX IF NOT EXISTS idx_review_logs_idempotency
    ON review_logs (user_id, idempotency_key) WHERE idempotency_key IS NOT NULL;

-- Add per-user unique index on client_event_id for spec-compliant idempotency.
CREATE UNIQUE INDEX IF NOT EXISTS idx_review_logs_client_event
    ON review_logs (user_id, client_event_id) WHERE client_event_id IS NOT NULL;

-- Add unique constraint on plans to prevent duplicate weekly plans per user.
CREATE UNIQUE INDEX IF NOT EXISTS idx_plans_user_week
    ON plans (user_id, week_start);
