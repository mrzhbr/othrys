-- Drop indexes
DROP INDEX IF EXISTS idx_events_project_created;
DROP INDEX IF EXISTS idx_agents_project_status;
DROP INDEX IF EXISTS idx_tasks_project_status;
DROP INDEX IF EXISTS idx_claims_active;

-- Drop tables (in reverse FK dependency order)
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS claims;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS projects;

-- Drop types
DROP TYPE IF EXISTS claim_status;
DROP TYPE IF EXISTS claim_type;
DROP TYPE IF EXISTS agent_status;
DROP TYPE IF EXISTS agent_tool_type;
DROP TYPE IF EXISTS task_status;
