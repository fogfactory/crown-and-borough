#!/usr/bin/env bash
# Starts the hotseat server on a forged game state, to try a feature without
# playing the turns that lead to it.
#
#   scripts/demo.sh [scenario] [--fg] [--skip-web]
#
# Scenarios live in cmd/server/demo_scenarios.go (built with -tags demo, so the
# production binary never carries them): `winter` (default) and `action`.
# Without --fg the server runs in the background (log in tmp/demo-server.log,
# pid in tmp/demo-server.pid) and a previous demo server is stopped first.
set -euo pipefail

cd "$(dirname "$0")/.."

scenario="winter"
foreground=0
skip_web=0
for arg in "$@"; do
	case "$arg" in
	--fg) foreground=1 ;;
	--skip-web) skip_web=1 ;;
	-h | --help)
		sed -n '2,11p' "$0"
		exit 0
		;;
	*) scenario="$arg" ;;
	esac
done

port="${PORT:-8080}"
mkdir -p tmp bin

if [ "$skip_web" = 0 ]; then
	make web-build-hotseat >/dev/null
fi
go build -tags demo -o bin/server-demo ./cmd/server

pidfile=tmp/demo-server.pid
if [ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" 2>/dev/null; then
	kill "$(cat "$pidfile")"
	sleep 1
fi

export ONLINE_DEV_MODE=true PORT="$port" DEMO_SCENARIO="$scenario"
if [ "$foreground" = 1 ]; then
	exec ./bin/server-demo
fi

nohup ./bin/server-demo >tmp/demo-server.log 2>&1 &
echo $! >"$pidfile"
for _ in $(seq 1 50); do
	if ! kill -0 "$(cat "$pidfile")" 2>/dev/null; then
		echo "the demo server exited (port $port busy?); see tmp/demo-server.log" >&2
		tail -5 tmp/demo-server.log >&2
		exit 1
	fi
	if curl -fs "http://localhost:$port/healthz" >/dev/null 2>&1; then
		echo "demo scenario '$scenario' ready on http://localhost:$port (log: tmp/demo-server.log)"
		exit 0
	fi
	sleep 0.2
done
echo "the demo server did not start; see tmp/demo-server.log" >&2
tail -5 tmp/demo-server.log >&2
exit 1
