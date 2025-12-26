-- Migration: 003_add_audit_logs
-- Description: Add audit logging tables for compliance and debugging
-- Created: 2025-12-26

-- ============================================================================
-- AUDIT LOGS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id VARCHAR(36) PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'info',
    user_id VARCHAR(36) REFERENCES users(id) ON DELETE SET NULL,
    user_email VARCHAR(255),
    user_ip VARCHAR(45),
    user_agent TEXT,
    request_id VARCHAR(36),
    resource_type VARCHAR(50),
    resource_id VARCHAR(36),
    action VARCHAR(50),
    description TEXT,
    old_value TEXT, -- JSON
    new_value TEXT, -- JSON
    metadata TEXT, -- JSON
    success BOOLEAN NOT NULL DEFAULT TRUE,
    error_message TEXT
);

-- Indexes for common query patterns
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX idx_audit_logs_severity ON audit_logs(severity);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX idx_audit_logs_resource_id ON audit_logs(resource_id);
CREATE INDEX idx_audit_logs_success ON audit_logs(success);

-- Composite indexes for complex queries
CREATE INDEX idx_audit_logs_user_time ON audit_logs(user_id, timestamp);
CREATE INDEX idx_audit_logs_resource_time ON audit_logs(resource_type, resource_id, timestamp);
CREATE INDEX idx_audit_logs_type_time ON audit_logs(event_type, timestamp);

-- ============================================================================
-- AUDIT LOG PARTITIONING (Optional - for high-volume environments)
-- ============================================================================
-- Uncomment for PostgreSQL partitioning by month:
-- CREATE TABLE audit_logs_2025_01 PARTITION OF audit_logs
--     FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- ============================================================================
-- SECURITY EVENTS TABLE (for security-specific tracking)
-- ============================================================================
CREATE TABLE IF NOT EXISTS security_events (
    id VARCHAR(36) PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_type VARCHAR(50) NOT NULL,
    source_ip VARCHAR(45),
    user_id VARCHAR(36) REFERENCES users(id) ON DELETE SET NULL,
    user_email VARCHAR(255),
    endpoint VARCHAR(255),
    method VARCHAR(10),
    status_code INT,
    blocked BOOLEAN NOT NULL DEFAULT FALSE,
    block_reason VARCHAR(255),
    metadata TEXT -- JSON
);

CREATE INDEX idx_security_events_timestamp ON security_events(timestamp);
CREATE INDEX idx_security_events_event_type ON security_events(event_type);
CREATE INDEX idx_security_events_source_ip ON security_events(source_ip);
CREATE INDEX idx_security_events_blocked ON security_events(blocked);

-- ============================================================================
-- LOGIN ATTEMPTS TABLE (for brute force protection)
-- ============================================================================
CREATE TABLE IF NOT EXISTS login_attempts (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    source_ip VARCHAR(45) NOT NULL,
    success BOOLEAN NOT NULL,
    failure_reason VARCHAR(100),
    user_agent TEXT,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_login_attempts_email ON login_attempts(email);
CREATE INDEX idx_login_attempts_source_ip ON login_attempts(source_ip);
CREATE INDEX idx_login_attempts_timestamp ON login_attempts(timestamp);
CREATE INDEX idx_login_attempts_email_ip_time ON login_attempts(email, source_ip, timestamp);

-- ============================================================================
-- RATE LIMIT TRACKING TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS rate_limit_events (
    id VARCHAR(36) PRIMARY KEY,
    source_ip VARCHAR(45) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    limit_type VARCHAR(50) NOT NULL, -- ip, endpoint, global
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_rate_limit_events_source_ip ON rate_limit_events(source_ip);
CREATE INDEX idx_rate_limit_events_timestamp ON rate_limit_events(timestamp);

-- ============================================================================
-- AUDIT LOG RETENTION FUNCTION
-- ============================================================================
-- Function to delete old audit logs (call via cron or scheduled job)
CREATE OR REPLACE FUNCTION cleanup_old_audit_logs(retention_days INT DEFAULT 90)
RETURNS INT AS $$
DECLARE
    deleted_count INT;
BEGIN
    DELETE FROM audit_logs
    WHERE timestamp < NOW() - (retention_days || ' days')::INTERVAL;
    GET DIAGNOSTICS deleted_count = ROW_COUNT;

    DELETE FROM security_events
    WHERE timestamp < NOW() - (retention_days || ' days')::INTERVAL;

    DELETE FROM login_attempts
    WHERE timestamp < NOW() - (retention_days || ' days')::INTERVAL;

    DELETE FROM rate_limit_events
    WHERE timestamp < NOW() - (retention_days / 7 || ' days')::INTERVAL;

    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- AUDIT SUMMARY VIEW
-- ============================================================================
CREATE VIEW audit_summary AS
SELECT
    DATE(timestamp) as date,
    event_type,
    COUNT(*) as event_count,
    SUM(CASE WHEN success THEN 1 ELSE 0 END) as success_count,
    SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failure_count
FROM audit_logs
WHERE timestamp > NOW() - INTERVAL '30 days'
GROUP BY DATE(timestamp), event_type
ORDER BY date DESC, event_count DESC;

-- ============================================================================
-- SECURITY SUMMARY VIEW
-- ============================================================================
CREATE VIEW security_summary AS
SELECT
    DATE(timestamp) as date,
    event_type,
    COUNT(*) as total_events,
    COUNT(DISTINCT source_ip) as unique_ips,
    SUM(CASE WHEN blocked THEN 1 ELSE 0 END) as blocked_count
FROM security_events
WHERE timestamp > NOW() - INTERVAL '7 days'
GROUP BY DATE(timestamp), event_type
ORDER BY date DESC;

-- ============================================================================
-- USER ACTIVITY VIEW
-- ============================================================================
CREATE VIEW user_activity AS
SELECT
    user_id,
    user_email,
    COUNT(*) as action_count,
    MAX(timestamp) as last_activity,
    array_agg(DISTINCT event_type) as event_types
FROM audit_logs
WHERE user_id IS NOT NULL
  AND timestamp > NOW() - INTERVAL '24 hours'
GROUP BY user_id, user_email;

-- ============================================================================
-- DOWN MIGRATION
-- ============================================================================
-- To rollback this migration, execute:
-- DROP VIEW IF EXISTS user_activity;
-- DROP VIEW IF EXISTS security_summary;
-- DROP VIEW IF EXISTS audit_summary;
-- DROP FUNCTION IF EXISTS cleanup_old_audit_logs;
-- DROP TABLE IF EXISTS rate_limit_events CASCADE;
-- DROP TABLE IF EXISTS login_attempts CASCADE;
-- DROP TABLE IF EXISTS security_events CASCADE;
-- DROP TABLE IF EXISTS audit_logs CASCADE;
