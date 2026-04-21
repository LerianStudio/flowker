-- ============================================
-- Migration: 000001_audit_events (down)
-- Description: Drop audit_events table and all related objects.
-- ============================================

-- Drop triggers first
DROP TRIGGER IF EXISTS prevent_audit_event_truncate_trigger ON audit_events;
DROP TRIGGER IF EXISTS prevent_audit_event_update_trigger ON audit_events;
DROP TRIGGER IF EXISTS prevent_audit_event_delete_trigger ON audit_events;
DROP TRIGGER IF EXISTS audit_events_hash_chain ON audit_events;

-- Drop table
DROP TABLE IF EXISTS audit_events;

-- Drop functions
DROP FUNCTION IF EXISTS verify_audit_hash_chain(BIGINT, BIGINT);
DROP FUNCTION IF EXISTS calculate_audit_event_hash();
DROP FUNCTION IF EXISTS prevent_truncate();
DROP FUNCTION IF EXISTS prevent_audit_event_update();
DROP FUNCTION IF EXISTS prevent_audit_event_delete();

-- Drop enums in reverse order
DROP TYPE IF EXISTS actor_type_enum;
DROP TYPE IF EXISTS resource_type_enum;
DROP TYPE IF EXISTS audit_result_enum;
DROP TYPE IF EXISTS audit_action_enum;
DROP TYPE IF EXISTS audit_event_type_enum;
