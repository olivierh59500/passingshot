#!/bin/sh
set -eu
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"
export GOCACHE="$PWD/.cache/go-build"
exec go run ./cmd/android "$@"
