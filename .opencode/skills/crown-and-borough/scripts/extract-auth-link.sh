#!/usr/bin/env bash
# Extract one Firebase Auth emulator email-link URL.
#
# With --email, query the emulator's per-project out-of-band-code endpoint so
# parallel bot enrollments cannot consume one another's first log line.
# Without --email, retain the legacy first-link-from-logs behavior.
set -euo pipefail

timeout_s="${CB_AUTH_LINK_TIMEOUT:-180}"
compose_file="${CB_COMPOSE_FILE:-docker-compose.yml}"
email=""
auth_url="${CB_AUTH_EMULATOR_URL:-http://127.0.0.1:9099}"
project="${CB_FIREBASE_PROJECT_ID:-demo-crown-and-borough}"

while (($# > 0)); do
  case "$1" in
    --email) email="${2:?--email requires an email address}"; shift 2 ;;
    --timeout) timeout_s="${2:?--timeout requires seconds}"; shift 2 ;;
    --auth-url) auth_url="${2:?--auth-url requires a URL}"; shift 2 ;;
    --project) project="${2:?--project requires an id}"; shift 2 ;;
    --compose-file) compose_file="${2:?--compose-file requires a path}"; shift 2 ;;
    --help)
      printf 'usage: extract-auth-link.sh --email EMAIL [--timeout SECONDS]\n'
      printf '       extract-auth-link.sh [--timeout SECONDS] [--compose-file FILE]\n'
      exit 0
      ;;
    [0-9]*) timeout_s="$1"; shift ;;
    *) compose_file="$1"; shift ;;
  esac
done

if [[ -n "$email" ]]; then
  command -v jq >/dev/null 2>&1 || {
    printf 'jq is required for email-specific auth-link lookup.\n' >&2
    exit 1
  }
  deadline=$((SECONDS + timeout_s))
  endpoint="${auth_url%/}/emulator/v1/projects/${project}/oobCodes"
  while ((SECONDS < deadline)); do
    response="$(curl -fsS "$endpoint" 2>/dev/null || true)"
    link="$(jq -r --arg email "$email" '
      [.oobCodes[]? | select(.email == $email)] | last |
      (.oobLink //
        (if .oobCode and .apiKey then
          "http://127.0.0.1:9099/emulator/action?mode=signIn&oobCode=" + .oobCode + "&apiKey=" + .apiKey
         else empty end)) // empty
    ' <<<"$response" 2>/dev/null || true)"
    if [[ -n "$link" ]]; then
      printf '%s\n' "$link"
      exit 0
    fi
    sleep 1
  done
  printf 'No sign-in link for %s appeared in the Auth emulator within %ss.\n' "$email" "$timeout_s" >&2
  exit 1
fi

url_pattern='https?://[^ <>"'"'']*emulator/action[^ <>"'"'']*'
docker compose -f "$compose_file" logs -f --no-color auth 2>/dev/null |
  timeout "$timeout_s" grep -Eom 1 "$url_pattern" || {
    echo "No sign-in link appeared in the auth logs within ${timeout_s}s." >&2
    exit 1
  }
