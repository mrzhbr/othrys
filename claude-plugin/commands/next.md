---
name: othrys-next
description: Pick up your next assigned task from Othrys — checkout branch, claim module, show instructions
---

# Othrys Next Task

The user wants to start working on their next Othrys task. This command automates the full setup.

## Steps

### 1. Load config

Read `.othrys/config.json` for server_url, api_key, project_id, agent_id.
If it doesn't exist, tell the user to run `/othrys-connect` first.

### 2. Poll for assigned tasks

Query tasks with status "assigned" or "in_progress" that belong to this agent:

```bash
curl -s "<URL>/api/v1/projects/<PROJECT_ID>/tasks?status=assigned" \
  -H "Authorization: Bearer <KEY>"
```

Also check `?status=in_progress`. Filter results where `assigned_agent_id` matches our agent_id.

If no tasks are assigned, tell the user:
```
No tasks assigned to you yet. Ask the PM to assign one:
  othrys task assign <TASK_ID> --agent-id <AGENT_ID>
```
And stop here.

### 3. Pick the first assigned/in_progress task

From the results, take the first task. Extract:
- Task ID
- Title
- Description
- Module path
- Branch name
- Agent briefing (if present)

### 4. Checkout the branch

```bash
git checkout -b <BRANCH_NAME> 2>/dev/null || git checkout <BRANCH_NAME>
```

### 5. Claim the module

```bash
curl -s -X POST "<URL>/api/v1/claims" \
  -H "Authorization: Bearer <KEY>" \
  -H "X-Agent-Id: <AGENT_ID>" \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"<AGENT_ID>","task_id":"<TASK_ID>","path":"<MODULE_PATH>","claim_type":"exclusive"}'
```

If the response is 409 (conflict), tell the user which agent holds the conflicting claim and stop.

### 6. Mark task as in_progress

```bash
curl -s -X PATCH "<URL>/api/v1/tasks/<TASK_ID>" \
  -H "Authorization: Bearer <KEY>" \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress"}'
```

### 7. Save current task context

Write the active task info to `.othrys/current_task.json`:

```json
{
  "task_id": "<TASK_ID>",
  "title": "<TITLE>",
  "description": "<DESCRIPTION>",
  "module_path": "<MODULE_PATH>",
  "branch_name": "<BRANCH_NAME>",
  "agent_briefing": "<AGENT_BRIEFING>",
  "started_at": "<ISO_TIMESTAMP>"
}
```

### 8. Present the task to the user

If the task has a non-empty `agent_briefing` field, display it verbatim — it already contains
all context including project info, concurrency notice, file ownership, shared contracts, and rules:

```
━━━ Task: <TITLE> ━━━
Branch: <BRANCH_NAME>

<AGENT_BRIEFING>

IMPORTANT: Only create or modify files under your owned module path. Read the Shared Contracts section above before writing any code.
```

If `agent_briefing` is empty (backward compatibility with older server versions), fall back to
the simple display format:

```
━━━ Task: <TITLE> ━━━
Module:  <MODULE_PATH>
Branch:  <BRANCH_NAME>

Description:
<FULL_DESCRIPTION>

You are now on branch <BRANCH_NAME> with an exclusive claim on <MODULE_PATH>.
No other agent can edit files in this path.

Plan and implement this task. When you're done, run /othrys-done.
```

Do NOT start implementing. Wait for the user to give instructions.
