#!/usr/bin/env bash
# check-generated-skill.sh — Ensure .claude and .agents skill mirrors match skills/confluence
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SRC="$REPO_ROOT/skills/confluence"

for dest in ".claude/skills/confluence" ".agents/skills/confluence"; do
  target="$REPO_ROOT/$dest"
  if [ ! -d "$target" ]; then
    echo "ERROR: Missing mirror directory $target. Run 'make sync-skills'."
    exit 1
  fi
  if ! diff -r -u "$SRC" "$target" > /dev/null; then
    echo "ERROR: Out of sync: $target differs from $SRC. Run 'make sync-skills'."
    exit 1
  fi
done

echo "OK: Agent skills mirrors are in sync."
