---
name: othrys-scaffold
description: Generate shared interface/type contracts from split tasks (LLM Pass 2) and let PM review before agents start
---

# Othrys Scaffold

The PM wants to generate shared contracts (interface/type definitions) that span multiple tasks,
review them, and optionally commit stub files before agents start working.

This runs Pass 2 of the two-pass LLM split flow. It takes the already-split tasks and asks the LLM
to define the shared interfaces/types that multiple agents will need to agree on.

## Prerequisites

Read `.othrys/config.json` for server_url, api_key, project_id.
If it doesn't exist, tell the user to run `/othrys:init` or `/othrys:connect` first.
Tasks must already be split (run `/othrys:design --split` first).

## Steps

### 1. Generate contracts via LLM Pass 2

```bash
curl -s -X POST "<SERVER_URL>/api/v1/projects/<PROJECT_ID>/scaffold" \
  -H "Authorization: Bearer <API_KEY>"
```

If the response is 501 (Not Implemented), tell the user:
```
The configured LLM provider does not support contract generation.
Only AnthropicProvider supports this feature.
```
And stop.

If the response is 400 (no tasks), tell the user to run `/othrys:design --split` first.

### 2. Display generated contracts

Show the contracts in a readable format:

```
Generated Contracts:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Contract 1: <NAME>
File: <FILE_PATH>
Definition:
  <DESCRIPTION>

Contract 2: <NAME>
...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### 3. Ask PM to review and confirm

Present options:
```
Review the contracts above. Choose an action:
  [1] Accept all contracts (store as-is)
  [2] Edit contracts (describe your changes)
  [3] Reject and regenerate
  [4] Skip contracts (agents will define their own interfaces)
```

Wait for the PM's response.

### 4. If confirmed (option 1 or after edits)

Store the final contracts by calling:

```bash
curl -s -X PUT "<SERVER_URL>/api/v1/projects/<PROJECT_ID>/contracts" \
  -H "Authorization: Bearer <API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{"contracts": [<FINAL_CONTRACTS_JSON>]}'
```

Confirm: "Contracts stored. These will be included in each agent's briefing."

### 5. Optionally generate stub files

Ask the PM:
```
Would you like to generate stub files on the current branch?
This gives every agent branch the shared interface definitions as a starting point.
  [y] Yes, create stub files
  [n] No, agents will see contracts in their briefings only
```

If yes:
- For each contract, create the stub file at the specified `file_path`
- Write a minimal stub containing the interface/type definition from the description
- Report which files were created
- Remind the PM: "Commit these files so they appear on every agent's branch."

If no:
- Print: "Contracts will be injected into agent briefings at assignment time."
