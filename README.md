# AgentInferno 🔥

A production-grade, zero-trust Linux monitoring agent written in Go.

## Security Architecture

AgentInferno follows a **zero-trust model**:

1. **HTTPS Only**: Agent refuses to start if `backend_url` is not `https://` (unless `dev_mode=true` for local testing).
2. **HMAC-Signed Actions**: The backend must cryptographically sign all action commands (e.g., reboot) using HMAC-SHA256. The agent verifies the signature before executing anything.
3. **Nonce Replay Protection**: Each action includes a unique nonce. The agent rejects any nonce it has seen before.
4. **Action Whitelist**: Only explicitly whitelisted actions are accepted. Unknown action names are discarded.
5. **No os/exec in Metrics**: Docker monitoring uses the Unix socket API directly. SSH audit reads `/var/log/auth.log` directly. Zero shell commands for data collection.
6. **Minimal Exec**: The only `os/exec` call is a hardcoded `/sbin/reboot` — and only after passing HMAC + whitelist + nonce checks.
7. **Dedicated User**: The systemd service runs as a restricted `agentinferno` user, not root.

## Build

### Prerequisites
- Go 1.21+
- A `.env` file with your configuration (see below)

### Configuration (.env)
```env
BACKEND_URL=https://api.yourbackend.com
AGENT_TOKEN=your-registration-token
HMAC_KEY=your-64-char-random-secret
HEARTBEAT_INTERVAL=10
DEV_MODE=false
```

> **IMPORTANT**: The `.env` values are baked into the binary at compile time via `ldflags`. The binary does NOT need a config file at runtime.

### Build Commands

**On Windows (PowerShell):**
```powershell
.\scripts\build.ps1
```

**On Linux/macOS:**
```bash
make build
```

## Install on Ubuntu VPS

```bash
# 1. Build
make build

# 2. Install binary + create service user + register systemd
sudo make install

# 3. Allow reboot (optional)
echo "agentinferno ALL=(root) NOPASSWD: /sbin/reboot" | sudo tee /etc/sudoers.d/agentinferno

# 4. Start
sudo systemctl enable --now agentinferno
```

## Collected Metrics

| Metric | Description |
|--------|-------------|
| CPU Usage | Current CPU utilization % |
| CPU Cores | Total logical processors |
| RAM Usage | Current memory utilization % |
| Total RAM | Total system memory |
| Disk Usage | Primary partition usage % |
| Total Disk | Primary partition capacity |
| Disk I/O | Read/Write bytes |
| Network RX/TX | Total bytes received/sent |
| Interfaces | Per-interface traffic breakdown |
| Connections | Active TCP connections (ESTABLISHED) |
| Docker | Container names, status, images |
| SSH Logins | Last 5 accepted SSH sessions |
| Processes | Top 10 resource-heavy processes |
| Public IP | Cached, refreshed every 5 minutes |

## What This Agent Does NOT Do

- ❌ Execute arbitrary commands
- ❌ Open any listening ports
- ❌ Accept inbound connections
- ❌ Read SSH keys, passwords, or environment secrets
- ❌ Provide shell access of any kind
- ❌ Auto-update itself
- ❌ Load plugins or scripts

## License
MIT
