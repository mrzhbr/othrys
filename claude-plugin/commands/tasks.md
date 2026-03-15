---
name: othrys-tasks
description: View and manage tasks in your Othrys project — list, filter, inspect
---

# Othrys Tasks

The PM wants to view and manage tasks in the current Othrys project.

## Steps

### 1. Load config

Read `.othrys/config.json` to get `server_url`, `api_key`, and `project_id`.
If the file doesn't exist, tell the user to run `/othrys-init` or `/othrys-connect` first.

### 2. Parse arguments

The user may pass arguments like:
- `/othrys-tasks` — show all tasks
- `/othrys-tasks proposed` — filter by status
- `/othrys-tasks approved` — filter by status

Valid statuses: `proposed`, `approved`, `assigned`, `in_progress`, `completed`, `failed`

### 3. Fetch tasks

```bash
curl -s "<URL>/api/v1/projects/<PROJECT_ID>/tasks" \
  -H "Authorization: Bearer <KEY>"
```

If a status filter was given, append `?status=<STATUS>` to the URL.

### 4. Display tasks

Show tasks in a clear table format:

```
# Othrys Tasks — <PROJECT_ID>

| # | Title | Status | Module | Agent | Dependencies |
|---|-------|--------|--------|-------|--------------|
| 1 | Setup database | proposed | internal/store/ | — | — |
| 2 | Build API | proposed | internal/server/ | — | Setup database |

Total: N tasks (X proposed, Y approved, Z assigned, ...)
```

For each task, show:
- **Title**
- **Status** (proposed/approved/assigned/in_progress/completed/failed)
- **Module path** (the directory this task owns)
- **Assigned agent** (name if assigned, "—" if not)
- **Dependencies** (titles of tasks in depends_on, "—" if none)

Group by status if showing all tasks. Show status counts in the summary line.

### 5. Show available actions

Based on what tasks exist, suggest next steps:

- If there are **proposed** tasks: "Use `/othrys-assign` to assign tasks to agents, or POST approve-all to approve them first."
- If there are **approved** tasks: "Use `/othrys-assign` to assign approved tasks to agents."
- If all tasks are **completed**: "All tasks done! Use `othrys merge check` to verify merge readiness."
- If there are **failed** tasks: "Some tasks failed. Review them and consider re-assigning."
