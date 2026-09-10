#!/usr/bin/env bash

set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
source "$script_dir/context.sh"
cb_require_jq
cb_ensure_instance_dir
file="$(cb_game_cache_file)"

command="${1:-show}"
shift || true

case "$command" in
  set)
    game="${CB_GAME_ID:-}"
    player=""
    invite_url=""
    human_slot="${CB_HUMAN_SLOT:-}"
    human_observer="${CB_HUMAN_OBSERVER:-0}"
    while (($# > 0)); do
      case "$1" in
        --id) game="${2:?--id requires a game id}"; shift 2 ;;
        --player) player="${2:?--player requires a player id}"; shift 2 ;;
        --invite-url) invite_url="${2:?--invite-url requires a URL}"; shift 2 ;;
        --human-slot) human_slot="${2:?--human-slot requires a player id}"; shift 2 ;;
        --spectate) human_observer=1; shift ;;
        *) printf 'unknown argument: %s\n' "$1" >&2; exit 2 ;;
      esac
    done
    [[ -n "$game" ]] || { printf 'game id is required.\n' >&2; exit 2; }
    jq -n --arg game_id "$game" --arg player "$player" --arg invite_url "$invite_url" \
      --arg human_slot "$human_slot" --argjson human_observer "$human_observer" \
      '{game_id:$game_id,player:$player,invite_url:$invite_url,human_slot:$human_slot,human_observer:($human_observer == 1),updated_at:(now|todateiso8601)}' \
      >"$file"
    chmod 600 "$file"
    mkdir -p "$CB_RUN_ROOT/$game/$(cb_instance_id)"
    if [[ -f "$(cb_instance_dir)/auth.json" ]]; then
      cp "$(cb_instance_dir)/auth.json" "$CB_RUN_ROOT/$game/$(cb_instance_id)/auth.json"
      chmod 600 "$CB_RUN_ROOT/$game/$(cb_instance_id)/auth.json"
    fi
    printf '%s\n' "$file"
    ;;
  show)
    [[ -f "$file" ]] || { printf 'no game context; run game-cache.sh set --id GAME_ID.\n' >&2; exit 1; }
    jq '.' "$file"
    ;;
  clear)
    rm -f "$file"
    ;;
  *)
    printf 'usage: game-cache.sh set --id GAME_ID [--player PLAYER_ID] [--invite-url URL]\n' >&2
    exit 2
    ;;
esac
