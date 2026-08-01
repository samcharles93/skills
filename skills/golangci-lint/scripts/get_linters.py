#!/usr/bin/env python3
"""
Fetch linter metadata from `golangci-lint help linters --json`.
Supports filtering deprecated linters or searching by group/name.
"""

import sys
import json
import subprocess
import argparse

def main():
    parser = argparse.ArgumentParser(description="Query golangci-lint linter catalogue.")
    parser.add_argument("--deprecated", action="store_true", help="List deprecated linters")
    parser.add_argument("--group", choices=["standard", "fast", "all"], help="Filter by group")
    parser.add_argument("--search", help="Search in linter name or description")
    args = parser.parse_args()

    try:
        res = subprocess.run(["golangci-lint", "help", "linters", "--json"], capture_output=True, text=True, check=True)
        linters = json.loads(res.stdout)
    except (subprocess.SubprocessError, json.JSONDecodeError, FileNotFoundError) as e:
        print(f"[ERROR] Failed to query golangci-lint: {e}", file=sys.stderr)
        sys.exit(1)

    filtered = []
    for l in linters:
        if args.deprecated and not l.get("deprecated", False):
            continue
        if not args.deprecated and l.get("deprecated", False):
            continue
        if args.group and args.group not in l.get("groups", []):
            continue
        if args.search:
            s = args.search.lower()
            name_match = s in l.get("name", "").lower()
            desc_match = s in l.get("description", "").lower()
            if not (name_match or desc_match):
                continue
        filtered.append(l)

    print(f"Found {len(filtered)} linters (deprecated={args.deprecated}, group={args.group or 'any'}):")
    for l in filtered:
        dep_str = " [DEPRECATED]" if l.get("deprecated") else ""
        fast_str = " [FAST]" if l.get("fast") else ""
        autofix_str = " [AUTOFIX]" if l.get("autoFix") else ""
        groups_str = f" ({','.join(l.get('groups', []))})"
        print(f"- {l['name']}{dep_str}{fast_str}{autofix_str}{groups_str}: {l['description']}")

if __name__ == "__main__":
    main()
