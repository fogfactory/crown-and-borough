#!/usr/bin/env bash
# Print the identifier of this agent instance.
#
# Resolution order:
#   1. $OMP_SESSION_ID, $OMP_SESSION or $SESSION_ID when the runtime exposes one
#   2. $CB_INSTANCE_ID when the agent passes it explicitly
#   3. a fresh random id, written to ~/.crown-borough/run/<id>/.id
#
# Every bash tool call is a separate process, so the id printed here does NOT
# survive into the next call by itself. The SKILL therefore instructs the agent
# to record the id when it first appears and to pass it explicitly on every
# later script call:  CB_INSTANCE_ID=<id> ./append-move.sh "…"
set -euo pipefail

id=""
for var in OMP_SESSION_ID OMP_SESSION SESSION_ID CB_INSTANCE_ID; do
  value="${!var:-}"
  if [[ -n "$value" ]]; then
    id="$value"
    break
  fi
done

if [[ -z "$id" ]]; then
  id="cb-$(od -An -N4 -tx1 /dev/urandom | tr -d ' \n')"
  mkdir -p "$HOME/.crown-borough/run/$id"
  printf '%s\n' "$id" >"$HOME/.crown-borough/run/$id/.id"
fi

printf '%s\n' "$id"