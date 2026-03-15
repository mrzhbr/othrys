-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table: projects
CREATE TABLE IF NOT EXISTS projects (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(255) NOT NULL UNIQUE,
    repo_url    TEXT NOT NULL DEFAULT '',
    api_key     VARCHAR(64) NOT NULL UNIQUE,
    pm_design   JSONB,
    config      JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Table: tasks
CREATE TYPE task_status AS ENUM ('proposed', 'approved', 'assigned', 'in_progress', 'completed', 'failed');

CREATE TABLE IF NOT EXISTS tasks (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id        UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title             VARCHAR(500) NOT NULL,
    description       TEXT NOT NULL DEFAULT '',
    module_path       VARCHAR(500) NOT NULL DEFAULT '',
    status            task_status NOT NULL DEFAULT 'proposed',
    assigned_agent_id UUID,
    branch_name       VARCHAR(255),
    depends_on        UUID[] NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Table: agents
CREATE TYPE agent_tool_type AS ENUM ('omo', 'cursor', 'copilot', 'generic');
CREATE TYPE agent_status AS ENUM ('idle', 'working', 'disconnected');

CREATE TABLE IF NOT EXISTS agents (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id     UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name           VARCHAR(255) NOT NULL,
    tool_type      agent_tool_type NOT NULL DEFAULT 'generic',
    status         agent_status NOT NULL DEFAULT 'idle',
    branch_name    VARCHAR(255),
    last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    connected_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, name)
);

-- Table: claims
CREATE TYPE claim_type AS ENUM ('exclusive', 'shared_read');
CREATE TYPE claim_status AS ENUM ('active', 'released', 'revoked');

CREATE TABLE IF NOT EXISTS claims (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    agent_id    UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    task_id     UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    path        VARCHAR(1000) NOT NULL,
    claim_type  claim_type NOT NULL DEFAULT 'exclusive',
    status      claim_status NOT NULL DEFAULT 'active',
    granted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at TIMESTAMPTZ
);

-- Table: events
CREATE TABLE IF NOT EXISTS events (
    id          BIGSERIAL PRIMARY KEY,
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    event_type  VARCHAR(100) NOT NULL,
    agent_id    UUID,
    payload     JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add FK constraint for tasks.assigned_agent_id after agents table created
ALTER TABLE tasks
    ADD CONSTRAINT fk_tasks_agent
    FOREIGN KEY (assigned_agent_id)
    REFERENCES agents(id)
    ON DELETE SET NULL;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_claims_active
    ON claims (project_id, path, status)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_tasks_project_status
    ON tasks (project_id, status);

CREATE INDEX IF NOT EXISTS idx_agents_project_status
    ON agents (project_id, status);

CREATE INDEX IF NOT EXISTS idx_events_project_created
    ON events (project_id, created_at DESC);
