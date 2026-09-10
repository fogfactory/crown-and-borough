#!/usr/bin/env bash

set -euo pipefail

personas="conqueror,diplomat,turtle"
game=""
invite=""
spectate=0
human_slot=""
dry_run=0

while (($# > 0)); do
  case "$1" in
    --game-id) game="${2:?--game-id requires a game id}"; shift 2 ;;
    --invite-url) invite="${2:?--invite-url requires a URL}"; shift 2 ;;
    --personas) personas="${2:?--personas requires a comma-separated list}"; shift 2 ;;
    --spectate) spectate=1; shift ;;
    --human-slot) human_slot="${2:?--human-slot requires a player id}"; shift 2 ;;
    --dry-run) dry_run=1; shift ;;
    --help)
      printf 'usage: start-bots.sh --game-id ID --invite-url URL [--personas a,b,c] [--spectate]\n'
      exit 0
      ;;
    *) printf 'unknown argument: %s\n' "$1" >&2; exit 2 ;;
  esac
done

[[ -n "$game" && -n "$invite" ]] || { printf '--game-id and --invite-url are required.\n' >&2; exit 2; }
IFS=',' read -r -a persona_list <<<"$personas"
command_template="${CB_HARNESS_COMMAND:-}"
if [[ -z "$command_template" ]]; then
  printf 'CB_HARNESS_COMMAND is unset; printing launch commands only.\n' >&2
fi

declare -a pids=()
index=0
for persona in "${persona_list[@]}"; do
  persona="${persona//[[:space:]]/}"
  [[ -n "$persona" ]] || continue
  index=$((index + 1))
  session="cb-${persona}-${index}"
  prompt="Use the crown-and-borough skill as persona ${persona}. Join game ${game} with invite ${invite}. Begin with onboarding, then use API-first state and orders."
  observer_env="$spectate"
  command=""
  if [[ -n "$command_template" ]]; then
    command="${command_template//\{session\}/$(printf '%q' "$session")}"
    command="${command//\{persona\}/$(printf '%q' "$persona")}"
    command="${command//\{game\}/$(printf '%q' "$game")}"
    command="${command//\{invite\}/$(printf '%q' "$invite")}"
    command="${command//\{prompt\}/$(printf '%q' "$prompt")}"
  fi
  if [[ -z "$command_template" || "$dry_run" == 1 ]]; then
    printf 'OMP_SESSION_ID=%q CB_INSTANCE_ID=%q CB_PERSONA=%q CB_GAME_ID=%q CB_INVITE_URL=%q CB_HUMAN_OBSERVER=%q' "$session" "$session" "$persona" "$game" "$invite" "$observer_env"
    [[ -n "$human_slot" ]] && printf ' CB_HUMAN_SLOT=%q' "$human_slot"
    printf ' %s\n' "${command:-<set CB_HARNESS_COMMAND>}"
    continue
  fi
  (
    export OMP_SESSION_ID="$session" CB_INSTANCE_ID="$session" CB_PERSONA="$persona" CB_GAME_ID="$game" CB_INVITE_URL="$invite" CB_HUMAN_OBSERVER="$observer_env"
    [[ -z "$human_slot" ]] || export CB_HUMAN_SLOT="$human_slot"
    bash -lc "$command"
  ) &
  pids+=("$!")
done

if ((${#pids[@]} > 0)); then
  status=0
  for pid in "${pids[@]}"; do
    wait "$pid" || status=$?
  done
  exit "$status"
fi
