#!/usr/bin/env bash

set -euo pipefail

CB_SHARED_ROOT="${CB_SHARED_ROOT:-${CB_RUN_ROOT:-$HOME/.crown-borough/run}}"
CB_RUN_ROOT="$CB_SHARED_ROOT"
CB_PRIVATE_ROOT="${CB_PRIVATE_ROOT:-$CB_SHARED_ROOT}"

cb_require_jq() {
  command -v jq >/dev/null 2>&1 || {
    printf 'jq is required by the Crown & Borough skill scripts.\n' >&2
    return 1
  }
}

cb_instance_id() {
  local id="${CB_INSTANCE_ID:-${OMP_SESSION_ID:-${OMP_SESSION:-${SESSION_ID:-}}}}"
  if [[ -z "$id" ]]; then
    printf 'CB_INSTANCE_ID or OMP_SESSION_ID is required for this operation.\n' >&2
    return 1
  fi
  if [[ ! "$id" =~ ^[A-Za-z0-9._-]+$ ]]; then
    printf 'invalid instance id: %s\n' "$id" >&2
    return 1
  fi
  printf '%s\n' "$id"
}

cb_instance_dir() {
  if [[ -n "${CB_BOT_PRIVATE_ROOT:-}" ]]; then
    printf '%s\n' "$CB_BOT_PRIVATE_ROOT"
    return
  fi
  printf '%s/%s\n' "$CB_PRIVATE_ROOT" "$(cb_instance_id)"
}

cb_game_id() {
  if [[ -n "${CB_GAME_ID:-}" ]]; then
    printf '%s\n' "$CB_GAME_ID"
    return
  fi
  local file="$(cb_instance_dir)/current-game.json"
  if [[ -f "$file" ]]; then
    cb_require_jq
    jq -er '.game_id // empty' "$file"
    return
  fi
  printf 'CB_GAME_ID is required; use game-cache.sh set --id GAME_ID.\n' >&2
  return 1
}

cb_game_dir() {
  local game
  game="$(cb_game_id)"
  if [[ ! "$game" =~ ^[A-Za-z0-9._:-]+$ ]]; then
    printf 'invalid game id: %s\n' "$game" >&2
    return 1
  fi
  printf '%s/%s\n' "$CB_SHARED_ROOT" "$game"
}

cb_bot_dir() {
  if [[ -n "${CB_BOT_PRIVATE_ROOT:-}" ]]; then
    printf '%s/%s\n' "$CB_BOT_PRIVATE_ROOT" "$(cb_game_id)"
    return
  fi
  local id
  id="$(cb_instance_id)"
  if [[ "${CB_PRIVATE_ROOT_ISOLATED:-0}" == "1" ]]; then
    printf '%s/%s/%s\n' "$CB_PRIVATE_ROOT" "$id" "$(cb_game_id)"
  else
    printf '%s/%s/%s\n' "$CB_SHARED_ROOT" "$(cb_game_id)" "$id"
  fi
}

cb_auth_file() {
  if [[ -n "${CB_GAME_ID:-}" ]] || [[ -f "$(cb_instance_dir)/current-game.json" ]]; then
    printf '%s/auth.json\n' "$(cb_bot_dir)"
  else
    printf '%s/auth.json\n' "$(cb_instance_dir)"
  fi
}

cb_game_cache_file() {
  printf '%s/current-game.json\n' "$(cb_instance_dir)"
}

cb_ensure_instance_dir() {
  umask 077
  mkdir -p "$(cb_instance_dir)"
}

cb_ensure_bot_dir() {
  umask 077
  mkdir -p "$(cb_bot_dir)"
}
