#!/usr/bin/env bash
# setup-othrys.sh — Configure Othrys integration for this project.
#
# Creates .claude/omo/othrys.json, registers the agent with the Othrys server,
# and performs an initial claims cache warm.
#
# Usage:
#   bash setup-othrys.sh
#   bash setup-othrys.sh --server http://... --api-key <key> --agent-name <name> --project-id <id>
#
# Non-interactive mode (all flags provided):
#   bash setup-othrys.sh \
#     --server http://localhost:8080 \
#     --api-key abc123 \
#     --project-id <uuid> \
#     --agent-name "omo-alice" \
#     --tool-type omo

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGIN_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CONFIG_FILE=".claude/omo/othrys.json"
CACHE_SCRIPT="${PLUGIN_ROOT}/hooks/othrys_cache.py"

# ── Helpers ──────────────────────────────────────────────────────────────────

info()    { echo "[othrys] $*"; }
success() { echo "[othrys] ✓ $*"; }
warn()    { echo "[othrys] WARNING: $*" >&2; }
error()   { echo "[othrys] ERROR: $*" >&2; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || error "$1 is required but not found in PATH"
}

# ── Parse flags ───────────────────────────────────────────────────────────────

SERVER_URL=""
API_KEY=""
PROJECT_ID=""
AGENT_NAME=""
TOOL_TYPE="omo"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --server)      SERVER_URL="$2";   shift 2 ;;
    --api-key)     API_KEY="$2";      shift 2 ;;
    --project-id)  PROJECT_ID="$2";   shift 2 ;;
    --agent-name)  AGENT_NAME="$2";   shift 2 ;;
    --tool-type)   TOOL_TYPE="$2";    shift 2 ;;
    *) error "Unknown argument: $1" ;;
  esac
done

# ── Prerequisites ─────────────────────────────────────────────────────────────

require_cmd python3
require_cmd curl

# ── Interactive prompts (if not provided via flags) ───────────────────────────

if [[ -z "$SERVER_URL" ]]; then
  read -rp "Othrys server URL [http://localhost:8080]: " SERVER_URL
  SERVER_URL="${SERVER_URL:-http://localhost:8080}"
fi
SERVER_URL="${SERVER_URL%/}"  # strip trailing slash

if [[ -z "$PROJECT_ID" ]]; then
  read -rp "Project ID (UUID from 'othrys project create'): " PROJECT_ID
fi

if [[ -z "$API_KEY" ]]; then
  read -rsp "Project API key: " API_KEY
  echo ""
fi

if [[ -z "$AGENT_NAME" ]]; then
  default_name="omo-$(hostname | tr '[:upper:]' '[:lower:]' | tr -d '.')"
  read -rp "Agent name [${default_name}]: " AGENT_NAME
  AGENT_NAME="${AGENT_NAME:-$default_name}"
fi

info "Tool type: ${TOOL_TYPE} (use --tool-type to change)"

# ── Verify server connection ──────────────────────────────────────────────────

info "Checking server connection to ${SERVER_URL}..."
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer ${API_KEY}" \
  "${SERVER_URL}/api/v1/projects/${PROJECT_ID}" || echo "000")

if [[ "$HTTP_STATUS" != "200" ]]; then
  error "Cannot reach Othrys server or invalid credentials (HTTP ${HTTP_STATUS}). Check SERVER_URL, PROJECT_ID, and API_KEY."
fi
success "Server connection verified."

# ── Register agent with Othrys server ─────────────────────────────────────────

info "Registering agent '${AGENT_NAME}' with project ${PROJECT_ID}..."
REGISTER_RESPONSE=$(curl -s -X POST \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d "{\"project_id\": \"${PROJECT_ID}\", \"name\": \"${AGENT_NAME}\", \"tool_type\": \"${TOOL_TYPE}\"}" \
  "${SERVER_URL}/api/v1/agents/register")

# Extract agent ID from response
AGENT_ID=$(echo "${REGISTER_RESPONSE}" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('id',''))" 2>/dev/null || echo "")
if [[ -z "$AGENT_ID" ]]; then
  echo "Register response: ${REGISTER_RESPONSE}" >&2
  error "Failed to register agent. Check the server response above."
fi
success "Agent registered: ID=${AGENT_ID}"

# ── Write config file ─────────────────────────────────────────────────────────

info "Writing config to ${CONFIG_FILE}..."
mkdir -p "$(dirname "${CONFIG_FILE}")"
python3 - << PYEOF
import json, os
config = {
    "server_url": "${SERVER_URL}",
    "api_key": "${API_KEY}",
    "project_id": "${PROJECT_ID}",
    "agent_id": "${AGENT_ID}",
    "agent_name": "${AGENT_NAME}",
    "tool_type": "${TOOL_TYPE}",
}
config_file = "${CONFIG_FILE}"
os.makedirs(os.path.dirname(config_file), exist_ok=True)
with open(config_file, "w") as f:
    json.dump(config, f, indent=2)
print(f"Config written to {config_file}")
PYEOF
success "Config file written."

# ── Warm the claims cache ─────────────────────────────────────────────────────

info "Warming claims cache..."
if python3 "${CACHE_SCRIPT}" refresh; then
  success "Claims cache warmed."
else
  warn "Cache warm failed — you can run 'python3 ${CACHE_SCRIPT} refresh' manually later."
fi

# ── Done ──────────────────────────────────────────────────────────────────────

echo ""
echo "╔═══════════════════════════════════════════════════════╗"
echo "║           Othrys setup complete!                     ║"
echo "╠═══════════════════════════════════════════════════════╣"
echo "║  Server:     ${SERVER_URL}"
echo "║  Project ID: ${PROJECT_ID}"
echo "║  Agent ID:   ${AGENT_ID}"
echo "║  Agent Name: ${AGENT_NAME}"
echo "╠═══════════════════════════════════════════════════════╣"
echo "║  Config:     ${CONFIG_FILE}"
echo "║  Cache:      .claude/omo/.othrys_claims_cache.json"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""
info "The collab_guard hook is now active. Before editing files, use:"
info "  /othrys-claim <path>    — to claim a module path"
info "  /othrys-status          — to see project status and active claims"
