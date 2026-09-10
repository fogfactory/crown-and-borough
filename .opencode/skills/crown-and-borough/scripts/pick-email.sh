#!/usr/bin/env bash
# Print a stable human-readable disposable email for one bot instance.
# Usage: pick-email.sh [--instance-id ID] [--domain DOMAIN]
set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
colors=(red blue green yellow violet amber emerald lavender crimson azure)
animals=(wolf fox lynx otter heron falcon badger weasel rabbit owl)
instance="${CB_INSTANCE_ID:-${OMP_SESSION_ID:-${OMP_SESSION:-${SESSION_ID:-}}}}"
domain="${CB_EMAIL_DOMAIN:-mail.com}"

while (($# > 0)); do
  case "$1" in
    --instance-id) instance="${2:?--instance-id requires an id}"; shift 2 ;;
    --domain) domain="${2:?--domain requires a domain}"; shift 2 ;;
    *) printf 'usage: pick-email.sh [--instance-id ID] [--domain DOMAIN]\n' >&2; exit 2 ;;
  esac
done

if [[ -z "$instance" ]]; then
  instance="$(bash "$script_dir/instance-id.sh")"
fi
safe_instance="${instance//[^A-Za-z0-9]/-}"
checksum="$(printf '%s' "$instance" | cksum | while read -r value _; do printf '%s' "$value"; done)"
color="${colors[$((checksum % ${#colors[@]}))]}"
animal="${animals[$(((checksum / ${#colors[@]}) % ${#animals[@]}))]}"
printf '%s.%s.%s@%s\n' "$color" "$animal" "$safe_instance" "$domain"
