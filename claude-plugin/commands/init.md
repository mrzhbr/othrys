---
name: othrys-init
description: Create a new Othrys project on the server and connect as the first agent (PM setup)
---

# Othrys Init

The user wants to create a new project on an Othrys server and connect to it. They should provide (or you should ask for):
- **Server URL** (e.g. `http://my-vps.tail1234.ts.net:8080`)
- **Project name**
- **Repo URL** (their GitHub/GitLab repo — just metadata, server never touches git)

If they provided these as arguments (e.g. `/othrys-init http://server:8080 my-project https://github.com/...`), parse them. Otherwise ask.

## Steps

### 1. Create the project

```bash
curl -s -X POST "<URL>/api/v1/projects" \
  -H "Content-Type: application/json" \
  -d '{"name":"<PROJECT_NAME>","repo_url":"<REPO_URL>"}'
```

This returns `id` (project ID) and `api_key`. Save both.

If the server returns an error, show it and stop.

### 2. Save config

Create `.othrys/` directory in the project root and write:

```json
{
  "server_url": "<URL>",
  "api_key": "<API_KEY from step 1>",
  "project_id": "<ID from step 1>",
  "agent_id": "",
  "agent_name": "",
  "tool_type": "claude-code"
}
```

### 3. Register as first agent

Pick an agent name — use the machine hostname or ask the user. Register:

```bash
curl -s -X POST "<URL>/api/v1/agents/register" \
  -H "Authorization: Bearer <API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{"project_id":"<ID>","name":"<AGENT_NAME>","tool_type":"claude-code"}'
```

Save the returned agent ID into `.othrys/config.json`.

### 4. Show result

Print:
```
Othrys project created!
  Project: <PROJECT_NAME> (<PROJECT_ID>)
  API Key: <API_KEY>
  Agent:   <AGENT_NAME> (<AGENT_ID>)

Share the server URL, API key, and project ID with your team.
They connect with: /othrys-connect <URL> <API_KEY> <PROJECT_ID>

Next steps:
  - Upload a design:    othrys project design --file design.json --api-key <KEY> --project-id <ID>
  - Or create tasks:    othrys task create --title "..." --module-path "src/..." --api-key <KEY> --project-id <ID>
  - Assign to agents:   othrys task assign <TASK_ID> --agent-id <AGENT_ID> --api-key <KEY> --project-id <ID>
```
