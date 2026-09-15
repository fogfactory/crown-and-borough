#!/usr/bin/env bash

set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
source "$script_dir/context.sh"
cb_require_jq
cb_ensure_bot_dir

game="$(cb_game_id)"
cache="$(cb_bot_dir)/state-cache.json"
previous="$(mktemp)"
trap 'rm -f "$previous"' EXIT
if [[ -f "$cache" ]]; then cp "$cache" "$previous"; else printf '{}' >"$previous"; fi

response="$(CB_API_URL="${CB_API_URL:-${PUBLIC_APP_URL:-http://localhost:8080}}" "$script_dir/api-call.sh" GET "/api/games/$game/state")"
printf '%s\n' "$response" | jq empty
printf '%s\n' "$response" >"$cache"
chmod 600 "$cache"

revision="$(jq -r '.revision // 0' "$cache")"
turn="$(jq -r '.turn // 0' "$cache")"
season="$(jq -r '.season // "unknown"' "$cache")"
player_id="$(jq -r '.player // empty' "$(cb_game_cache_file)" 2>/dev/null || true)"
owned="$(jq -r --arg id "$player_id" '[.territories[]? | select(.owner == $id) | .id] | join(", ")' "$cache")"
armies="$(jq -r --arg id "$player_id" '[.territories[]? | select(.army.owner == $id) | ("\(.id):\(.army.size)")] | join(", ")' "$cache")"

printf 'Turn %s · %s · revision %s\n' "$turn" "$season" "$revision"

if [[ "${1:-}" == "--diff" && -s "$previous" && "$(jq -r '.revision // 0' "$previous")" != "0" ]]; then
  old_turn="$(jq -r '.turn // 0' "$previous")"
  old_revision="$(jq -r '.revision // 0' "$previous")"
  if [[ "$old_turn" != "$turn" || "$old_revision" != "$revision" ]]; then
    printf 'Changed since revision %s: turn/revision advanced from %s/%s.\n' "$old_revision" "$old_turn" "$old_revision"
  else
    printf 'No newer server revision.\n'
  fi
fi
