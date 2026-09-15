#!/usr/bin/env bash

set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
source "$script_dir/context.sh"

channel=""
peer=""
alliance=""
all_dms=0
since=0
watch=0
include_self=0
while (($# > 0)); do
  case "$1" in
    --channel) channel="${2:?--channel requires table or alliance}"; shift 2 ;;
    --with) peer="${2:?--with requires a peer id}"; shift 2 ;;
    --all-dms) all_dms=1; shift ;;
    --name) alliance="${2:?--name requires an alliance name}"; shift 2 ;;
    --since) since="${2:?--since requires a line number}"; shift 2 ;;
    --watch)
      if (($# > 1)) && [[ "$2" != --* ]]; then watch="$2"; shift 2; else watch=300; shift; fi
      ;;
    --include-self) include_self=1; shift ;;
    --help)
      printf 'usage: chat-poll.sh --with PEER [--since LINE]\n'
      printf '       chat-poll.sh --all-dms [--since LINE]\n'
      printf '       chat-poll.sh --channel table|alliance --name NAME [--watch SECONDS]\n'
      exit 0
      ;;
    *) printf 'unknown argument: %s\n' "$1" >&2; exit 2 ;;
  esac
done

me="$(cb_instance_id)"
log="$(cb_game_dir)/chat.log"
filter=""
if ((all_dms)); then
  [[ -z "$peer" && -z "$channel" && -z "$alliance" ]] || {
    printf '--all-dms cannot be combined with another channel filter.\n' >&2
    exit 2
  }
  filter="__all_dms__"
elif [[ -n "$peer" ]]; then
  [[ "$peer" != "$me" ]] || { printf 'cannot poll a DM with yourself.\n' >&2; exit 2; }
  if [[ "$me" < "$peer" ]]; then filter="dm:$me+$peer"; else filter="dm:$peer+$me"; fi
elif [[ "$channel" == "table" ]]; then
  [[ -z "$alliance" ]] || { printf 'table polling does not use --name.\n' >&2; exit 2; }
  filter="table"
elif [[ "$channel" == "alliance" ]]; then
  [[ "$alliance" =~ ^[A-Za-z0-9._-]+$ ]] || { printf 'alliance polling requires a valid --name.\n' >&2; exit 2; }
  filter="alliance:$alliance"
else
  printf 'choose --with PEER or --channel table|alliance.\n' >&2
  exit 2
fi

show_from() {
  local start="$1"
  [[ -f "$log" ]] || return 1
  local line_number=0 line matched
  while IFS= read -r line; do
    line_number=$((line_number + 1))
    ((line_number > start)) || continue
    if [[ "$filter" == "__all_dms__" ]]; then
      case "$line" in
        *"[ch=dm:$me+"*|*"[ch=dm:*+$me]"*) matched=1 ;;
        *) matched=0 ;;
      esac
    else
      case "$line" in *"[ch=$filter]"*) matched=1 ;; *) matched=0 ;; esac
    fi
    ((matched == 1)) || continue
    if ((include_self == 0)) && [[ "$line" == *"[from=$me]"* ]]; then continue; fi
    printf '[%s] %s\n' "$line_number" "$line"
  done <"$log"
}

if [[ "$watch" == "0" ]]; then
  show_from "$since" || true
  exit 0
fi

deadline=$((SECONDS + watch))
marker="$since"
show_from "$marker" || true
if [[ -f "$log" ]]; then marker="$(wc -l <"$log")"; fi
while ((SECONDS < deadline)); do
  sleep "${CB_POLL_INTERVAL:-2}"
  if output="$(show_from "$marker" 2>/dev/null)" && [[ -n "$output" ]]; then
    printf '%s\n' "$output"
    exit 0
  fi
  if [[ -f "$log" ]]; then marker="$(wc -l <"$log")"; fi
done
printf '[watch] no matching messages within %ss\n' "$watch" >&2
exit 1
