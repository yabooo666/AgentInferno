# AgentInferno Build Script for Windows

$BinaryName = "agentinferno"
$Version = "1.0.0"
$BuildDir = "bin"

if (-not (Test-Path $BuildDir)) {
    New-Item -Path $BuildDir -ItemType Directory
}

# Load .env if it exists
$BackendURL = ""
$AgentToken = ""
$HeartbeatInterval = "10"

if (Test-Path ".env") {
    Get-Content .env | ForEach-Object {
        $name, $value = $_.split('=')
        if ($name -eq "BACKEND_URL") { $BackendURL = $value }
        if ($name -eq "AGENT_TOKEN") { $AgentToken = $value }
        if ($name -eq "HEARTBEAT_INTERVAL") { $HeartbeatInterval = $value }
    }
}

$Pkg = "github.com/yabooo666/AgentInferno/internal/config"
$LdFlags = "-s -w -X '$Pkg.BackendURL=$BackendURL' -X '$Pkg.AgentToken=$AgentToken' -X '$Pkg.HeartbeatInterval=$HeartbeatInterval'"

Write-Host "Building AgentInferno for Linux with embedded config..." -ForegroundColor Cyan
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

go build -ldflags="$LdFlags" -o "$BuildDir/$BinaryName" ./main.go

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful! Binary located at $BuildDir/$BinaryName" -ForegroundColor Green
} else {
    Write-Host "Build failed!" -ForegroundColor Red
}

# Optional: Build for Windows (Dev/Test)
# Write-Host "Building AgentInferno for Windows (Local)..." -ForegroundColor Cyan
# go build -o "$BuildDir/$BinaryName.exe" ./cmd/agentinferno
