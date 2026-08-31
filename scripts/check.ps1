$ErrorActionPreference = 'Stop'

$unformatted = gofmt -l .
if ($unformatted) {
    Write-Error "Go files require gofmt:`n$unformatted"
}

go vet ./...
go test ./...
go run ./cmd/scaffold-agent doctor --json

