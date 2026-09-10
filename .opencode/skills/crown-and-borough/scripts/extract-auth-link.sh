#!/usr/bin/env bash
# Wait for the Firebase Auth emulator to print an email sign-in link and
# print the first matching URL to stdout.
#
# The auth container (auth-emulator.Dockerfile, CI=true) logs something like:
#   i  auth: To verify the sign-in link, open the following URL in your browser:
#   http://127.0.0.1:9099/emulator/action?mode=signIn&oobCode=...&apiKey=...
#
# usage: extract-auth-link.sh [timeout_seconds] [compose_file]
set -euo pipefail

timeout_s="${1:-180}"
compose_file="${2:-docker-compose.yml}"
url_pattern='https?://[^ <>"'"'"']*emulator/action[^ <>"'"'"']*'

docker compose -f "$compose_file" logs -f --no-color auth 2>/dev/null |
  timeout "$timeout_s" grep -Eom 1 "$url_pattern" || {
    echo "No sign-in link appeared in the auth logs within ${timeout_s}s." >&2
    exit 1
  }