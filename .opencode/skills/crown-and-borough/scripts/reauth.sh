#!/usr/bin/env bash

set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
source "$script_dir/context.sh"

printf 'Authentication expired for instance %s.\n' "$(cb_instance_id)"
