# Othrys

*In Greek mythology, Mount Othrys was the base of the Titans during the Titanomachy — the place where powerful beings gathered, coordinated, and waged war against chaos. Othrys serves a similar purpose: a meeting ground where AI agents assemble, claim their territory, and build together without stepping on each other.*

A coordination server that enables multiple AI coding agents (Claude Code, Cursor, Copilot, etc.) to work on the same project simultaneously without conflicts.

Othrys provides LLM-powered task splitting, branch-per-agent isolation, module-level claim/lock coordination, shared interface contracts, and real-time notifications — all driven by a human PM.

```
                         PM (you)
                           │
                    /othrys-init
                    /othrys-design
                    /othrys-assign
                           │
       ┌───────────────────┼───────────────────┐
       ▼                   ▼                   ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│  Agent Alpha │   │  Agent Beta  │   │  Agent Gamma │
│ (Claude Code)│   │   (Cursor)   │   │  (Copilot)   │
│              │   │              │   │              │
│ /othrys-next │   │ /othrys-next │   │ /othrys-next │
│ /othrys-done │   │ /othrys-done │   │ /othrys-done │
└──────┬───────┘   └──────┬───────┘   └──────┬───────┘
       │                  │                   │
       └──────────────────┼───────────────────┘
                          ▼
              ┌───────────────────────┐
              │     Othrys Server     │
              │                       │
              │  Tasks · Claims ·     │
              │  Events · WebSocket   │
              │                       │
              │     PostgreSQL        │
              └───────────────────────┘
```

---

# Part 1: Using Othrys (PM & Agents)

This section is for anyone who wants to **use** Othrys — you have a server running and want to coordinate agents on a project.

## Prerequisites

