---
name: othrys-done
description: Mark current Othrys task as complete — release claims, commit, and switch back
---

# Othrys Task Done

The user has finished their current task and wants to wrap up.

## Steps

### 1. Load config and task context

Read `.othrys/config.json` for connection details (server_url, api_key, project_id, agent_id).
Read `.othrys/current_task.json` for the active task (task_id, title, module_path, branch_name).

If no active task file exists, tell the user there's nothing to complete.

### 2. Commit any uncommitted work

Check `git status`. If there are uncommitted changes:
- Stage all changes in the module path: `git add <MODULE_PATH>`
- Create a commit with a message based on the task title
- Example: `feat: implement <task title>`

Ask the user before committing if there are changes outside the module path.

### 3. Release all claims

List active claims for the project and release ones owned by this agent:

```bash
curl -s "<URL>/api/v1/projects/<PROJECT_ID>/claims" \
  -H "Authorization: Bearer <KEY>"
```

For each claim where agent_id matches our agent:
```bash
curl -s -X DELETE "<URL>/api/v1/claims/<CLAIM_ID>" \
  -H "Authorization: Bearer <KEY>" \
  -H "X-Agent-Id: <AGENT_ID>"
```

### 4. Mark task as completed

```bash
curl -s -X PATCH "<URL>/api/v1/tasks/<TASK_ID>" \
  -H "Authorization: Bearer <KEY>" \
  -H "Content-Type: application/json" \
  -d '{"status":"completed"}'
```

### 5. Switch back to main branch

```bash
git checkout main
```

### 6. Clean up task context

Delete `.othrys/current_task.json`.

### 7. Check if there are more tasks

Query for more assigned tasks for this agent (same as /othrys-next step 2).

### 8. Report

Print:
```
━━━ Task Complete ━━━
  Task:   <TITLE>
  Branch: <BRANCH_NAME>
  Status: completed
  Claims: released

<If more tasks assigned:>
You have more tasks assigned. Run /othrys-next to pick up the next one.

<If no more tasks:>
No more tasks assigned. Ask the PM for your next assignment.
```
