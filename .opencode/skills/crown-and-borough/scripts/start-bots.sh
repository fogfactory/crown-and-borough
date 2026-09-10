#!/usr/bin/env bash

set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
personas="conqueror,diplomat,turtle"
game=""
invite=""
spectate=0
human_slot=""
dry_run=0
enroll_only=0
shared_root="${CB_SHARED_ROOT:-${CB_RUN_ROOT:-$HOME/.crown-borough/run}}"

while (($# > 0)); do
  case "$1" in
    --game-id) game="${2:?--game-id requires a game id}"; shift 2 ;;
    --invite-url) invite="${2:?--invite-url requires a URL}"; shift 2 ;;
    --personas) personas="${2:?--personas requires a comma-separated list}"; shift 2 ;;
    --spectate) spectate=1; shift ;;
    --human-slot) human_slot="${2:?--human-slot requires a player id}"; shift 2 ;;
    --dry-run) dry_run=1; shift ;;
    --enroll-only) enroll_only=1; shift ;;
    --help)
      printf 'usage: start-bots.sh --game-id ID --invite-url URL [--personas a,b,c] [--spectate]\n'
      printf '       start-bots.sh --personas a,b,c --enroll-only\n'
      exit 0
      ;;
    *) printf 'unknown argument: %s\n' "$1" >&2; exit 2 ;;
  esac
done

if ((enroll_only == 0)); then
  [[ -n "$game" && -n "$invite" ]] || { printf '--game-id and --invite-url are required.\n' >&2; exit 2; }
fi
IFS=',' read -r -a persona_list <<<"$personas"
command_template="${CB_HARNESS_COMMAND:-}"
if [[ -z "$command_template" ]]; then
  printf 'CB_HARNESS_COMMAND is unset; printing launch commands only.\n' >&2
fi

declare -a pids=()
index=0
if ((enroll_only)); then
  printf 'Enrollment plan (register these exact addresses before launching bots):\n'
  printf '%-22s %-16s %s\n' 'instance' 'persona' 'email'
fi
for persona in "${persona_list[@]}"; do
  persona="${persona//[[:space:]]/}"
  [[ -n "$persona" ]] || continue
  index=$((index + 1))
  session="cb-${persona}-${index}"
  email="$(CB_INSTANCE_ID="$session" "$script_dir/pick-email.sh" --instance-id "$session")"
  display_name="${email%@*}"
  private_root="${shared_root%/}/private"
  bot_private_root="$private_root/$session"
  if ((enroll_only)); then
    printf '%-22s %-16s %s\n' "$session" "$persona" "$email"
    continue
  fi
  prompt="Use the crown-and-borough skill as persona ${persona}. Join game ${game} with invite ${invite}. Begin with onboarding, then use API-first state and orders."
  observer_env="$spectate"
  command=""
  if [[ -n "$command_template" ]]; then
    command="${command_template//\{session\}/$(printf '%q' "$session")}"
    command="${command//\{persona\}/$(printf '%q' "$persona")}"
    command="${command//\{game\}/$(printf '%q' "$game")}"
    command="${command//\{invite\}/$(printf '%q' "$invite")}"
    command="${command//\{email\}/$(printf '%q' "$email")}"
    command="${command//\{display_name\}/$(printf '%q' "$display_name")}"
    command="${command//\{private_root\}/$(printf '%q' "$bot_private_root")}"
    command="${command//\{prompt\}/$(printf '%q' "$prompt")}"
  fi
  if [[ -z "$command_template" || "$dry_run" == 1 ]]; then
    printf 'OMP_SESSION_ID=%q CB_INSTANCE_ID=%q CB_PERSONA=%q CB_GAME_ID=%q CB_INVITE_URL=%q CB_HUMAN_OBSERVER=%q CB_EMAIL=%q CB_DISPLAY_NAME=%q CB_SHARED_ROOT=%q CB_PRIVATE_ROOT=%q CB_BOT_PRIVATE_ROOT=%q CB_PRIVATE_ROOT_ISOLATED=1 CB_BROWSER_PROFILE_DIR=%q' \
      "$session" "$session" "$persona" "$game" "$invite" "$observer_env" "$email" "$display_name" "$shared_root" "$private_root" "$bot_private_root" "$bot_private_root/browser"
    [[ -n "$human_slot" ]] && printf ' CB_HUMAN_SLOT=%q' "$human_slot"
    printf ' %s\n' "${command:-<set CB_HARNESS_COMMAND>}"
    continue
  fi
  umask 077
  mkdir -p "$bot_private_root/browser"
  (
    export OMP_SESSION_ID="$session" CB_INSTANCE_ID="$session" CB_PERSONA="$persona" CB_GAME_ID="$game" CB_INVITE_URL="$invite" CB_HUMAN_OBSERVER="$observer_env"
    export CB_EMAIL="$email" CB_AUTH_EMAIL="$email" CB_DISPLAY_NAME="$display_name"
    export CB_SHARED_ROOT="$shared_root" CB_RUN_ROOT="$shared_root" CB_PRIVATE_ROOT="$private_root" CB_BOT_PRIVATE_ROOT="$bot_private_root" CB_PRIVATE_ROOT_ISOLATED=1
    export CB_BROWSER_PROFILE_DIR="$bot_private_root/browser"
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