- **Claude Code** — [install](https://docs.anthropic.com/en/docs/claude-code)
- **Git** — for branch-per-agent isolation
- **Server URL and API key** — from whoever is hosting the Othrys server

## Installing the Plugin

### First-time setup

```bash
git clone <your-repo-url> && cd othrys
```

Add the marketplace to `~/.claude/settings.json` (replace the path with your actual clone location):

```json
{
  "enabledPlugins": {
    "othrys@othrys-marketplace": true
  },
  "extraKnownMarketplaces": {
    "othrys-marketplace": {
      "source": {
        "source": "directory",
        "path": "/path/to/othrys/marketplace"
      }
    }
  }
}
```

Restart Claude Code. You should see the `/othrys-*` commands available.

### Updating the plugin

```bash
git pull
# Restart Claude Code
```

## Commands

| Command | Role | Purpose |
|---------|------|---------|
| `/othrys-init` | PM | Create a new project on the server |
| `/othrys-design` | PM | Upload a design document and split into tasks |
| `/othrys-tasks` | PM | View tasks — list, filter by status |
| `/othrys-resplit` | PM | Reject proposed tasks and re-split with feedback |
| `/othrys-scaffold` | PM | Generate shared interface contracts (LLM Pass 2) |
| `/othrys-assign` | PM | Assign approved tasks to agents |
| `/othrys-connect` | Agent | Join an existing project |
| `/othrys-next` | Agent | Pick up next assigned task, checkout branch, claim module |
| `/othrys-done` | Agent | Complete task, release claims, commit, switch back |

## Example Workflow

A complete walkthrough: a PM sets up a project, two agents work on it concurrently.

### Phase 1: PM sets up the project

**Terminal 1 — PM's Claude Code session:**

```
> /othrys-init

Create a project on http://your-server:8080 with name "ecommerce-api".

Project created!
  ID:      a1b2c3d4-...
  API Key: 7f8a9b0c...
  Agent:   pm (registered as PM)

Config saved to .othrys/config.json
```

The PM writes a design document (`design.md`) describing the project, then uploads it:

```
> /othrys-design design.md --split

Uploading design... done.
Auto-detected project context:
  Tech stack: Go 1.22, Fiber, PostgreSQL
  Module path: github.com/you/ecommerce-api
  Directory tree: cmd/, internal/, migrations/

Splitting design into tasks...

Proposed tasks (4):
  1. Auth module          internal/auth/       (no deps)
  2. Product service      internal/products/   (no deps)
  3. Cart service         internal/cart/       depends on: Product service
  4. Shared models        internal/models/     (no deps)

Review these tasks. Happy? Approve with approve-all.
Not right? Run /othrys-resplit <your feedback>
```

The PM doesn't like the split — the shared models should be handled by the auth task:

```
> /othrys-resplit merge "Shared models" into the auth module, auth should own internal/models/ too

Rejected 4 proposed tasks.
Re-splitting with PM feedback...

New proposed tasks (3):
  1. Auth + models        internal/auth/, internal/models/   (no deps)
  2. Product service      internal/products/                 (no deps)
  3. Cart service         internal/cart/                     depends on: Product service
```

The PM is happy. Now generate shared contracts so agents don't create conflicting types:

```
> /othrys-scaffold

Generating shared interface contracts...

Contracts (2):
  1. ProductRepository    internal/models/interfaces.go
     GetByID(ctx, id) (*Product, error)
     List(ctx, filter) ([]*Product, error)

  2. CartItem             internal/models/cart.go
     ProductID string, Quantity int

Review and confirm? (y/n)
> y

Contracts stored. They will be included in agent briefings.
```

Approve and assign:

```
> /othrys-tasks

| # | Title            | Status   | Module             | Agent | Deps            |
|---|------------------|----------|--------------------|-------|-----------------|
| 1 | Auth + models    | proposed | internal/auth/     | —     | —               |
| 2 | Product service  | proposed | internal/products/ | —     | —               |
| 3 | Cart service     | proposed | internal/cart/     | —     | Product service |

Total: 3 tasks (3 proposed)
```

The PM approves all tasks (via API: `POST /projects/:id/tasks/approve-all`), then assigns:

```
> /othrys-assign all

Assigned 3 tasks:
  Auth + models     → agent-alice   branch: othrys/agent-alice/auth-models
  Product service   → agent-bob     branch: othrys/agent-bob/product-service
  Cart service      → agent-alice   branch: othrys/agent-alice/cart-service

Agents notified via WebSocket.
```

### Phase 2: Agents work on their tasks

**Terminal 2 — Alice's Claude Code session (different machine):**

The PM shares three things with Alice: the **server URL**, **API key**, and **project ID**.

```
> /othrys-connect http://your-server:8080 7f8a9b0c... a1b2c3d4-...

Connected to Othrys!
  Agent: agent-alice (uuid-...)
  Project: ecommerce-api
  Tasks assigned to you: 2

> /othrys-next

Checking out branch: othrys/agent-alice/auth-models
Claiming module: internal/auth/, internal/models/

═══════════════════════════════════════════════
AGENT BRIEFING
═══════════════════════════════════════════════

## Project Context
Tech stack: Go 1.22, Fiber, PostgreSQL
Module path: github.com/you/ecommerce-api

## Your Task
Auth + models — Build JWT authentication and shared data models.

## Concurrency Notice
You are 1 of 2 agents working simultaneously.
Other tasks in progress:
  - Product service (internal/products/) — agent-bob

## File Ownership
You OWN: internal/auth/, internal/models/
You may READ: internal/products/
You must NOT touch: internal/cart/, internal/products/

## Shared Contracts
You PROVIDE:
  - ProductRepository (internal/models/interfaces.go)
  - CartItem (internal/models/cart.go)

## Rules
- Use github.com/you/ecommerce-api as the module path
- Do not create stubs for other agents' modules
- Do not produce handoff documents
═══════════════════════════════════════════════
```

Alice works on her task. The `collab_guard` hook automatically blocks if she tries to write to `internal/products/` (Bob's territory).

When done:

```
> /othrys-done

Committing changes in internal/auth/, internal/models/...
Releasing claims...
Marking task "Auth + models" as completed.
Switching back to main.

You have 1 more task assigned. Run /othrys-next to pick it up.
```

**Terminal 3 — Bob's session** follows the same flow with `/othrys-connect`, `/othrys-next`, work, `/othrys-done`.

### Phase 3: Merge

Once all agents are done, the PM checks merge readiness:

```bash
othrys merge check
```

```
Merge Readiness Report
  All tasks completed: yes
  Active claims:       0
  Branches:
    - othrys/agent-alice/auth-models
    - othrys/agent-alice/cart-service
    - othrys/agent-bob/product-service
  Conflicts: None detected
```

## Key Concepts

**Module = directory path prefix.** Claiming `internal/auth/` gives exclusive access to all files under that path.

**Branch-per-agent.** Each assigned task gets a branch: `othrys/{agent-name}/{task-slug}`. Agents work in isolation.

**Shared contracts.** When multiple tasks need the same types/interfaces, the PM defines contracts before agents start. Contracts are injected into each agent's briefing so they code against the same interface without seeing each other's work.

**collab_guard hook.** Intercepts file writes and checks a local claims cache. If another agent holds an exclusive claim on the path, the write is blocked — no HTTP calls during writes.

**Agent briefings.** When a task is assigned, the server assembles a rich briefing containing project context, ownership boundaries, sibling task list, shared contracts, and rules. This prevents agents from inventing module paths, creating stub types, or producing handoff documents.

## Task Status Lifecycle

```
proposed → approved → assigned → in_progress → completed
                                              → failed
```

## Claim Conflict Rules

| Existing Claim | New Request | Overlap? | Result |
|---|---|---|---|
| None | Exclusive | — | Granted |
| None | Shared Read | — | Granted |
| Exclusive (other agent) | Any | Yes | Denied |
| Shared Read | Shared Read | Yes | Granted |
| Exclusive (same agent) | Exclusive | Yes | Granted (re-claim) |

## CLI (Alternative to Plugin)

The CLI (`bin/othrys`) provides the same functionality for non-Claude-Code environments or scripting:

```bash
make build          # builds bin/othrys

othrys project create --name "my-app" --repo "https://github.com/you/my-app"
othrys project design --file design.json
othrys project split
othrys task list [--status proposed|approved|assigned|in_progress|completed|failed]
othrys task mine
othrys task approve <ID> [--all]
othrys task assign <TASK_ID> --agent-id <AGENT_ID>
othrys task update <TASK_ID> --status <STATUS>
othrys agent register --name <NAME> --tool <TYPE>
othrys agent list
othrys claim request --task <ID> --path <PATH> [--type exclusive|shared_read]
othrys claim release <CLAIM_ID>
othrys merge check
```

---

# Part 2: Hosting the Server

This section is for whoever runs the Othrys server that agents connect to.

## Prerequisites

- **Docker & Docker Compose**
- **An LLM API key** (Anthropic or OpenAI) — required for task splitting and scaffold generation

## Deployment

### 1. Clone and configure

```bash
git clone <your-repo-url> && cd othrys

cp server/.env.example server/.env
```

Edit `server/.env`:

```env
# Required
API_SECRET=<generate with: openssl rand -hex 32>
POSTGRES_PASSWORD=<strong password>

# Required for LLM task splitting
LLM_PROVIDER=anthropic
LLM_API_KEY=sk-ant-your-key-here
LLM_MODEL=claude-sonnet-4-6

# Optional
PORT=8080
DOMAIN=othrys.yourdomain.com    # for production HTTPS via Caddy
```

| Variable | Required | Default | Description |
|---|---|---|---|
| `API_SECRET` | Yes | — | Secret for internal key generation |
| `POSTGRES_PASSWORD` | Yes | — | PostgreSQL password |
| `LLM_PROVIDER` | No | `anthropic` | `anthropic` or `openai` |
| `LLM_API_KEY` | For splitting | — | API key for the LLM provider |
| `LLM_MODEL` | No | `claude-sonnet-4-6` | Model for task splitting |
| `PORT` | No | `8080` | Server listen port |
| `DOMAIN` | For HTTPS | — | Domain for Caddy TLS |

### 2. Start

```bash
# Start server + PostgreSQL
make server-up

# Verify
curl http://localhost:8080/api/v1/health
# → {"status":"ok"}
```

Migrations run automatically on startup.

### 3. Production deployment

For production with HTTPS, use the Caddy-based compose file:

```bash
docker compose -f server/docker-compose.prod.yml up -d
```

This adds a Caddy reverse proxy with automatic TLS via Let's Encrypt (requires `DOMAIN` set in `.env`).

### 4. Share with agents

Give each agent three things:
1. **Server URL** — `http://your-server:8080` (or `https://othrys.yourdomain.com`)
2. **API Key** — generated when the PM creates a project via `/othrys-init`
3. **Project ID** — also from `/othrys-init`

Agents install the plugin on their machine and run `/othrys-connect` with these values.

## Server Management

```bash
make server-up          # Start server + PostgreSQL
make server-down        # Stop everything
make server-build       # Rebuild Docker image after code changes

# View logs
docker compose -f server/docker-compose.yml logs -f server
```

### Database

5 tables: `projects`, `tasks`, `agents`, `claims`, `events`

Migrations are in `migrations/` and applied automatically on startup. Key indexes:
- Claims: `(project_id, path, status)` WHERE active
- Tasks: `(project_id, status)`
- Events: `(project_id, created_at DESC)`

### Updating

```bash
git pull
make server-build
make server-down && make server-up
```

## REST API Reference

All endpoints except project creation and health require `Authorization: Bearer <api-key>`. Agent-specific endpoints also require `X-Agent-Id: <agent-uuid>` header.

### Projects

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/projects` | Create project (returns API key) |
| `GET` | `/api/v1/projects/:id` | Get project details |
| `PUT` | `/api/v1/projects/:id/design` | Submit PM design document |
| `PUT` | `/api/v1/projects/:id/context` | Set project context (tech stack, conventions) |
| `PUT` | `/api/v1/projects/:id/contracts` | Set shared interface contracts |
| `GET` | `/api/v1/projects/:id/events` | Event feed (paginated) |

### Tasks

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/projects/:id/tasks` | List tasks (`?status=` filter) |
| `POST` | `/api/v1/projects/:id/tasks` | Create a task manually |
| `POST` | `/api/v1/projects/:id/split` | LLM-powered task splitting |
| `POST` | `/api/v1/projects/:id/scaffold` | Generate shared contracts (LLM Pass 2) |
| `POST` | `/api/v1/projects/:id/tasks/approve-all` | Approve all proposed tasks |
| `POST` | `/api/v1/projects/:id/tasks/reject-all` | Reject all proposed tasks |
| `PATCH` | `/api/v1/tasks/:id` | Update task (approve, reject, change status) |
| `POST` | `/api/v1/tasks/:id/assign` | Assign task to agent |

### Agents

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/agents/register` | Register agent (idempotent) |
| `GET` | `/api/v1/projects/:id/agents` | List agents |
| `POST` | `/api/v1/agents/:id/heartbeat` | Agent heartbeat |

### Claims

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/claims` | Request a claim (requires `X-Agent-Id`) |
| `DELETE` | `/api/v1/claims/:id` | Release a claim (requires `X-Agent-Id`) |
| `GET` | `/api/v1/projects/:id/claims` | List active claims |

### Merge

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/projects/:id/merge-check` | Check merge readiness |

## WebSocket

Connect for real-time notifications:

```
ws://host:8080/ws?token=<api-key>&agent=<agent-id>
```

**Server pushes:**

| Message Type | When | Routing |
|---|---|---|
| `task_assigned` | Task assigned to agent | Direct to assigned agent |
| `task_status_changed` | Any task changed status | Broadcast to project |
| `claim_granted` | Claim was granted | Broadcast to project |
| `claim_conflict` | Claim denied (conflict) | Broadcast to project |
| `claim_released` | Claim was released | Broadcast to project |
| `merge_ready` | All tasks done, claims released | Broadcast to project |

**Client sends:**

| Message Type | Purpose |
|---|---|
| `claim_request` | Request a module claim |
| `task_update` | Update task status |
| `claims_sync` | Request full claims snapshot |
| `heartbeat` | Keep connection alive |

---

# Development

For contributors working on the Othrys codebase itself.

```bash
make build              # Build server + CLI → bin/
make server-up          # Start server + PostgreSQL (Docker)
make server-down        # Stop everything
make test               # go test ./... -v -count=1
make vet                # go vet ./...
make plugin-sync        # Sync claude-plugin/ to Claude Code cache
make clean              # rm -rf bin/
```

## Project Structure

```
othrys/
  cmd/server/              Server entry point
  cmd/cli/                 CLI entry point
  internal/
    config/                Environment configuration
    server/handlers/       REST endpoint handlers
    auth/                  API key middleware
    models/                Database models (project, task, agent, claim, event)
    store/                 PostgreSQL repositories
    events/                EventBus interface + PG LISTEN/NOTIFY
    ws/                    WebSocket hub, bridge, connections
    coordinator/           Core logic (claims, tasks, approval, briefing, merge)
    planner/               LLM task splitting + contract generation
    git/                   GitService interface (CLI-side only)
    cli/                   CLI command definitions
  migrations/              SQL migrations (auto-applied on startup)
  claude-plugin/           Claude Code plugin (source of truth)
    commands/              Slash commands (9 commands)
    hooks/                 collab_guard.py, othrys_cache.py
  server/                  Docker deployment (Dockerfile, docker-compose.yml)
  client/                  Client installer (install.sh)
```

## Architecture Notes

- **Server has zero git dependency.** Git ops are client-side only. The server tracks branch names, not repos.
- **EventBus abstraction.** PG LISTEN/NOTIFY today, swappable to Redis/NATS.
- **Claims cache for hooks.** `collab_guard.py` reads local JSON — no HTTP per write. Stale cache allows writes.
- **Module = directory path prefix.** Path overlap = one is a prefix of the other.
- **Briefings assembled server-side.** Works for any agent tool, not just Claude Code.

## License

MIT
