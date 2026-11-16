#!/usr/bin/env bash
# Drop the changelog entries that only repeat another one.
#
# A "Merge pull request #N" commit carries the PR title as its body, so
# release-please parses it as a Conventional Commit and lists the change a
# second time next to the real commit of the PR branch. A line that links to a
# merge commit (several parents) is removed when another line, linking to a
# regular commit, has the same description; a merge entry that stands alone is
# kept, since it is then the only record of the change.
#
# Usage: scripts/dedupe-release-notes.sh <file>   (CHANGELOG.md or a PR body)
# Run it from a clone with the full history.
set -euo pipefail

file=${1:?usage: dedupe-release-notes.sh <file>}

merges=$(mktemp)
out=$(mktemp)
trap 'rm -f "$merges" "$out"' EXIT

grep -oE '/commit/[0-9a-f]{40}' "$file" | cut -d/ -f3 | sort -u |
  xargs -r git rev-list --no-walk --min-parents=2 >"$merges"
[[ -s $merges ]] || exit 0

# Pass 1 records the description of every entry that links to a regular
# commit; pass 2 drops the merge entries whose description is among them.
awk -v mfile="$merges" '
  function hash(l) { return match(l, /\/commit\/[0-9a-f]{40}/) ? substr(l, RSTART + 8, 40) : "" }
  function key(l) { return substr(l, 1, index(l, " ([") - 1) }
  BEGIN { while ((getline h < mfile) > 0) merge[h] = 1 }
  NR == FNR { h = hash($0); if (h != "" && !(h in merge)) seen[key($0)] = 1; next }
  { h = hash($0); if (h in merge && key($0) in seen) next; print }
' "$file" "$file" >"$out"
cat "$out" >"$file"
