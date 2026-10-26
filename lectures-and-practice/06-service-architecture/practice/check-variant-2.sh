#!/bin/sh
set -eu

practice_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

for variant in variant-2-original variant-2; do
 printf '%s\n' "$variant"
 (
  cd "$practice_dir/$variant"
  go test -race -count=1 ./...
 )
done
