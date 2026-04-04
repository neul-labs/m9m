#!/usr/bin/env bash

set -eu

cd "$(dirname "$0")/.."

backup_files="$(
  find . \
    -path './.git' -prune -o \
    -path './web/node_modules' -prune -o \
    -path './web/dist' -prune -o \
    -path './internal/web/dist' -prune -o \
    -type f \( -name '*.bak' -o -name '*.old' -o -name '*.disabled' \) \
    -print | sort
)"

if [ -n "$backup_files" ]; then
  echo "Backup or disabled source artifacts are not allowed:"
  printf '  %s\n' "$backup_files"
  exit 1
fi

echo "No backup artifacts found."
