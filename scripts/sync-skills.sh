#!/usr/bin/env bash
# sync-skills.sh — Mirror skills/confluence into .claude/skills/confluence and .agents/skills/confluence
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SRC="$REPO_ROOT/skills/confluence"

for dest in ".claude/skills/confluence" ".agents/skills/confluence"; do
  target="$REPO_ROOT/$dest"
  echo "Syncing $SRC -> $target"
  rm -rf "$target"
  mkdir -p "$(dirname "$target")"
  cp -r "$SRC" "$target"
done

echo "Done. Run 'git diff' to review."
