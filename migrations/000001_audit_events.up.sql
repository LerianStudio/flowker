-- ============================================
-- Migration: 000001_audit_events
-- Description: Create audit_events table with hash chain, immutability rules,
--              and Flowker-specific enums for workflow orchestration audit trail.
-- Date: 2026-03-20
-- ============================================

-- Enable pgcrypto extension for SHA-256 hashing in audit functions
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================
-- Function: calculate_audit_event_hash
-- Trigger function to calculate hash chain for audit events.
-- Each record's hash includes the previous record's hash for tamper detection.
--
-- Hash includes (in canonical order with pipe delimiter):
--   1. previous_hash (or "GENESIS")
--   2. event_id
--   3. event_type
--   4. action
--   5. result
--   6. resource_id
--   7. resource_type
--   8. actor_type
--   9. actor_id
--  10. actor_ip
--  11. context (JSONB)
--  12. metadata (JSONB)
--  13. created_at
-- ============================================

CREATE OR REPLACE FUNCTION calculate_audit_event_hash()
RETURNS TRIGGER AS $$
DECLARE
    prev_hash VARCHAR(64);
    hash_input TEXT;
BEGIN
    -- Acquire advisory lock to serialize access to the last audit event hash.
    -- This prevents concurrent inserts from reading the same previous_hash.
    -- Lock is held for the transaction and released automatically at commit/rollback.
    -- Advisory lock using pi digits (314159265) to serialize hash chain inserts
    PERFORM pg_advisory_xact_lock(314159265);

    -- Get the hash of the previous record (if any)
    SELECT hash INTO prev_hash
    FROM audit_events
    ORDER BY id DESC
    LIMIT 1;

    NEW.previous_hash := prev_hash;

    -- Build hash input with pipe delimiter (canonical field order).
    -- All immutable fields are included to prevent undetected tampering.
    hash_input := COALESCE(prev_hash, 'GENESIS')
        || '|' || NEW.event_id::text
        || '|' || NEW.event_type::text
        || '|' || NEW.action::text
        || '|' || NEW.result::text
        || '|' || NEW.resource_id
        || '|' || NEW.resource_type::text
        || '|' || NEW.actor_type::text
        || '|' || NEW.actor_id
        || '|' || NEW.actor_ip
        || '|' || NEW.context::text
        || '|' || NEW.metadata::text
        || '|' || to_char(NEW.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');

    -- Calculate SHA-256 hash
    NEW.hash := encode(sha256(hash_input::bytea), 'hex');

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- Function: verify_audit_hash_chain
-- Verifies the integrity of the hash chain.
-- Returns true if chain is valid, false if tampered.
--
-- IMPORTANT: Must use the same field order and delimiter as calculate_audit_event_hash()
-- Order: previous_hash | event_id | event_type | action | result | resource_id |
--        resource_type | actor_type | actor_id | actor_ip | context | metadata | created_at
-- ============================================

DROP FUNCTION IF EXISTS verify_audit_hash_chain(BIGINT, BIGINT);

CREATE FUNCTION verify_audit_hash_chain(
    start_id BIGINT DEFAULT 1,
    end_id BIGINT DEFAULT NULL
)
RETURNS TABLE (
    is_valid BOOLEAN,
    first_invalid_id BIGINT,
    total_checked BIGINT,
    error_detail TEXT
) AS $$
DECLARE
    rec RECORD;
    prev_hash VARCHAR(64);
    expected_hash VARCHAR(64);
    hash_input TEXT;
    checked_count BIGINT := 0;
    invalid_id BIGINT := NULL;
    chain_valid BOOLEAN := TRUE;
    err_detail TEXT := NULL;
BEGIN
    -- Seed prev_hash from the row before start_id (or GENESIS if start_id is first)
    SELECT hash INTO prev_hash FROM audit_events WHERE id < start_id ORDER BY id DESC LIMIT 1;
    IF prev_hash IS NULL THEN
        prev_hash := 'GENESIS';
    END IF;

    FOR rec IN
        SELECT * FROM audit_events
        WHERE id >= start_id
        AND (end_id IS NULL OR id <= end_id)
        ORDER BY id ASC
    LOOP
        checked_count := checked_count + 1;

        -- Calculate expected hash (MUST match calculate_audit_event_hash exactly).
        hash_input := prev_hash
            || '|' || rec.event_id::text
            || '|' || rec.event_type::text
            || '|' || rec.action::text
            || '|' || rec.result::text
            || '|' || rec.resource_id
            || '|' || rec.resource_type::text
            || '|' || rec.actor_type::text
            || '|' || rec.actor_id
            || '|' || rec.actor_ip
            || '|' || rec.context::text
            || '|' || rec.metadata::text
            || '|' || to_char(rec.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
        expected_hash := encode(sha256(hash_input::bytea), 'hex');

        -- Check if stored hash matches expected
        IF rec.hash != expected_hash THEN
            chain_valid := FALSE;
            invalid_id := rec.id;
            err_detail := 'Hash mismatch: expected ' || expected_hash || ', got ' || rec.hash;
            EXIT;
        END IF;

        -- Check if previous_hash chain is intact
        IF COALESCE(rec.previous_hash, 'GENESIS') != prev_hash THEN
            chain_valid := FALSE;
            invalid_id := rec.id;
            err_detail := 'Chain break: expected previous_hash ' || prev_hash || ', got ' || COALESCE(rec.previous_hash, 'NULL');
            EXIT;
        END IF;

        prev_hash := rec.hash;
    END LOOP;

    RETURN QUERY SELECT chain_valid, invalid_id, checked_count, err_detail;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- Enum types
-- ============================================

CREATE TYPE audit_event_type_enum AS ENUM (
    'WORKFLOW_CREATED', 'WORKFLOW_UPDATED', 'WORKFLOW_ACTIVATED',
    'WORKFLOW_DEACTIVATED', 'WORKFLOW_DRAFTED', 'WORKFLOW_DELETED',
    'EXECUTION_STARTED', 'EXECUTION_COMPLETED', 'EXECUTION_FAILED',
    'PROVIDER_CALL_STARTED', 'PROVIDER_CALL_COMPLETED', 'PROVIDER_CALL_FAILED',
    'PROVIDER_CONFIG_CREATED', 'PROVIDER_CONFIG_UPDATED', 'PROVIDER_CONFIG_DELETED'
);

CREATE TYPE audit_action_enum AS ENUM (
    'CREATE', 'UPDATE', 'DELETE', 'ACTIVATE', 'DEACTIVATE', 'DRAFT', 'EXECUTE'
);

CREATE TYPE audit_result_enum AS ENUM ('SUCCESS', 'FAILED');

CREATE TYPE resource_type_enum AS ENUM ('workflow', 'execution', 'provider_config');

CREATE TYPE actor_type_enum AS ENUM ('user', 'system', 'api_key');

-- ============================================
-- Audit events table (append-only, immutable)
-- ============================================

CREATE TABLE IF NOT EXISTS audit_events (
    -- Internal fields (system-managed, not exposed in API)
    id              BIGSERIAL PRIMARY KEY,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    hash            VARCHAR(64)     NOT NULL,
    previous_hash   VARCHAR(64),

    -- Core indexed fields
    event_id        UUID            NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    event_type      audit_event_type_enum NOT NULL,
    action          audit_action_enum     NOT NULL,

    -- Result field
    result          audit_result_enum     NOT NULL,

    -- Resource fields (indexed)
    resource_id     VARCHAR(255)    NOT NULL,
    resource_type   resource_type_enum    NOT NULL,

    -- Actor fields
    actor_type      actor_type_enum NOT NULL,
    actor_id        VARCHAR(255)    NOT NULL,
    actor_ip        VARCHAR(45)     NOT NULL DEFAULT '0.0.0.0',

    -- Context field (JSONB)
    -- For CRUD: { before: {...}, after: {...} }
    -- For executions: { input: {...}, output: {...} }
    context         JSONB           NOT NULL DEFAULT '{}',

    -- Metadata field (JSONB) - additional info like correlationId, traceId
    metadata        JSONB           NOT NULL DEFAULT '{}'
);

-- ============================================
-- Indexes
-- ============================================

-- Primary query indexes
CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON audit_events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_created_at ON audit_events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_action ON audit_events(action);
CREATE INDEX IF NOT EXISTS idx_audit_events_result ON audit_events(result);

-- Resource lookup indexes
CREATE INDEX IF NOT EXISTS idx_audit_events_resource ON audit_events(resource_type, resource_id);

-- Actor lookup indexes
CREATE INDEX IF NOT EXISTS idx_audit_events_actor ON audit_events(actor_type, actor_id);

-- Hash chain verification index
CREATE INDEX IF NOT EXISTS idx_audit_events_hash ON audit_events(hash);

-- JSONB metadata index for correlationId and traceId lookups
CREATE INDEX IF NOT EXISTS idx_audit_events_metadata ON audit_events USING GIN (metadata jsonb_path_ops);

-- ============================================
-- Immutability rules (SOX compliance)
-- ============================================

-- Prevent UPDATE — fail explicitly for SOX compliance
CREATE OR REPLACE FUNCTION prevent_audit_event_update() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'UPDATE is not allowed on audit_events table (SOX compliance)';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS prevent_audit_event_update_trigger ON audit_events;
CREATE TRIGGER prevent_audit_event_update_trigger
    BEFORE UPDATE ON audit_events
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_event_update();

-- Prevent DELETE — fail explicitly for SOX compliance
CREATE OR REPLACE FUNCTION prevent_audit_event_delete() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'DELETE is not allowed on audit_events table (SOX compliance)';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS prevent_audit_event_delete_trigger ON audit_events;
CREATE TRIGGER prevent_audit_event_delete_trigger
    BEFORE DELETE ON audit_events
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_event_delete();

-- TRUNCATE protection function
CREATE OR REPLACE FUNCTION prevent_truncate()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'TRUNCATE is not allowed on this table';
END;
$$ LANGUAGE plpgsql;

-- TRUNCATE protection trigger
CREATE TRIGGER prevent_audit_event_truncate_trigger
    BEFORE TRUNCATE ON audit_events
    FOR EACH STATEMENT
    EXECUTE FUNCTION prevent_truncate();

-- ============================================
-- Hash chain trigger
-- ============================================

-- Apply hash chain trigger before insert
-- Uses calculate_audit_event_hash() function defined above
CREATE TRIGGER audit_events_hash_chain
    BEFORE INSERT ON audit_events
    FOR EACH ROW
    EXECUTE FUNCTION calculate_audit_event_hash();
