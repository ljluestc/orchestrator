# Pre-commit hook for Go project (PowerShell version)

Write-Host "Running pre-commit checks..." -ForegroundColor Blue

# Check if Go is installed
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}

# Run go fmt
Write-Host "Running go fmt..." -ForegroundColor Blue
$fmtResult = go fmt ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: go fmt failed" -ForegroundColor Red
    exit 1
}

# Run go vet
Write-Host "Running go vet..." -ForegroundColor Blue
$vetResult = go vet ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: go vet failed" -ForegroundColor Red
    exit 1
}

# Run tests
Write-Host "Running tests..." -ForegroundColor Blue
$testResult = go test ./... -v -timeout=60s
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Tests failed" -ForegroundColor Red
    exit 1
}

# Run test coverage
Write-Host "Running test coverage..." -ForegroundColor Blue
$coverageResult = go test ./... -coverprofile=coverage.out -covermode=count
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Test coverage failed" -ForegroundColor Red
    exit 1
}

# Check coverage threshold
$coverageOutput = go tool cover -func coverage.out | Select-Object -Last 1
if ($coverageOutput -match '(\d+\.\d+)%') {
    $coverage = $matches[1]
} else {
    Write-Host "Error: Could not parse coverage percentage" -ForegroundColor Red
    exit 1
}

$threshold = 80

if ([double]$coverage -lt $threshold) {
    Write-Host "Error: Test coverage $coverage% is below threshold $threshold%" -ForegroundColor Red
    exit 1
}

Write-Host "Pre-commit checks passed!" -ForegroundColor Green
Write-Host "Test coverage: $coverage%" -ForegroundColor Green
