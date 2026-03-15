---
name: othrys-resplit
description: Reject current proposed tasks and re-split the design with optional feedback
---

# Othrys Re-split

The PM is not happy with the current task split and wants to re-do it. This command rejects all proposed tasks and triggers a new LLM split, optionally with feedback to guide the splitter.

## Steps

### 1. Load config

Read `.othrys/config.json` to get `server_url`, `api_key`, and `project_id`.
If the file doesn't exist, tell the user to run `/othrys-init` or `/othrys-connect` first.

### 2. Check current state

Fetch existing tasks:

```bash
curl -s "<URL>/api/v1/projects/<PROJECT_ID>/tasks" \
  -H "Authorization: Bearer <KEY>"
```

Check if there are tasks in non-proposed status (approved, assigned, in_progress):
- If yes, **WARN the user**: "There are N tasks that are already approved/assigned/in_progress. Re-splitting only removes PROPOSED tasks. You may want to handle those manually first."
- If all tasks are proposed, proceed directly.
- If there are no tasks at all, skip rejection and go straight to re-split.

Show current proposed tasks briefly:
```
Current proposed tasks (will be deleted):
  1. Setup database (internal/store/)
  2. Build API (internal/server/)
  ...

These will be rejected and replaced with a new split.
```

### 3. Parse feedback

The user may pass feedback as arguments:
- `/othrys-resplit` — re-split with no changes (same design, fresh LLM attempt)
- `/othrys-resplit split into fewer, larger tasks` — feedback for the splitter
- `/othrys-resplit the auth module should be separate from the API handlers` — specific guidance

If feedback was provided, update the project's design document by appending the feedback:

```bash
# First get the current project to read the existing design
curl -s "<URL>/api/v1/projects/<PROJECT_ID>" \
  -H "Authorization: Bearer <KEY>"
```

Then update the design with feedback appended:

```bash
curl -s -X PUT "<URL>/api/v1/projects/<PROJECT_ID>/design" \
  -H "Authorization: Bearer <KEY>" \
  -H "Content-Type: application/json" \
  -d '{"pm_design": "<EXISTING_DESIGN>\n\n---\nSPLITTING GUIDANCE (from PM):\n<FEEDBACK>"}'
```

### 4. Reject all proposed tasks

```bash
curl -s -X POST "<URL>/api/v1/projects/<PROJECT_ID>/tasks/reject-all" \
  -H "Authorization: Bearer <KEY>"
```

Show: "Rejected N proposed tasks."

### 5. Re-split

```bash
curl -s -X POST "<URL>/api/v1/projects/<PROJECT_ID>/split" \
  -H "Authorization: Bearer <KEY>"
```

This may take 30-60 seconds. Tell the user: "Re-splitting design into tasks..."

### 6. Show new tasks

Display the new proposed tasks in a table:

```
New task split (N tasks):

| # | Title | Module | Dependencies |
|---|-------|--------|--------------|
| 1 | ... | ... | ... |

Review these tasks:
- Happy? Run: /othrys-tasks to view details, then approve with approve-all
- Still not right? Run: /othrys-resplit <your feedback>
- Want to scaffold shared contracts? Run: /othrys-scaffold
```

### 7. Edge cases

- If the split endpoint returns an error (e.g., LLM failure, no design set), show the error and suggest:
  - "Make sure a design document is uploaded: /othrys-design <path>"
  - "Check that the LLM provider is configured on the server"
- If reject-all fails, show the error and stop (don't split with stale tasks).
