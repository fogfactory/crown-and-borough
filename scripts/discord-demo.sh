#!/usr/bin/env bash
# Online demo of the Discord webhook: starts the local online stack (Firebase
# emulators and server, see scripts/league-server.sh), creates a two-house game,
# lets its host (the observer, admin@mail.com) configure a webhook with an
# @here template, then triggers a test, a public room message and a new turn.
#
#   scripts/discord-demo.sh [--skip-web]
#   scripts/discord-demo.sh stop
#
# The server only accepts https://discord.com/api/webhooks/... URLs and posts
# to Discord. So that the demo needs no real webhook, the server is built with
# -tags demo and DISCORD_DEMO_RECEIVER: that build (never a production one)
# hands each announcement to a local receiver instead of discord.com, with the
# body Discord would get (content and allowed_mentions). The URL validation is
# the production one: the demo webhook URL is a well-formed fake.
#
# PORT (default 8080) is the server port, DISCORD_RECEIVER_PORT (default 8099)
# the receiver's; it prints to tmp/discord-receiver.log. At the end the script
# prints a sign-in link for the host, who finds the "Discord" button in the
# game menu. Stop with: scripts/discord-demo.sh stop, then
# docker compose -p crown-and-borough down.
set -euo pipefail

cd "$(dirname "$0")/.."

skip_web=0
stop=0
for arg in "$@"; do
	case "$arg" in
	--skip-web) skip_web=1 ;;
	stop) stop=1 ;;
	-h | --help)
		sed -n '2,21p' "$0"
		exit 0
		;;
	*)
		echo "unknown argument $arg" >&2
		exit 2
		;;
	esac
done

port="${PORT:-8080}"
auth_port="${AUTH_PORT:-9099}"
receiver_port="${DISCORD_RECEIVER_PORT:-8099}"
base="http://localhost:$port"
auth="http://127.0.0.1:$auth_port"
project=demo-crown-and-borough
key=fake-api-key
pidfile=tmp/discord-receiver.pid
log=tmp/discord-receiver.log
mkdir -p tmp

stop_receiver() {
	if [ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" 2>/dev/null; then
		kill "$(cat "$pidfile")"
	fi
	rm -f "$pidfile"
}

if [ "$stop" = 1 ]; then
	stop_receiver
	scripts/league-server.sh stop
	exit 0
fi

stop_receiver
: >"$log"
# Stands for Discord: prints one line per POST and answers 204.
python3 -u -c '
import json, sys
from http.server import BaseHTTPRequestHandler, HTTPServer

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        body = self.rfile.read(int(self.headers.get("Content-Length", 0)))
        try:
            body = json.dumps(json.loads(body), ensure_ascii=False)
        except ValueError:
            pass
        print("POST", body, flush=True)
        self.send_response(204)
        self.end_headers()

    def log_message(self, *args):
        pass

HTTPServer(("127.0.0.1", int(sys.argv[1])), Handler).serve_forever()
' "$receiver_port" >>"$log" 2>&1 &
echo $! >"$pidfile"

if [ "$skip_web" = 0 ]; then
	make web-build >/dev/null
fi
SERVER_TAGS=demo DISCORD_DEMO_RECEIVER="http://127.0.0.1:$receiver_port" \
	PORT="$port" AUTH_PORT="$auth_port" scripts/league-server.sh start

# request_link EMAIL CONTINUE_URL: asks the emulator for a sign-in link and
# prints it.
request_link() {
	curl -fsS -X POST "$auth/identitytoolkit.googleapis.com/v1/accounts:sendOobCode?key=$key" \
		-H 'Content-Type: application/json' \
		-d "$(jq -n --arg email "$1" --arg url "$2" \
			'{requestType: "EMAIL_SIGNIN", email: $email, continueUrl: $url, canHandleCodeInApp: true}')" >/dev/null
	curl -fsS "$auth/emulator/v1/projects/$project/oobCodes" |
		jq -r --arg email "$1" '[.oobCodes[] | select(.email == $email)] | last'
}

# sign_in EMAIL: prints an ID token.
sign_in() {
	local code
	code=$(request_link "$1" "$base/finish" | jq -r .oobCode)
	curl -fsS -X POST "$auth/identitytoolkit.googleapis.com/v1/accounts:signInWithEmailLink?key=$key" \
		-H 'Content-Type: application/json' \
		-d "$(jq -n --arg email "$1" --arg code "$code" '{email: $email, oobCode: $code}')" |
		jq -r .idToken
}

# api TOKEN METHOD PATH [BODY]
api() {
	curl -fsS -X "$2" "$base$3" -H "Authorization: Bearer $1" \
		-H 'Content-Type: application/json' ${4:+-d "$4"}
}

host=$(sign_in admin@mail.com)
alice=$(sign_in alice@mail.com)
bruno=$(sign_in bruno@mail.com)
api "$host" PUT /api/auth/me '{"displayName":"Observateur"}' >/dev/null
api "$alice" PUT /api/auth/me '{"displayName":"Alice"}' >/dev/null
api "$bruno" PUT /api/auth/me '{"displayName":"Bruno"}' >/dev/null

game=$(api "$host" POST /api/games \
	'{"name":"Démo Discord","seed":"discord-demo","years":4,"spectate":true,"players":2}')
game_id=$(jq -r .id <<<"$game")
invite=$(jq -r .inviteCode <<<"$game")
api "$alice" POST "/api/games/$game_id/join" "{\"inviteCode\":\"$invite\"}" >/dev/null
api "$bruno" POST "/api/games/$game_id/join" "{\"inviteCode\":\"$invite\"}" >/dev/null

# The host's webhook: a well-formed fake URL, and an @here on the new turn.
api "$host" PUT "/api/games/$game_id/discord-webhook" "$(jq -n \
	'{url: "https://discord.com/api/webhooks/123456789012345678/demo-token", language: "fr",
	  templates: {newTurn: "@here Tour {turn} ({season}) de {game} : {url}", publicMessage: ""}}')" >/dev/null

echo "test message:"
api "$host" POST "/api/games/$game_id/discord-webhook/test" >/dev/null
# One public room announcement per game every few minutes.
api "$bruno" POST "/api/games/$game_id/messages" \
	'{"channel":"public","body":"Bonjour à tous."}' >/dev/null
echo "new turn (forced resolution):"
api "$host" POST "/api/games/$game_id/resolve" '{}' >/dev/null || echo "  (resolution refused)"
sleep 2

echo
echo "what the webhook received (the same lines are in $log):"
cat "$log"

link=$(request_link admin@mail.com "$base/finish?redirect=/games/$game_id" | jq -r .oobLink | sed "s#^http://[^/]*/#$auth/#")
cat <<EOF

discord demo ready: game "Démo Discord" ($game_id) on $base
  The host (admin@mail.com, observer) opens the game menu, then "Discord", to
  read the settings back (the URL is never shown), edit the messages with their
  live preview, send a test or remove the webhook. Alice and Bruno (alice@mail.com,
  bruno@mail.com) see the notice that a webhook is active. Sign in as the host
  with this link (enter admin@mail.com if asked):
  $link
  Watch the announcements: tail -f $log
EOF
