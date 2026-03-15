-- Migration 003 rollback: Remove task enrichment fields

-- Remove columns from tasks table
ALTER TABLE tasks
    DROP COLUMN IF EXISTS read_only_paths,
    DROP COLUMN IF EXISTS forbidden_paths,
    DROP COLUMN IF EXISTS integration_points,
    DROP COLUMN IF EXISTS contracts,
    DROP COLUMN IF EXISTS agent_briefing;

-- Remove columns from projects table
ALTER TABLE projects
    DROP COLUMN IF EXISTS shared_contracts,
    DROP COLUMN IF EXISTS project_context;
