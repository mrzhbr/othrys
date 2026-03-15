#!/usr/bin/env python3
"""
othrys_cache.py — Local claims cache manager for Othrys.

Cache file format:
{
  "updated_at": <unix_timestamp>,
  "project_id": "...",
  "claims": [
    {
      "id": "...",
      "agent_id": "...",
      "agent_name": "...",
      "path": "...",
      "claim_type": "exclusive|shared_read",
      "status": "active"
    }
  ]
}

Usage (standalone):
  python3 othrys_cache.py refresh          — fetch claims from server, write cache
  python3 othrys_cache.py check <path>     — check if path has a conflict

Import usage:
  from othrys_cache import read_cache, is_fresh, check_conflict, refresh_cache
"""

import json
import os
import sys
import time
import urllib.request
import urllib.error

# Default TTL: 120 seconds
DEFAULT_TTL = 120

# Path to the Othrys config (relative to project root)
CONFIG_PATH = ".claude/omo/othrys.json"

# Path to the claims cache (relative to project root)
CACHE_PATH = ".claude/omo/.othrys_claims_cache.json"


def _find_project_root():
    """Walk up from cwd to find the project root (dir containing .claude/omo/othrys.json)."""
    cwd = os.path.abspath(os.getcwd())
    candidate = cwd
    for _ in range(10):  # max 10 levels up
        if os.path.isfile(os.path.join(candidate, CONFIG_PATH)):
            return candidate
        parent = os.path.dirname(candidate)
        if parent == candidate:
            break
        candidate = parent
    return cwd  # fallback to cwd


def _load_config(root=None):
    """Load Othrys config from .claude/omo/othrys.json."""
    if root is None:
        root = _find_project_root()
    config_file = os.path.join(root, CONFIG_PATH)
    try:
        with open(config_file) as f:
            return json.load(f)
    except (OSError, json.JSONDecodeError):
        return {}


def read_cache(root=None):
    """Load and return the claims cache dict. Returns empty dict if missing or corrupt."""
    if root is None:
        root = _find_project_root()
    cache_file = os.path.join(root, CACHE_PATH)
    try:
        with open(cache_file) as f:
            data = json.load(f)
            if isinstance(data, dict) and "claims" in data and "updated_at" in data:
                return data
    except (OSError, json.JSONDecodeError):
        pass
    return {}


def is_fresh(cache, ttl_seconds=DEFAULT_TTL):
    """
    Check if the cache updated_at is within TTL.
    Returns False if cache is empty or updated_at is missing/expired.
    """
    updated_at = cache.get("updated_at")
    if updated_at is None:
        return False
    age = time.time() - updated_at
    return age <= ttl_seconds


def check_conflict(cache, file_path, my_agent_id):
    """
    Check if any cached exclusive claim from another agent overlaps file_path.
    Path overlap: one path is a prefix of the other.

    Returns:
      (True, claim_dict)  — if there is a conflict
      (False, None)       — if no conflict
    """
    claims = cache.get("claims", [])
    for claim in claims:
        if claim.get("claim_type") != "exclusive":
            continue
        if claim.get("agent_id") == my_agent_id:
            continue
        if claim.get("status") != "active":
            continue
        claimed_path = claim.get("path", "")
        # Path overlap: either claimed_path is a prefix of file_path, or vice versa
        if _paths_overlap(file_path, claimed_path):
            return True, claim
    return False, None


def _paths_overlap(path_a, path_b):
    """
    Returns True if one path is a prefix of the other.
    Handles trailing slash normalization.
    """
    # Normalize: ensure directory paths end with /
    def norm(p):
        p = p.strip()
        if p and not p.endswith("/") and not os.path.splitext(p)[1]:
            # Looks like a directory (no extension), add trailing slash
            p = p + "/"
        return p

    a = norm(path_a)
    b = norm(path_b)
    return a.startswith(b) or b.startswith(a)


def refresh_cache(othrys_config=None, root=None):
    """
    Fetch all active claims from the Othrys server and write them to the local cache.

    Returns the new cache dict on success, or raises an exception on failure.
    """
    if root is None:
        root = _find_project_root()
    if othrys_config is None:
        othrys_config = _load_config(root)

    server_url = othrys_config.get("server_url", "").rstrip("/")
    api_key = othrys_config.get("api_key", "")
    project_id = othrys_config.get("project_id", "")

    if not server_url or not api_key or not project_id:
        raise ValueError("othrys.json missing required fields: server_url, api_key, project_id")

    url = f"{server_url}/api/v1/projects/{project_id}/claims"
    req = urllib.request.Request(
        url,
        headers={"Authorization": f"Bearer {api_key}", "Accept": "application/json"},
    )

    with urllib.request.urlopen(req, timeout=10) as resp:
        data = json.loads(resp.read().decode())

    # data is a list of claim objects from the server
    claims = []
    for item in (data if isinstance(data, list) else data.get("claims", [])):
        claims.append({
            "id": item.get("id", ""),
            "agent_id": item.get("agent_id", ""),
            "agent_name": item.get("agent_name", ""),
            "path": item.get("path", ""),
            "claim_type": item.get("claim_type", "exclusive"),
            "status": item.get("status", "active"),
        })

    cache = {
        "updated_at": time.time(),
        "project_id": project_id,
        "claims": claims,
    }

    # Ensure cache directory exists
    cache_file = os.path.join(root, CACHE_PATH)
    os.makedirs(os.path.dirname(cache_file), exist_ok=True)
    with open(cache_file, "w") as f:
        json.dump(cache, f, indent=2)

    return cache


def _cmd_refresh():
    """Standalone: refresh cache from server."""
    root = _find_project_root()
    config = _load_config(root)
    if not config:
        print("ERROR: Could not load .claude/omo/othrys.json", file=sys.stderr)
        sys.exit(1)
    try:
        cache = refresh_cache(config, root)
        print(f"Cache refreshed: {len(cache['claims'])} active claim(s) for project {cache['project_id']}")
    except Exception as e:
        print(f"ERROR refreshing cache: {e}", file=sys.stderr)
        sys.exit(1)


def _cmd_check(file_path):
    """Standalone: check if a path has a conflict in the cached claims."""
    root = _find_project_root()
    config = _load_config(root)
    my_agent_id = config.get("agent_id", "")
    cache = read_cache(root)
    if not cache:
        print("No cache found. Run: python3 othrys_cache.py refresh")
        sys.exit(0)
    if not is_fresh(cache):
        print("WARNING: Cache is stale (>120s old). Run: python3 othrys_cache.py refresh")
    conflict, claim = check_conflict(cache, file_path, my_agent_id)
    if conflict:
        print(f"CONFLICT: {file_path} is claimed by agent {claim['agent_name']} ({claim['agent_id']})")
        print(f"  Claim ID: {claim['id']}, Path: {claim['path']}, Type: {claim['claim_type']}")
        sys.exit(1)
    else:
        print(f"OK: No conflict for {file_path}")
        sys.exit(0)


if __name__ == "__main__":
    args = sys.argv[1:]
    if not args or args[0] == "refresh":
        _cmd_refresh()
    elif args[0] == "check" and len(args) >= 2:
        _cmd_check(args[1])
    else:
        print("Usage: python3 othrys_cache.py [refresh | check <path>]", file=sys.stderr)
        sys.exit(1)
