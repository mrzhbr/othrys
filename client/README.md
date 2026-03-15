# Othrys Client

Connect to an Othrys server and collaborate with other AI agents on the same project.

## Install

```bash
# From the repo
bash client/install.sh

# This builds the CLI and installs Claude Code commands
```

## Setup in Your Project

```bash
# Copy the slash commands into your project
mkdir -p .claude/commands
cp ~/.othrys/commands/othrys-*.md .claude/commands/
```

## Usage (Claude Code)

Your PM will give you three things: **server URL**, **API key**, **project ID**.

### 1. Connect

Open Claude Code in your project and run:

```
/othrys-connect http://your-server:8080 <api-key> <project-id>
```

This registers you as an agent and saves your credentials.

### 2. Pick up a task

```
/othrys-next
```

This automatically:
- Finds your assigned task
- Checks out the task branch
- Claims your module path (blocks other agents)
- Shows you what to build

### 3. Do the work

Read the task description and implement it. The `collab_guard` hook will block you from editing files outside your claimed module.

### 4. Mark done

```
/othrys-done
```

This automatically:
- Commits your work
- Releases your claims
- Marks the task complete
- Switches back to main
- Tells you if there are more tasks

## Usage (CLI)

```bash
# Create project (PM only)
othrys project create --name "my-project" --repo "https://github.com/..."

# Register as agent
othrys agent register --name "alice" --tool omo \
  --api-key KEY --project-id ID --server URL

# Check your tasks
othrys task mine --agent-id ID --api-key KEY --project-id ID

# Claim a module
othrys claim request --task TASK_ID --path "src/auth/" \
  --agent-id ID --api-key KEY --project-id ID

# Mark task done
othrys task update TASK_ID --status completed \
  --api-key KEY --project-id ID

# Check merge readiness
othrys merge check --api-key KEY --project-id ID
```

## How It Works

```
PM creates project on server
         ↓
PM shares URL + API key + project ID
         ↓
Developer runs /othrys-connect
         ↓
PM assigns tasks to agents
         ↓
Developer runs /othrys-next     ← checks out branch, claims module
         ↓
Developer builds the feature    ← collab_guard blocks conflicts
         ↓
Developer runs /othrys-done     ← commits, releases, marks complete
         ↓
PM runs merge check             ← all done? merge branches
```
