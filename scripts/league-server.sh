#!/usr/bin/env bash
# Starts (or stops) the local online stack a bot league plays on: the Firebase
# Auth and Firestore emulators (docker compose) and the online server, in the
# background. The chat only exists in this mode, not in hotseat or
# ONLINE_DEV_MODE.
#
#   scripts/league-server.sh [start|stop|status]
#
# PORT (default 8080) is the server port. The log is tmp/league-server.log and
# the pid tmp/league-server.pid. Sign in as admin@mail.com (the observer host
# of botleague.example.yaml); sign-in links are listed at
# http://127.0.0.1:9099/emulator/v1/projects/demo-crown-and-borough/oobCodes
set -euo pipefail

cd "$(dirname "$0")/.."

action="${1:-start}"
port="${PORT:-8080}"
firestore_port="${FIRESTORE_PORT:-8081}"
auth_port="${AUTH_PORT:-9099}"
project="${COMPOSE_PROJECT_NAME:-crown-and-borough}"
pidfile=tmp/league-server.pid
mkdir -p tmp bin

stop_server() {
	if [ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" 2>/dev/null; then
		kill "$(cat "$pidfile")"
		sleep 1
	fi
	rm -f "$pidfile"
}

case "$action" in
stop)
	stop_server
	echo "league server stopped (the emulators keep running: docker compose -p $project down to stop them)"
	exit 0
	;;
status)
	if curl -fs "http://localhost:$port/healthz/ready" >/dev/null 2>&1; then
		echo "league server ready on http://localhost:$port"
	else
		echo "league server not running on port $port" >&2
		exit 1
	fi
	exit 0
	;;
start) ;;
*)
	sed -n 2,13p "$0"
	exit 2
	;;
esac

stop_server
go build -o bin/server-league ./cmd/server
SERVER_PORT="$port" FIRESTORE_PORT="$firestore_port" AUTH_PORT="$auth_port" \
	docker compose -p "$project" up -d --wait firestore auth >/dev/null

ASSETS_DIR=assets \
	FIRESTORE_EMULATOR_HOST="127.0.0.1:$firestore_port" \
	FIREBASE_AUTH_EMULATOR_HOST="127.0.0.1:$auth_port" \
	FIREBASE_PROJECT_ID=demo-crown-and-borough \
	GOOGLE_CLOUD_PROJECT=crown-and-borough-local \
	ALLOWED_CREATOR_EMAILS="${ALLOWED_CREATOR_EMAILS:-admin@mail.com}" \
	ONLINE_DEV_MODE=false \
	PUBLIC_APP_URL="http://localhost:$port" \
	PORT="$port" \
	nohup ./bin/server-league >tmp/league-server.log 2>&1 &
echo $! >"$pidfile"

for _ in $(seq 1 60); do
	if ! kill -0 "$(cat "$pidfile")" 2>/dev/null; then
		echo "the server exited (port $port busy?); see tmp/league-server.log" >&2
		tail -5 tmp/league-server.log >&2
		exit 1
	fi
	if curl -fs "http://localhost:$port/healthz/ready" >/dev/null 2>&1; then
		echo "league server ready on http://localhost:$port (log: tmp/league-server.log)"
		exit 0
	fi
	sleep 0.5
done
echo "the server did not become ready; see tmp/league-server.log" >&2
exit 1
