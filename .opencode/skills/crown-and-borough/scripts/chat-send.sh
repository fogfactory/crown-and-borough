#!/usr/bin/env bash

set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
source "$script_dir/context.sh"

channel=""
peer=""
alliance=""
message=""
while (($# > 0)); do
  case "$1" in
    --channel) channel="${2:?--channel requires table, dm, or alliance}"; shift 2 ;;
    --to) peer="${2:?--to requires an instance id}"; shift 2 ;;
    --name) alliance="${2:?--name requires an alliance name}"; shift 2 ;;
    --help)
      printf 'usage: chat-send.sh --channel table "message"\n'
      printf '       chat-send.sh --channel dm --to PEER "message"\n'
      printf '       chat-send.sh --channel alliance --name NAME [--to PEER] "message"\n'
      exit 0
      ;;
    --*) printf 'unknown argument: %s\n' "$1" >&2; exit 2 ;;
    *) message="$1"; shift; [[ $# -eq 0 ]] || { printf 'message must be one argument.\n' >&2; exit 2; } ;;
  esac
done

me="$(cb_instance_id)"
game="$(cb_game_id)"
[[ -n "$channel" && -n "$message" ]] || { printf 'a channel and message are required.\n' >&2; exit 2; }
[[ "$message" != *$'\n'* && "$message" != *$'\r'* ]] || { printf 'messages must be one line.\n' >&2; exit 2; }
[[ ${#message} -le 500 ]] || { printf 'message is too long (maximum 500 characters).\n' >&2; exit 2; }
[[ "$me" != "$peer" ]] || { printf 'cannot send a DM to yourself.\n' >&2; exit 2; }

case "$channel" in
  table)
    [[ -z "$peer" && -z "$alliance" ]] || { printf 'table messages do not use --to or --name.\n' >&2; exit 2; }
    channel_key="table"
    recipient="table"
    ;;
  dm)
    [[ -n "$peer" && -z "$alliance" ]] || { printf 'DMs require exactly one --to peer.\n' >&2; exit 2; }
    if [[ "$me" < "$peer" ]]; then channel_key="dm:$me+$peer"; else channel_key="dm:$peer+$me"; fi
    recipient="$peer"
    ;;
  alliance)
    [[ "$alliance" =~ ^[A-Za-z0-9._-]+$ ]] || { printf 'invalid alliance name.\n' >&2; exit 2; }
    channel_key="alliance:$alliance"
    recipient="${peer:-alliance}"
    ;;
  *) printf 'channel must be table, dm, or alliance.\n' >&2; exit 2 ;;
esac

log="$(cb_game_dir)/chat.log"
umask 077
mkdir -p "$(cb_game_dir)"
timestamp="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
line="[$timestamp] [ch=$channel_key] [from=$me] [to=$recipient] $message"
exec 9>>"$log"
flock -x 9
printf '%s\n' "$line" >&9
line_number="$(wc -l <"$log")"
flock -u 9
printf 'sent %s line %s\n' "$channel_key" "$line_number"
