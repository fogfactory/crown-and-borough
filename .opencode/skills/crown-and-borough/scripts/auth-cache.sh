#!/usr/bin/env bash

set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
source "$script_dir/context.sh"
cb_require_jq

command="${1:-show}"
shift || true
file="$(cb_auth_file)"

case "$command" in
  save)
    token="${CB_AUTH_TOKEN:-}"
    email="${CB_AUTH_EMAIL:-}"
    uid="${CB_UID:-}"
    api_url="${CB_API_URL:-http://localhost:8080}"
    expires_at="$(date -u -d '+1 hour' +%s)"
    while (($# > 0)); do
      case "$1" in
        --token) token="${2:?--token requires a value}"; shift 2 ;;
        --email) email="${2:?--email requires a value}"; shift 2 ;;
        --uid) uid="${2:?--uid requires a value}"; shift 2 ;;
        --api-url) api_url="${2:?--api-url requires a URL}"; shift 2 ;;
        --expires-at) expires_at="${2:?--expires-at requires epoch seconds}"; shift 2 ;;
        *) printf 'unknown argument: %s\n' "$1" >&2; exit 2 ;;
      esac
    done
    [[ -n "$token" ]] || { printf 'an ID token is required.\n' >&2; exit 2; }
    mkdir -p "$(dirname "$file")"
    umask 077
    jq -n --arg token "$token" --arg email "$email" --arg uid "$uid" \
      --arg api_url "${api_url%/}" --argjson expires_at "$expires_at" \
      '{token:$token,email:$email,uid:$uid,api_url:$api_url,expires_at:$expires_at,saved_at:(now|todateiso8601)}' \
      >"$file"
    chmod 600 "$file"
    printf '%s\n' "$file"
    ;;
  show)
    if [[ ! -f "$file" ]]; then
      printf 'no auth cache at %s\n' "$file" >&2
      exit 1
    fi
    jq 'del(.token)' "$file"
    ;;
  token)
    [[ -f "$file" ]] || { printf 'no auth cache at %s\n' "$file" >&2; exit 1; }
    jq -er '.token'
    <"$file"
    ;;
  clear)
    rm -f "$file"
    ;;
  *)
    printf 'usage: auth-cache.sh save --token TOKEN [--email EMAIL] [--uid UID]\n' >&2
    exit 2
    ;;
esac
