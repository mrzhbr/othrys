# Othrys — Multi-Agent Collaboration Plugin

You have access to the Othrys coordination system. When the user uses Othrys commands:

- `/othrys:init` — Create a new project on the server and connect as the first agent
- `/othrys:connect` — Join an existing project (needs server URL, API key, project ID)
- `/othrys:design` — Upload a PM design document and optionally split into tasks
- `/othrys:tasks` — View and manage tasks (list, filter by status)
- `/othrys:assign` — Assign approved tasks to agents (single, bulk, or auto-distribute)
- `/othrys:resplit` — Reject proposed tasks and re-split with optional PM feedback
- `/othrys:scaffold` — Generate shared interface contracts (LLM Pass 2)
- `/othrys:next` — Pick up your next assigned task (checks out branch, claims module)
- `/othrys:done` — Complete current task (commits, releases claims, marks done)

Config is stored in `.othrys/config.json` in the project root.
The `collab_guard` hook will block writes to paths claimed by other agents.
