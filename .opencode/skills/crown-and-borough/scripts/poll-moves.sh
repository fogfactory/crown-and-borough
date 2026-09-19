#!/usr/bin/env bash
# Print negotiation lines written by OTHER instances (never your own).
#
#   poll-moves.sh            → snapshot: last lines of every other moves.txt
#   poll-moves.sh --watch N  → print now, then block up to N seconds and exit
#                              as soon as new lines appear anywhere.
#
# The instance id must come from $CB_INSTANCE_ID (see SKILL.md).
# Polling interval: $CB_POLL_INTERVAL (default 2s).
# Lines shown per instance: $CB_POLL_LINES (default 5).
set -euo pipefail

root="$HOME/.crown-borough/run"
id="$(bash "$(dirname "$0")/instance-id.sh")"

show() {
  for f in "$root"/*/moves.txt; do
    [[ -e "$f" ]] || continue
    other="$(basename "$(dirname "$f")")"
    [[ "$other" == "$id" ]] && continue
    echo "== $other =="
    tail -n "${CB_POLL_LINES:-5}" "$f"
  done
}

newest_mtime() {
  local newest=""
  for f in "$root"/*/moves.txt; do
    [[ -e "$f" ]] || continue
    [[ "$(basename "$(dirname "$f")")" == "$id" ]] && continue
    m="$(stat -c %Y "$f")"
    if [[ -z "$newest" || "$m" -gt "$newest" ]]; then newest="$m"; fi
  done
  printf '%s' "$newest"
}

show

if [[ "${1:-}" == "--watch" ]]; then
  timeout_s="${2:-300}"
  deadline=$(( $(date +%s) + timeout_s ))
  last="$(newest_mtime)"
  while (( $(date +%s) < deadline )); do
    sleep "${CB_POLL_INTERVAL:-2}"
    latest="$(newest_mtime)"
    if [[ -n "$latest" && "$latest" != "$last" ]]; then
      show
      exit 0
    fi
  done
  echo "[watch] no new moves within ${timeout_s}s" >&2
  exit 1
fi