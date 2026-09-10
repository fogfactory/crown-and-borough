#!/usr/bin/env bash

set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
# shellcheck source=./context.sh
source "$script_dir/context.sh"

persona="${CB_PERSONA:-diplomat}"
play_style="opportunistic"
trust="medium"
tone="formal"
display_name="The Opportunist"
declare -a rules=()

load_persona() {
  case "$1" in
    conqueror)
      display_name="The Conqueror"; play_style="aggressive"; trust="none"; tone="curt"
      rules=("Expand toward the weakest valuable border before a neighbor can consolidate." "Accept a pact only when it frees an army for a stronger target." "Break a pact when the partner is exposed and the attack cannot create a fatal counter-front.") ;;
    diplomat)
      display_name="The Diplomat"; play_style="opportunistic"; trust="medium"; tone="formal"
      rules=("Maintain a credible bilateral channel with every nearby power." "Trade specific short pacts for time; never promise indefinite peace by default." "Betray only when the immediate gain outweighs the diplomatic cost.") ;;
    turtle)
      display_name="The Turtle"; play_style="defensive"; trust="high"; tone="curt"
      rules=("Protect the capital route and preserve a legal retreat before seeking expansion." "Prefer mills, depots, and supplied armies over speculative attacks." "Counterattack only with a unique force advantage or decisive supply denial.") ;;
    brigand)
      display_name="The Brigand"; play_style="opportunistic"; trust="none"; tone="threatening"
      rules=("Prefer pillage, famine, and isolated infrastructure over honorable front-line fights." "Promise safety only while the neighbor's stock or army is useful to preserve." "Convert every successful raid into leverage for the next negotiation.") ;;
    merchant)
      display_name="The Merchant"; play_style="mercantile"; trust="medium"; tone="friendly"
      rules=("Treat resources, mills, depots, and safe transfers as strategic assets." "Offer concrete gifts or routes in exchange for turn-limited non-aggression." "Avoid expensive war unless it opens a source or denies supply.") ;;
    honorable)
      display_name="The Honorable Lord"; play_style="honest"; trust="high"; tone="formal"
      rules=("State the scope and expiry of every promise and honor it exactly." "Refuse profitable betrayal unless the other party broke the pact first." "Punish a broken promise decisively so future agreements remain meaningful.") ;;
    intriguer)
      display_name="The Intriguer"; play_style="treacherous"; trust="low"; tone="theatrical"
      rules=("Send different compatible promises when doing so hides the target." "Never reveal the decisive order before resolution." "Betray when the target has committed its defense elsewhere.") ;;
    chaotic)
      display_name="The Wildcard"; play_style="random"; trust="low"; tone="curt"
      rules=("Vary openings and targets, but keep every order legal and supplied." "Make agreements narrow enough that changing direction remains possible." "Do not randomize survival decisions: protect a threatened capital first.") ;;
    opportunist)
      display_name="The Opportunist"; play_style="opportunistic"; trust="medium"; tone="formal"
      rules=("Wait for a unique force or supply advantage before committing." "Accept help without giving away the real target." "Re-evaluate every pact when a player's border becomes exposed.") ;;
    *)
      printf 'unknown persona: %s\n' "$1" >&2
      return 1
      ;;
  esac
}

load_persona "$persona"

while (($# > 0)); do
  case "$1" in
    --id)
      persona="${2:?--id requires a persona id}"
      load_persona "$persona"
      shift 2
      ;;
    --trait)
      value="${2:?--trait requires name=value}"
      name="${value%%=*}"
      value="${value#*=}"
      case "$name" in
        play_style|style) play_style="$value" ;;
        trust|trust_level) trust="$value" ;;
        tone|negotiation_tone) tone="$value" ;;
        *) printf 'unknown trait: %s\n' "$name" >&2; exit 2 ;;
      esac
      shift 2
      ;;
    --help)
      printf 'usage: persona-init.sh [--id PERSONA] [--trait name=value]\n'
      exit 0
      ;;
    *) printf 'unknown argument: %s\n' "$1" >&2; exit 2 ;;
  esac
done

case "$play_style" in aggressive|defensive|opportunistic|mercantile|honest|treacherous|random) ;; *) printf 'invalid play_style: %s\n' "$play_style" >&2; exit 2 ;; esac
case "$trust" in high|medium|low|none) ;; *) printf 'invalid trust: %s\n' "$trust" >&2; exit 2 ;; esac
case "$tone" in curt|formal|friendly|theatrical|threatening) ;; *) printf 'invalid tone: %s\n' "$tone" >&2; exit 2 ;; esac

cb_require_jq
cb_ensure_instance_dir
if [[ -n "${CB_GAME_ID:-}" ]] || [[ -f "$(cb_game_cache_file)" ]]; then
  cb_ensure_bot_dir
  output="$(cb_bot_dir)/persona.json"
else
  output="$(cb_instance_dir)/persona.json"
fi
rules_json="$(printf '%s\n' "${rules[@]}" | jq -Rsc 'split("\n") | map(select(length > 0))')"
jq -n \
  --arg id "$persona" \
  --arg display_name "$display_name" \
  --arg play_style "$play_style" \
  --arg trust "$trust" \
  --arg tone "$tone" \
  --argjson rules "$rules_json" \
  '{id:$id,display_name:$display_name,traits:{play_style:$play_style,trust:$trust,tone:$tone},decision_rules:$rules,created_at:(now|todateiso8601)}' \
  >"$output"
chmod 600 "$output"
