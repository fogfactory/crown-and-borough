#!/usr/bin/env bash

set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
source "$script_dir/context.sh"

method="${1:-}"
path="${2:-}"
body="${3:-}"
[[ -n "$method" && -n "$path" ]] || {
  printf 'usage: api-call.sh METHOD PATH [JSON| -]\n' >&2
  exit 2
}

base_url="${CB_API_URL:-${PUBLIC_APP_URL:-http://localhost:8080}}"
if [[ "$path" == http://* || "$path" == https://* ]]; then
  url="$path"
else
  url="${base_url%/}/${path#/}"
fi

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
args=(-sS -o "$tmp" -w '%{http_code}' -X "$method" -H 'Accept: application/json')
if [[ -n "$body" ]]; then
  if [[ "$body" == "-" ]]; then
    body="$(cat)"
  fi
  args+=(-H 'Content-Type: application/json' --data "$body")
fi
auth_file="$(cb_auth_file)"
if [[ -f "$auth_file" ]]; then
  token="$(jq -er '.token' "$auth_file")"
  args+=(-H "Authorization: Bearer $token")
fi

status="$(curl "${args[@]}" "$url")"
cat "$tmp"
printf '\n'
if [[ "$status" =~ ^[0-9]+$ ]] && ((status >= 400)); then
  if ((status == 401)); then
    printf 'API returned 401. Re-authenticate this instance and refresh auth-cache.json.\n' >&2
  fi
  exit 22
fi
