-- Migration: 001_initial_schema
-- Description: Initial database schema for n8n-go
-- Created: 2025-12-26

-- ============================================================================
-- USERS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_is_active ON users(is_active);

-- ============================================================================
-- API KEYS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(10) NOT NULL,
    scopes TEXT, -- JSON array of scopes
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_key_prefix ON api_keys(key_prefix);

-- ============================================================================
-- WORKFLOWS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflows (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    nodes TEXT NOT NULL, -- JSON array of nodes
    connections TEXT NOT NULL, -- JSON object of connections
    settings TEXT, -- JSON object of settings
    static_data TEXT, -- JSON object of static data
    pin_data TEXT, -- JSON object of pinned data
    version_id VARCHAR(36),
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_by VARCHAR(36) REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_workflows_name ON workflows(name);
CREATE INDEX idx_workflows_active ON workflows(active);
CREATE INDEX idx_workflows_is_archived ON workflows(is_archived);
CREATE INDEX idx_workflows_created_by ON workflows(created_by);
CREATE INDEX idx_workflows_updated_at ON workflows(updated_at);

-- ============================================================================
-- WORKFLOW VERSIONS TABLE (for versioning)
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_versions (
    id VARCHAR(36) PRIMARY KEY,
    workflow_id VARCHAR(36) NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    version_number INT NOT NULL,
    nodes TEXT NOT NULL,
    connections TEXT NOT NULL,
    settings TEXT,
    change_description TEXT,
    created_by VARCHAR(36) REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workflow_id, version_number)
);

CREATE INDEX idx_workflow_versions_workflow_id ON workflow_versions(workflow_id);
CREATE INDEX idx_workflow_versions_version_number ON workflow_versions(version_number);

-- ============================================================================
-- EXECUTIONS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS executions (
    id VARCHAR(36) PRIMARY KEY,
    workflow_id VARCHAR(36) NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL, -- running, completed, failed, cancelled
    mode VARCHAR(50) NOT NULL, -- manual, trigger, webhook, retry
    started_at TIMESTAMP NOT NULL,
    finished_at TIMESTAMP,
    data TEXT, -- JSON execution data
    error TEXT,
    retry_of VARCHAR(36),
    retry_success_id VARCHAR(36),
    wait_till TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_executions_workflow_id ON executions(workflow_id);
CREATE INDEX idx_executions_status ON executions(status);
CREATE INDEX idx_executions_mode ON executions(mode);
CREATE INDEX idx_executions_started_at ON executions(started_at);
CREATE INDEX idx_executions_finished_at ON executions(finished_at);

-- ============================================================================
-- CREDENTIALS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS credentials (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    data TEXT NOT NULL, -- Encrypted JSON credential data
    no_data_saved BOOLEAN NOT NULL DEFAULT FALSE,
    created_by VARCHAR(36) REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_credentials_name ON credentials(name);
CREATE INDEX idx_credentials_type ON credentials(type);
CREATE INDEX idx_credentials_created_by ON credentials(created_by);

-- ============================================================================
-- WEBHOOKS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS webhooks (
    id VARCHAR(36) PRIMARY KEY,
    workflow_id VARCHAR(36) NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    node_name VARCHAR(255) NOT NULL,
    path VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    is_test BOOLEAN NOT NULL DEFAULT FALSE,
    path_length INT NOT NULL DEFAULT 1,
    webhook_id VARCHAR(36),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_webhooks_workflow_id ON webhooks(workflow_id);
CREATE INDEX idx_webhooks_path ON webhooks(path);
CREATE INDEX idx_webhooks_method ON webhooks(method);
CREATE UNIQUE INDEX idx_webhooks_unique_path ON webhooks(path, method, is_test);

-- ============================================================================
-- SETTINGS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    load_on_startup BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- VARIABLES TABLE (for environment variables)
-- ============================================================================
CREATE TABLE IF NOT EXISTS variables (
    id VARCHAR(36) PRIMARY KEY,
    key VARCHAR(255) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'string',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_variables_key ON variables(key);

-- ============================================================================
-- WORKFLOW STATISTICS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_statistics (
    id VARCHAR(36) PRIMARY KEY,
    workflow_id VARCHAR(36) NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    count INT NOT NULL DEFAULT 0,
    latest_event TIMESTAMP,
    UNIQUE(workflow_id, name)
);

CREATE INDEX idx_workflow_statistics_workflow_id ON workflow_statistics(workflow_id);

-- ============================================================================
-- INITIAL SETTINGS
-- ============================================================================
INSERT INTO settings (key, value, load_on_startup) VALUES
    ('timezone', '"UTC"', TRUE),
    ('userManagement.isInstanceOwnerSetUp', 'false', TRUE),
    ('userManagement.skipInstanceOwnerSetup', 'false', TRUE)
ON CONFLICT (key) DO NOTHING;

-- ============================================================================
-- DOWN MIGRATION
-- ============================================================================
-- To rollback this migration, execute:
-- DROP TABLE IF EXISTS workflow_statistics CASCADE;
-- DROP TABLE IF EXISTS variables CASCADE;
-- DROP TABLE IF EXISTS settings CASCADE;
-- DROP TABLE IF EXISTS webhooks CASCADE;
-- DROP TABLE IF EXISTS credentials CASCADE;
-- DROP TABLE IF EXISTS executions CASCADE;
-- DROP TABLE IF EXISTS workflow_versions CASCADE;
-- DROP TABLE IF EXISTS workflows CASCADE;
-- DROP TABLE IF EXISTS api_keys CASCADE;
-- DROP TABLE IF EXISTS users CASCADE;
