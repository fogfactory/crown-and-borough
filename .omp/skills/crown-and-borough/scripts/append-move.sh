#!/usr/bin/env bash
# Append one human-readable negotiation line to this instance's moves file.
# Usage: append-move.sh "I move north."
# The instance id must come from $CB_INSTANCE_ID (see SKILL.md); a missing id
# falls back to a fresh random one, which WILL NOT match your identity.
set -euo pipefail

id="$(bash "$(dirname "$0")/instance-id.sh")"
dir="$HOME/.crown-borough/run/$id"
mkdir -p "$dir"
printf '[%s] %s\n' "$(date +%H:%M:%S)" "$*" >>"$dir/moves.txt"