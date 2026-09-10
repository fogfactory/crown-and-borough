#!/usr/bin/env bash

set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
source "$script_dir/context.sh"
cb_require_jq
cb_ensure_bot_dir
game="$(cb_game_id)"

json_file=""
force=0
while (($# > 0)); do
  case "$1" in
    --json) json_file="${2:?--json requires a file or -}"; shift 2 ;;
    --force) force=1; shift ;;
    *) printf 'unknown argument: %s\n' "$1" >&2; exit 2 ;;
  esac
done

if ((force)); then
  response="$("$script_dir/api-call.sh" POST "/api/games/$game/resolve")"
else
  [[ -n "$json_file" ]] || { printf '--json is required unless --force is used.\n' >&2; exit 2; }
  if [[ "$json_file" == "-" ]]; then payload="$(cat)"; else payload="$(cat "$json_file")"; fi
  jq empty <<<"$payload"
  revision="$(jq -r '.revision // empty' "$(cb_bot_dir)/state-cache.json" 2>/dev/null || true)"
  if [[ -n "$revision" ]]; then
    payload="$(jq --argjson revision "$revision" 'if has("revision") then . else . + {revision:$revision} end' <<<"$payload")"
  fi
  response="$("$script_dir/api-call.sh" POST "/api/games/$game/orders" "$payload")"
fi

printf '%s\n' "$response" | jq empty
printf '%s\n' "$response" >"$(cb_bot_dir)/submission-last.json"
chmod 600 "$(cb_bot_dir)/submission-last.json"
printf '%s\n' "$response" | jq '{status,player,submitted,remaining,resolved,forced,revision}'
