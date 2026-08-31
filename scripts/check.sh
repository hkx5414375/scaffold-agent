#!/usr/bin/env sh
set -eu

unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
  printf 'Go files require gofmt:\n%s\n' "$unformatted" >&2
  exit 1
fi

go vet ./...
go test ./...
go run ./cmd/scaffold-agent doctor --json

