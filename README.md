# AgentInferno 🔥

AgentInferno is a production-grade, lightweight Linux monitoring agent written in Go. It is designed for high-security environments where minimal footprint and maximum reliability are required.

## Features

- **Lightweight**: Minimal CPU and RAM usage.
- **Secure**: No inbound ports, outbound HTTPS only, TLS 1.2+ defaults.
- **Resilient**: Exponential backoff on backend failure, graceful shutdown.
- **Modern**: Structured JSON logging (Zap), systemd integration.
- **No Dangerous Features**: ZERO remote-shell, ZERO arbitrary command execution.

## Installation

### Prerequisites
- Go 1.21+ (for building)
- Ubuntu 22.04+ / Debian-based system

### Build
Since the target is Linux, you can cross-compile from any platform:

**On Linux/macOS:**
```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/agentinferno ./cmd/agentinferno
```
(Or use `make build` if you have `make` installed)

**On Windows (PowerShell):**
```powershell
.\scripts\build.ps1
```

The static binary will be available in `bin/agentinferno`.

### Quick Setup (as Root)
```bash
# 1. Build and install binary and service
sudo make install

# 2. Configure the agent
sudo nano /etc/agentinferno/config.json

# 3. Start the service
sudo systemctl enable agentinferno
sudo systemctl start agentinferno
```

## Configuration

The agent loads configuration from `/etc/agentinferno/config.json`.

```json
{
  "backend_url": "https://api.yourbackend.com",
  "agent_token": "YOUR_SECRET_TOKEN",
  "heartbeat_interval": 10
}
```

## Security Philosophy

1. **Outbound Only**: The agent never listens on any port. It only initiates connections to the backend.
2. **Hardened Service**: The systemd service runs with restricted privileges (`NoNewPrivileges`, `ProtectSystem=full`).
3. **No Execution**: The codebase contains no `os/exec` or similar calls that could be abused for remote command execution.
4. **Data Privacy**: Only performance metrics are collected. No sensitive files, environment variables, or private keys are ever read.

## Collected Metrics

- Hostname & OS Version
- System Uptime
- CPU Usage (%)
- RAM Usage (%)
- Disk Usage (%)
- Network RX/TX Bytes
- Machine UUID (Persistent)

## License
MIT
