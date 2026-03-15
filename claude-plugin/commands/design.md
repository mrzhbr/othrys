---
name: othrys-design
description: Upload a design document and optionally split it into tasks via LLM
argument-hint: "[path-to-design.json] [--split]"
---

# Othrys Design

The user wants to upload a PM design document to their Othrys project and optionally have the LLM split it into tasks.

## Prerequisites

Read `.othrys/config.json` from the project root to get `server_url`, `api_key`, and `project_id`. If it doesn't exist, tell the user to run `/othrys:init` or `/othrys:connect` first.

## Arguments

- First argument (optional): path to a JSON file containing the design. If not provided, ask the user for the path, or ask if they want to describe the design interactively (in which case, construct the JSON for them).
- `--split` flag (optional): if present, also trigger LLM task splitting after uploading.

## Steps

### 1. Get the design document

If a file path was provided, read it. The design can be any JSON structure — typically something like:

```json
{
  "title": "Feature name",
  "description": "What to build",
  "modules": [
    {"path": "src/auth/", "description": "Authentication module"},
    {"path": "src/api/", "description": "API endpoints"}
  ]
}
```

If no file was provided, ask the user to describe what they want to build. Then construct a design JSON from their description with `title`, `description`, and `modules` fields.

### 2. Auto-detect and upload project context

Before uploading the design, auto-detect the project's tech stack and conventions from the repository:

**Language/framework detection:**
- Check for `go.mod` → Go project. Read module name from first line (`module github.com/...`).
- Check for `package.json` → Node.js project. Read `name` and dependencies.
- Check for `pyproject.toml` or `setup.py` → Python project.
- Check for `Cargo.toml` → Rust project.
- Check for `pom.xml` or `build.gradle` → Java project.

**Directory structure:**
Run the following to get a shallow directory tree (max 3 levels deep):
```bash
find . -type d -maxdepth 3 -not -path './.git/*' -not -path './node_modules/*' -not -path './vendor/*'
```

**Conventions:**
- Read `CLAUDE.md` if it exists (contains project conventions)
- Read the first 100 lines of `README.md` if it exists

Build a context object:
```json
{
  "tech_stack": "<detected language and framework>",
  "module_path": "<detected module/package path>",
  "directory_tree": ["cmd/", "internal/auth/", "..."],
  "conventions": "<key conventions from CLAUDE.md or README>",
  "additional_context": ""
}
```

Present the detected context to the PM for review:
```
Detected project context:
  Tech Stack:    Go 1.22, Fiber, PostgreSQL
  Module Path:   github.com/example/myproject
  Directory Tree: [cmd/, internal/, migrations/, ...]
  Conventions:   [first 200 chars of CLAUDE.md or README]

Is this correct? (press Enter to accept, or describe changes)
```

Upload the context:
```bash
curl -s -X PUT "<SERVER_URL>/api/v1/projects/<PROJECT_ID>/context" \
  -H "Authorization: Bearer <API_KEY>" \
  -H "Content-Type: application/json" \
  -d '<CONTEXT_JSON>'
```

### 3. Upload the design

```bash
curl -s -X PUT "<SERVER_URL>/api/v1/projects/<PROJECT_ID>/design" \
  -H "Authorization: Bearer <API_KEY>" \
  -H "Content-Type: application/json" \
  -d '<DESIGN_JSON>'
```

If this fails, show the error and stop.

### 4. Optionally split into tasks

If the user passed `--split` or if you ask and they confirm, trigger LLM task splitting:

```bash
curl -s -X POST "<SERVER_URL>/api/v1/projects/<PROJECT_ID>/split" \
  -H "Authorization: Bearer <API_KEY>"
```

This returns a list of proposed tasks. Show them in a table:

```
Tasks proposed from design:
  #  Title                    Module Path      Status
  1  Implement auth login     src/auth/        proposed
  2  Add user endpoints       src/api/users/   proposed
  ...

To approve all tasks:
  curl -s -X POST "<SERVER_URL>/api/v1/projects/<PROJECT_ID>/tasks/approve-all" \
    -H "Authorization: Bearer <API_KEY>"

Or approve individually:
  curl -s -X PUT "<SERVER_URL>/api/v1/tasks/<TASK_ID>/status" \
    -H "Authorization: Bearer <API_KEY>" \
    -H "Content-Type: application/json" \
    -d '{"status":"approved"}'
```

After showing the tasks, suggest running the scaffold command:
```
Tip: Run /othrys:scaffold to generate shared interface contracts from these tasks.
This gives all agents a shared foundation of agreed-upon types and interfaces
before they start working concurrently.
```

### 5. If no --split

Print:
```
Design uploaded to project <PROJECT_ID>.

To split into tasks via LLM, run:
  /othrys:design --split
Or manually:
  curl -s -X POST "<SERVER_URL>/api/v1/projects/<PROJECT_ID>/split" \
    -H "Authorization: Bearer <API_KEY>"
```
