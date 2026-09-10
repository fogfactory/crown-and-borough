#!/usr/bin/env bash
# Print a human-readable disposable email: <color>.<animal>@mail.com
set -euo pipefail

colors=(red blue green yellow violet amber emerald lavender crimson azure)
animals=(wolf fox lynx otter heron falcon badger weasel rabbit owl)

c="${colors[$((RANDOM % ${#colors[@]}))]}"
a="${animals[$((RANDOM % ${#animals[@]}))]}"
printf '%s.%s@mail.com\n' "$c" "$a"