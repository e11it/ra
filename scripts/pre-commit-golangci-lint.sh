#!/usr/bin/env bash
set -euo pipefail

mode="${1:-}"
case "$mode" in
  public|company) ;;
  *)
    echo "usage: $0 {public|company}" >&2
    exit 2
    ;;
esac

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

if [ ! -x ./bin/golangci-lint ]; then
  make go-lint-install
fi

if [ "$mode" = "public" ]; then
  ./bin/golangci-lint run --timeout=5m --config=.golangci.yml ./...
else
  ./bin/golangci-lint run --build-tags=nomsgpack,company --timeout=5m --config=.golangci.yml ./...
fi
