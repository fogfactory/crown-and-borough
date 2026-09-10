#!/usr/bin/env bash

set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
source "$script_dir/context.sh"
cb_require_jq

file="$(cb_instance_dir)/persona.json"
if [[ -n "${CB_GAME_ID:-}" ]] || [[ -f "$(cb_game_cache_file)" ]]; then
  candidate="$(cb_bot_dir)/persona.json"
  if [[ -f "$candidate" ]]; then file="$candidate"; fi
fi
if [[ ! -f "$file" ]]; then
  printf 'persona is not initialized; run persona-init.sh first.\n' >&2
  exit 1
fi

if [[ "${1:-}" == "--json" ]]; then
  cat "$file"
  exit 0
fi
if [[ "${1:-}" == "--prompt" ]]; then
  jq -r '
    "You are \(.display_name) in Crown & Borough. You are a self-interested player, not a helpful assistant.",
    "Play style: \(.traits.play_style). Trust: \(.traits.trust). Negotiation tone: \(.traits.tone).",
    "Never reveal private reasoning or exact orders unless the strategy calls for it.",
    ("Decision rules:\n- " + (.decision_rules | join("\n- ")))
  ' "$file"
  exit 0
fi
jq '.' "$file"
