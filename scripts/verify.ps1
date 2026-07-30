$ErrorActionPreference = "Stop"

Write-Host "Checking formatting..."
$unformatted = gofmt -l .
if ($unformatted) {
    Write-Host "The following files require gofmt:"
    Write-Host $unformatted
    exit 1
}

Write-Host "Running tests..."
go test ./...

Write-Host "Running go vet..."
go vet ./...

Write-Host "Verification passed."