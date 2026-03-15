-- Migration 003: Add task enrichment fields for concurrent agent coordination

-- Add new columns to tasks table
ALTER TABLE tasks
    ADD COLUMN read_only_paths    TEXT[]  NOT NULL DEFAULT '{}',
    ADD COLUMN forbidden_paths    TEXT[]  NOT NULL DEFAULT '{}',
    ADD COLUMN integration_points TEXT[]  NOT NULL DEFAULT '{}',
    ADD COLUMN contracts          JSONB   NOT NULL DEFAULT '[]',
    ADD COLUMN agent_briefing     TEXT    NOT NULL DEFAULT '';

-- Add new columns to projects table
ALTER TABLE projects
    ADD COLUMN shared_contracts  JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN project_context   JSONB NOT NULL DEFAULT '{}';
