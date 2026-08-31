$ErrorActionPreference = 'Stop'

$unformatted = git ls-files '*.go' | ForEach-Object { gofmt -l $_ }
if ($unformatted) {
    Write-Error "Go files require gofmt:`n$unformatted"
}

go vet ./...
go test ./...
go run ./cmd/scaffold-agent doctor --json
