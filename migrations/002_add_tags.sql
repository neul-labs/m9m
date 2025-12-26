-- Migration: 002_add_tags
-- Description: Add tags functionality for organizing workflows
-- Created: 2025-12-26

-- ============================================================================
-- TAGS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS tags (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    color VARCHAR(7), -- Hex color code
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tags_name ON tags(name);

-- ============================================================================
-- WORKFLOW TAGS JUNCTION TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_tags (
    workflow_id VARCHAR(36) NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    tag_id VARCHAR(36) NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (workflow_id, tag_id)
);

CREATE INDEX idx_workflow_tags_workflow_id ON workflow_tags(workflow_id);
CREATE INDEX idx_workflow_tags_tag_id ON workflow_tags(tag_id);

-- ============================================================================
-- TAG USAGE VIEW
-- ============================================================================
CREATE VIEW tag_usage AS
SELECT
    t.id,
    t.name,
    t.color,
    COUNT(wt.workflow_id) as workflow_count
FROM tags t
LEFT JOIN workflow_tags wt ON t.id = wt.tag_id
GROUP BY t.id, t.name, t.color;

-- ============================================================================
-- CREDENTIAL TAGS JUNCTION TABLE (optional - for organizing credentials)
-- ============================================================================
CREATE TABLE IF NOT EXISTS credential_tags (
    credential_id VARCHAR(36) NOT NULL REFERENCES credentials(id) ON DELETE CASCADE,
    tag_id VARCHAR(36) NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (credential_id, tag_id)
);

CREATE INDEX idx_credential_tags_credential_id ON credential_tags(credential_id);
CREATE INDEX idx_credential_tags_tag_id ON credential_tags(tag_id);

-- ============================================================================
-- DEFAULT TAGS
-- ============================================================================
INSERT INTO tags (id, name, color, description) VALUES
    ('tag-production', 'Production', '#22c55e', 'Production workflows'),
    ('tag-development', 'Development', '#3b82f6', 'Development/test workflows'),
    ('tag-archived', 'Archived', '#6b7280', 'Archived workflows'),
    ('tag-critical', 'Critical', '#ef4444', 'Critical business workflows'),
    ('tag-scheduled', 'Scheduled', '#f59e0b', 'Scheduled/cron workflows')
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- DOWN MIGRATION
-- ============================================================================
-- To rollback this migration, execute:
-- DROP VIEW IF EXISTS tag_usage;
-- DROP TABLE IF EXISTS credential_tags CASCADE;
-- DROP TABLE IF EXISTS workflow_tags CASCADE;
-- DROP TABLE IF EXISTS tags CASCADE;
