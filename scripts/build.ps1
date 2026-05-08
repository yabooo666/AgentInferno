# AgentInferno Build Script for Windows

$BinaryName = "agentinferno"
$Version = "1.0.0"
$BuildDir = "bin"

if (-not (Test-Path $BuildDir)) {
    New-Item -Path $BuildDir -ItemType Directory
}

Write-Host "Building AgentInferno for Linux (Target)..." -ForegroundColor Cyan
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

go build -ldflags="-s -w" -o "$BuildDir/$BinaryName" ./main.go

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful! Binary located at $BuildDir/$BinaryName" -ForegroundColor Green
} else {
    Write-Host "Build failed!" -ForegroundColor Red
}

# Optional: Build for Windows (Dev/Test)
# Write-Host "Building AgentInferno for Windows (Local)..." -ForegroundColor Cyan
# go build -o "$BuildDir/$BinaryName.exe" ./cmd/agentinferno
