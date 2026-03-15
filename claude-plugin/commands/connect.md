---
name: othrys-connect
description: Connect to an Othrys project — register your agent and start receiving tasks
---

# Othrys Connect

The user wants to connect to an Othrys coordination server. They should provide (or you should ask for):
- **Server URL** (e.g. `http://localhost:8080`)
- **API Key** (from the PM)
- **Project ID** (from the PM)

If they provided these as arguments (e.g. `/othrys-connect http://localhost:8080 KEY ID`), parse them. Otherwise ask.

## Steps

### 1. Save config

Create `.othrys/` directory in the project root and write the connection config to `.othrys/config.json`:

```json
{
  "server_url": "<URL>",
  "api_key": "<KEY>",
  "project_id": "<ID>",
  "agent_id": "",
  "agent_name": "",
  "tool_type": "claude-code"
}
```

### 2. Register agent

Pick an agent name — use the machine hostname or ask the user. Then register via the REST API:

```bash
curl -s -X POST "<URL>/api/v1/agents/register" \
  -H "Authorization: Bearer <KEY>" \
  -H "Content-Type: application/json" \
  -d '{"project_id":"<ID>","name":"<AGENT_NAME>","tool_type":"claude-code"}'
```

Save the returned agent ID into `.othrys/config.json`.

### 3. Show status

Query the task list:

```bash
curl -s "<URL>/api/v1/projects/<ID>/tasks" \
  -H "Authorization: Bearer <KEY>"
```

Show:
- How many tasks exist and their statuses
- Which tasks (if any) are already assigned to this agent

### 4. Tell the user what to do next

Print:
```
Connected to Othrys!
  Agent: <NAME> (<AGENT_ID>)
  Project: <PROJECT_ID>

Use /othrys-next to pick up your next assigned task.
If the PM hasn't assigned you a task yet, ask them to run:
  othrys task assign <TASK_ID> --agent-id <AGENT_ID>
```
