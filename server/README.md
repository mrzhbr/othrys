# Othrys Server

Coordination server for multi-agent software development. Deploy this on your VPS — developers connect using the [Othrys client](../client/).

## Quick Deploy

```bash
# 1. Configure
cp .env.example .env
# Edit .env — set API_SECRET and POSTGRES_PASSWORD at minimum

# 2. Start
docker compose up -d

# 3. Verify
curl http://your-vps:8080/api/v1/health
# → {"status":"ok"}
```

## Create a Project

```bash
curl -X POST http://your-vps:8080/api/v1/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"my-project","repo_url":"https://github.com/you/repo"}'
```

This returns a **Project ID** and **API Key**. Share these with your team.

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `API_SECRET` | Yes | — | Internal secret (generate: `openssl rand -hex 32`) |
| `POSTGRES_PASSWORD` | Yes | `othrys` | PostgreSQL password |
| `PORT` | No | `8080` | Server port |
| `LLM_PROVIDER` | No | `anthropic` | `anthropic` or `openai` |
| `LLM_API_KEY` | No | — | Only needed for LLM task splitting |
| `LLM_MODEL` | No | `claude-sonnet-4-6` | Model for task splitting |

## What the Server Does

- Stores projects, tasks, agents, claims, events in PostgreSQL
- Coordinates module-level claims (prevents overlapping edits)
- Provides REST API + WebSocket for real-time events
- Optionally splits PM designs into tasks via LLM
- Checks merge readiness (all tasks done, all claims released)

## What the Server Does NOT Do

- No git operations (git is client-side only)
- No code execution
- No direct access to developer machines or repos
