# AgentInferno Build Script for Windows
# Reads .env and injects config into the binary via ldflags

$BinaryName = "agentinferno"
$Version = "1.0.0"
$BuildDir = "bin"

if (-not (Test-Path $BuildDir)) {
    New-Item -Path $BuildDir -ItemType Directory
}

# Load .env
$BackendURL = ""
$AgentToken = ""
$HMACKey = ""
$HeartbeatInterval = "10"
$DevMode = "false"

if (Test-Path ".env") {
    Get-Content .env | ForEach-Object {
        if ($_ -match "^\s*#" -or $_ -match "^\s*$") { return }
        $name, $value = $_.split('=', 2)
        $name = $name.Trim()
        $value = $value.Trim()
        if ($name -eq "BACKEND_URL") { $BackendURL = $value }
        if ($name -eq "AGENT_TOKEN") { $AgentToken = $value }
        if ($name -eq "HMAC_KEY") { $HMACKey = $value }
        if ($name -eq "HEARTBEAT_INTERVAL") { $HeartbeatInterval = $value }
        if ($name -eq "DEV_MODE") { $DevMode = $value }
    }
} else {
    Write-Host "ERROR: .env file not found! Cannot build without configuration." -ForegroundColor Red
    exit 1
}

if (-not $BackendURL -or -not $AgentToken -or -not $HMACKey) {
    Write-Host "ERROR: BACKEND_URL, AGENT_TOKEN, and HMAC_KEY must be set in .env" -ForegroundColor Red
    exit 1
}

$Pkg = "github.com/yabooo666/AgentInferno/internal/config"
$LdFlags = "-s -w -X '${Pkg}.BackendURL=${BackendURL}' -X '${Pkg}.AgentToken=${AgentToken}' -X '${Pkg}.HMACKey=${HMACKey}' -X '${Pkg}.HeartbeatInterval=${HeartbeatInterval}' -X '${Pkg}.DevMode=${DevMode}'"

# --- Build for Linux (Production Target) ---
Write-Host ""
Write-Host "=== AgentInferno Build ===" -ForegroundColor Cyan
Write-Host "Backend:   $BackendURL"
Write-Host "Dev Mode:  $DevMode"
Write-Host ""

$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

go build -ldflags="$LdFlags" -o "$BuildDir/$BinaryName" .

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful: $BuildDir/$BinaryName" -ForegroundColor Green
} else {
    Write-Host "Build FAILED!" -ForegroundColor Red
    exit 1
}

# --- Build for Windows (Local Testing) ---
Write-Host ""
Write-Host "Building Windows binary for local testing..." -ForegroundColor Cyan
$env:GOOS = "windows"
$env:GOARCH = "amd64"

go build -ldflags="$LdFlags" -o "$BuildDir/$BinaryName.exe" .

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful: $BuildDir/$BinaryName.exe" -ForegroundColor Green
} else {
    Write-Host "Windows build FAILED!" -ForegroundColor Red
}

Write-Host ""
Write-Host "Done." -ForegroundColor Green
