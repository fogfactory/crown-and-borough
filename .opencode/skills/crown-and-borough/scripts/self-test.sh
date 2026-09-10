#!/usr/bin/env bash

# Offline smoke test for the parallel-session cache boundary.
set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
root="$(mktemp -d "${TMPDIR:-/tmp}/crown-borough-skill.XXXXXX")"
trap 'rm -rf "$root"' EXIT

shared="$root/shared"
private="$shared/private"

run_bot() {
  local id="$1" persona="$2" player="$3"
  local bot_root="$private/$id"
  env \
    CB_SHARED_ROOT="$shared" \
    CB_PRIVATE_ROOT="$private" \
    CB_PRIVATE_ROOT_ISOLATED=1 \
    CB_BOT_PRIVATE_ROOT="$bot_root" \
    CB_GAME_ID=offline-game \
    CB_INSTANCE_ID="$id" \
    "$script_dir/persona-init.sh" --id "$persona" >/dev/null
  env \
    CB_SHARED_ROOT="$shared" \
    CB_PRIVATE_ROOT="$private" \
    CB_PRIVATE_ROOT_ISOLATED=1 \
    CB_BOT_PRIVATE_ROOT="$bot_root" \
    CB_GAME_ID=offline-game \
    CB_INSTANCE_ID="$id" \
    "$script_dir/game-cache.sh" set --id offline-game --player "$player" >/dev/null
}

run_bot cb-alpha-1 conqueror P1
run_bot cb-beta-2 turtle P2

test "$(jq -r '.id' "$private/cb-alpha-1/offline-game/persona.json")" = conqueror
test "$(jq -r '.id' "$private/cb-beta-2/offline-game/persona.json")" = turtle
test "$(jq -r '.player' "$private/cb-alpha-1/current-game.json")" = P1
test "$(jq -r '.player' "$private/cb-beta-2/current-game.json")" = P2
test "$("$script_dir/pick-email.sh" --instance-id cb-alpha-1)" = "$("$script_dir/pick-email.sh" --instance-id cb-alpha-1)"
test "$("$script_dir/pick-email.sh" --instance-id cb-alpha-1)" != "$("$script_dir/pick-email.sh" --instance-id cb-beta-2)"

printf 'parallel cache isolation passed\n'
