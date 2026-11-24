#!/usr/bin/env bash
# Demo of the winter turn report: starts the `winter` scenario (scripts/demo.sh)
# and submits the winter orders of P2, P3 and P4 through the API (a fief, a
# recruitment, a mill and a marriage that P4 refuses or accepts). It then
# stays on P1's orders: type them in the browser, seal them, and the report of
# the winter opens, one block per house.
#
#   scripts/winter-report-demo.sh [--skip-web]
#
# PORT (default 8080) is the server port. P1 orders to type:
#   T F RAI PCT PYL VAR
#   Q RAI HUO
#   C M PYL
set -euo pipefail

cd "$(dirname "$0")/.."

port="${PORT:-8080}"
scripts/demo.sh winter "$@"

id="$(curl -fs "http://localhost:$port/api/games" | python3 -c 'import json,sys; print(json.load(sys.stdin)[0]["id"])')"
submit() {
	curl -fs -X POST "http://localhost:$port/api/games/$id/orders?player=$1" \
		-H 'content-type: application/json' -d "$2" >/dev/null
}
submit P2 '{"chains":[],"special":[],"winter":[{"lines":"T F DRE PRT MEC FRM"},{"lines":"R T MEC"}]}'
submit P3 '{"chains":[],"special":[],"winter":[{"lines":"C M LON"},{"lines":"M N LOU PHL"}]}'
submit P4 '{"chains":[],"special":[],"winter":[{"lines":"R T CTL"},{"lines":"M N PHL LOU"}]}'

cat <<MSG
P2, P3 and P4 have sealed their winter orders. Open http://localhost:$port as P1
and seal these winter orders to resolve the turn:
  T F RAI PCT PYL VAR
  Q RAI HUO
  C M PYL
MSG
