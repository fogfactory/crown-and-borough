#!/usr/bin/env bash
# Online demo of the chat: starts the local online stack (Firebase emulators
# and server, see scripts/league-server.sh), creates a two-house game, and has
# Bruno write messages citing places and nobles (names, elisions, codes, and
# words that must stay plain text), so the links to the map can be checked.
#
#   scripts/chat-demo.sh [--skip-web]
#
# PORT (default 8080) is the server port. At the end it prints a sign-in link
# for Alice (alice@mail.com), who receives the messages; Bruno is
# bruno@mail.com. Stop with: scripts/league-server.sh stop, then
# docker compose -p crown-and-borough down.
set -euo pipefail

cd "$(dirname "$0")/.."

skip_web=0
for arg in "$@"; do
	case "$arg" in
	--skip-web) skip_web=1 ;;
	-h | --help)
		sed -n '2,12p' "$0"
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
base="http://localhost:$port"
auth="http://127.0.0.1:$auth_port"
project=demo-crown-and-borough
key=fake-api-key

if [ "$skip_web" = 0 ]; then
	make web-build >/dev/null
fi
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
	'{"name":"Démo du chat","seed":"chat-demo","years":4,"spectate":true,"players":2}')
game_id=$(jq -r .id <<<"$game")
invite=$(jq -r .inviteCode <<<"$game")
alice_house=$(api "$alice" POST "/api/games/$game_id/join" "{\"inviteCode\":\"$invite\"}" | jq -r .player.id)
api "$bruno" POST "/api/games/$game_id/join" "{\"inviteCode\":\"$invite\"}" >/dev/null

map=$(api "$bruno" GET "/api/games/$game_id/map")
state=$(api "$bruno" GET "/api/games/$game_id/state")

# Material for the messages, read from the forged game.
pick() { jq -nr --argjson map "$map" --argjson state "$state" --arg alice "$alice_house" "$1"; }
alice_noble=$(pick '[$state.nobles[] | select(.owner == $alice)][0]')
alice_first=$(jq -r .firstName <<<"$alice_noble")
alice_name=$(jq -r .name <<<"$alice_noble")
alice_code=$(jq -r .code <<<"$alice_noble")
held=$(pick '$state.territories[] | select(.owner == $alice) | .id' | head -1)
held_name=$(pick "\$map.territories[] | select(.id == \"$held\") | .name")
vowel=$(pick '[$map.territories[] | select(.name | test("^[AEIOUHÉ]"))][0].name // "Hautebourg"')
hyphen=$(pick '[$map.territories[] | select(.name | test("-"))][0].name // empty')
other=$(pick "[\$map.territories[] | select(.id != \"$held\")][3]")
other_name=$(jq -r .name <<<"$other")
other_code=$(jq -r .id <<<"$other")
shared=$(pick '[$state.nobles[].code] as $nobles | [$map.territories[] | select(.id as $id | $nobles | index($id))][0].id // empty')
lower_held=$(tr '[:upper:]' '[:lower:]' <<<"$held_name")
lower_code=$(tr '[:upper:]' '[:lower:]' <<<"$held")

# The chat takes at most one message per second from a house.
post() {
	sleep 1.1
	api "$bruno" POST "/api/games/$game_id/messages" \
		"$(jq -n --arg channel "$1" --arg body "$2" '{channel: $channel, body: $body}')" >/dev/null
}

post public "Nos armées passeront à $other_name, près d'$vowel.${hyphen:+ Rendez-vous ensuite à $hyphen.}"
post public "$alice_first tient $held_name ($held). Que $alice_name ($alice_code) se méfie : nous marchons de $other_code vers $held."
post "$alice_house" "Message privé : $alice_first, retirez-vous de $held_name avant l'été."
post "$alice_house" "Rien de ceci ne doit être lié : Saint-$held_name, ${held_name}ais, $lower_held, $lower_code, ${held}X, ${alice_first}net."
if [ -n "$shared" ]; then
	post public "Le trigramme $shared désigne à la fois un territoire et un noble : le survol éclaire les deux."
fi

link=$(request_link alice@mail.com "$base/finish?redirect=/games/$game_id" | jq -r .oobLink)
cat <<EOF

chat demo ready: game "Démo du chat" ($game_id) on $base
  Alice ($alice_house) reads Bruno's messages in the public room and in their
  private conversation. Sign in as Alice with this link (enter
  alice@mail.com if asked):
  $link
  Other links: $auth/emulator/v1/projects/$project/oobCodes
EOF
