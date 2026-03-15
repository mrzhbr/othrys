---
name: othrys-assign
description: Assign approved tasks to agents — single, multiple, or auto-distribute
---

# Othrys Assign

The PM wants to assign tasks to agents.

## Steps

### 1. Load config

Read `.othrys/config.json` to get `server_url`, `api_key`, and `project_id`.
If the file doesn't exist, tell the user to run `/othrys-init` or `/othrys-connect` first.

### 2. Fetch approved tasks

```bash
curl -s "<URL>/api/v1/projects/<PROJECT_ID>/tasks?status=approved" \
  -H "Authorization: Bearer <KEY>"
```

If no approved tasks exist, check for proposed tasks and suggest approving them first:
```
No approved tasks found. There are N proposed tasks.
Run this to approve them all:
  curl -s -X POST "<URL>/api/v1/projects/<PROJECT_ID>/tasks/approve-all" -H "Authorization: Bearer <KEY>"
Or approve individually via PATCH /api/v1/tasks/<TASK_ID> with {"approve": true}
```

### 3. Fetch available agents

```bash
curl -s "<URL>/api/v1/projects/<PROJECT_ID>/agents" \
  -H "Authorization: Bearer <KEY>"
```

If no agents are registered besides the PM, tell the user:
```
No worker agents registered yet. Have each agent run /othrys-connect to join the project.
```

### 4. Parse arguments

The user may pass arguments like:
- `/othrys-assign` — interactive: show tasks and agents, ask which to assign
- `/othrys-assign all` — auto-distribute all approved tasks across all available agents (round-robin)
- `/othrys-assign <TASK_TITLE_OR_ID> <AGENT_NAME_OR_ID>` — assign a specific task to a specific agent

### 5. Assign tasks

For each assignment, call:

```bash
curl -s -X POST "<URL>/api/v1/tasks/<TASK_ID>/assign" \
  -H "Authorization: Bearer <KEY>" \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"<AGENT_ID>"}'
```

The server will:
- Generate a branch name: `othrys/{agent-name}/{task-slug}`
- Assemble a rich agent briefing with project context, ownership boundaries, and shared contracts
- Send a WebSocket notification to the assigned agent

### 6. Show results

After assigning, show:

```
Assigned N tasks:

| Task | Agent | Branch |
|------|-------|--------|
| Setup database | agent-1 | othrys/agent-1/setup-database |
| Build API | agent-2 | othrys/agent-2/build-api |

Agents will be notified via WebSocket. They can pick up their task with /othrys-next.
```

If any assignment fails (e.g., conflict), show the error and continue with remaining tasks.

### 7. Interactive mode (no arguments)

If called without arguments, present the available tasks and agents:

```
Approved tasks ready for assignment:
  1. Setup database (internal/store/)
  2. Build API (internal/server/)
  3. Add auth (internal/auth/)

Available agents:
  a. agent-1 (claude-code) — idle
  b. agent-2 (cursor) — idle

Assign all tasks round-robin? Or specify: <task#> -> <agent letter>
Example: "1 -> a, 2 -> b, 3 -> a" or just "all"
```

Wait for user input, then execute the assignments.
