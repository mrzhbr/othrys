#!/usr/bin/env python3
"""
collab_guard.py — OmO PreToolUse hook for Othrys collaboration conflict detection.

Intercepts Write and Edit tool calls and checks the local claims cache to ensure
no other agent holds an exclusive claim on the target file path.

Cache-first approach:
  1. Load local cache at .claude/omo/.othrys_claims_cache.json
  2. If cache is fresh (updated_at within TTL): use cache for decision
  3. If cache is stale or missing: ALLOW the write and log a warning
     (never block on stale/unavailable data)
  4. If another agent holds an exclusive claim on an overlapping path: BLOCK

Othrys config is read from .claude/omo/othrys.json:
  {
    "server_url": "http://...",
    "api_key": "...",
    "project_id": "...",
    "agent_id": "...",
    "agent_name": "..."
  }

Hook protocol (same as write_guard.py):
  - Reads JSON from stdin (tool call info)
  - Writes JSON to stdout: {"action": "allow"} or {"action": "block", "reason": "..."}
  - Exit 0 always (non-zero exit blocks all tools unconditionally)
"""

import json
import os
import sys

# Import cache utilities (stdlib only)
try:
    from othrys_cache import read_cache, is_fresh, check_conflict, _find_project_root, _load_config
except ImportError:
    # If running from a different directory, try adding the hooks dir to path
    sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
    from othrys_cache import read_cache, is_fresh, check_conflict, _find_project_root, _load_config


def _extract_file_path(tool_name, tool_input):
    """
    Extract the target file path from a Write or Edit tool call input.
    Returns None if the path cannot be determined.
    """
    if not isinstance(tool_input, dict):
        return None

    # Write tool: {"file_path": "...", "content": "..."}
    # Edit tool (str_replace_based_edit_tool): {"path": "...", ...}
    # MultiEdit: {"path": "...", ...}
    for key in ("file_path", "path"):
        val = tool_input.get(key)
        if val and isinstance(val, str):
            return val

    return None


def _make_path_relative(file_path, project_root):
    """
    Make an absolute file path relative to the project root for cache comparison.
    If already relative, return as-is.
    """
    if os.path.isabs(file_path):
        try:
            return os.path.relpath(file_path, project_root)
        except ValueError:
            return file_path
    return file_path


def main():
    # Read tool call JSON from stdin
    try:
        raw = sys.stdin.read()
        hook_input = json.loads(raw) if raw.strip() else {}
    except (json.JSONDecodeError, OSError):
        hook_input = {}

    tool_name = hook_input.get("tool_name", "")
    tool_input = hook_input.get("tool_input", {})

    # Only intercept Write and Edit tools
    write_tools = {"Write", "Edit", "str_replace_based_edit_tool", "MultiEdit", "create_file", "edit_file"}
    if tool_name not in write_tools:
        print(json.dumps({"action": "allow"}))
        sys.exit(0)

    # Extract the target file path
    file_path = _extract_file_path(tool_name, tool_input)
    if not file_path:
        # Cannot determine path — allow (don't block on ambiguity)
        print(json.dumps({"action": "allow"}))
        sys.exit(0)

    # Find project root and load config
    root = _find_project_root()
    config = _load_config(root)
    my_agent_id = config.get("agent_id", "")

    # Make path relative to project root for comparison
    rel_path = _make_path_relative(file_path, root)

    # Load the local claims cache
    cache = read_cache(root)

    if not cache or not is_fresh(cache):
        # Cache is missing or stale — allow write, log warning
        # Never block on stale data
        msg = (
            f"[othrys/collab_guard] WARNING: claims cache is stale or missing. "
            f"Allowing write to {rel_path}. Run 'python3 othrys_cache.py refresh' to update."
        )
        print(json.dumps({"action": "allow"}), flush=True)
        sys.stderr.write(msg + "\n")
        sys.exit(0)

    # Check for conflicts in the fresh cache
    conflict, claim = check_conflict(cache, rel_path, my_agent_id)

    if conflict:
        agent_name = claim.get("agent_name") or claim.get("agent_id", "another agent")
        claimed_path = claim.get("path", "")
        reason = (
            f"Othrys conflict: {agent_name} holds an exclusive claim on '{claimed_path}', "
            f"which overlaps with '{rel_path}'. "
            f"Release their claim or request a claim for a different path."
        )
        print(json.dumps({"action": "block", "reason": reason}))
        sys.exit(0)

    # No conflict — allow the write
    print(json.dumps({"action": "allow"}))
    sys.exit(0)


if __name__ == "__main__":
    main()
