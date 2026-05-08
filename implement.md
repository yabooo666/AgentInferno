# AgentInferno — Backend Implementation Guide

This document defines the exact data structures, API requirements, and security protocols required to build the production backend and website for the AgentInferno monitoring agents. Provide this file to any AI assisting you in building the web dashboard.

---

## 1. System Architecture

AgentInferno uses a **Zero-Trust, Outbound-Only** architecture:
- **No Listening Ports**: The VPS agent has no open ports. It only makes outbound HTTP POST requests to the backend.
- **Stateless Polling**: The agent polls the backend via a Heartbeat.
- **Cryptographic Action Verification**: The agent will not execute commands (like reboot) unless they are cryptographically signed by the backend using a shared HMAC-SHA256 secret.
- **Token Authentication**: All requests are authenticated via a Bearer token.

---

## 2. Authentication Requirements

Every request from the agent to the backend includes an HTTP header:
```http
Authorization: Bearer <AGENT_TOKEN>
```
The backend MUST verify this token before accepting data.

---

## 3. Required API Endpoints

The backend must implement two core POST endpoints to receive data from the agents.

### A. Registration: `POST /api/agent/register`
Fired once when the agent starts up (with exponential backoff if the backend is down).
- **Request Body**: Full `Stats` JSON (see Data Payload below).
- **Response**: `200 OK` (JSON: `{"message": "registered"}`)

### B. Heartbeat: `POST /api/agent/heartbeat`
Fired every `N` seconds (configured via `.env`).
- **Request Body**: Full `Stats` JSON (see Data Payload below).
- **Response**: Must return a 200 OK. If an action (like reboot) is queued for this agent, the response MUST include the cryptographic signature (see Action Execution below).

---

## 4. The Data Payload (JSON)

Every register and heartbeat request sends this exact JSON structure. The backend database/schema should be designed to store or visualize this data.

```json
{
  "machine_uuid": "e8b2a3f1-...", // Unique persistent ID for the VPS
  "hostname": "VDS-017",
  "os": "ubuntu 22.04 (5.15.0-100-generic)",
  "uptime": 174830, // in seconds
  
  "local_ip": "10.0.0.5",
  "public_ip": "198.51.100.23",
  "timestamp": 1715132000, // Unix epoch

  // Hardware Specs & Usage
  "cpu_count": 4,
  "cpu_usage": 12.5, // Percentage
  "total_ram": 8589934592, // Bytes (8GB)
  "ram_usage": 45.2, // Percentage
  "total_disk": 107374182400, // Bytes (100GB)
  "disk_usage": 60.1, // Percentage

  // Disk I/O (Cumulative Bytes)
  "disk_io": {
    "read_bytes": 104857600,
    "write_bytes": 52428800
  },

  // Network (Cumulative Bytes)
  "net_rx": 500000000, 
  "net_tx": 120000000,
  
  // Breakdown of active Network Interfaces
  "interfaces": [
    { "name": "eth0", "rx": 490000000, "tx": 115000000 }
  ],

  // Security: Live Network Connections (Top 15)
  "connections": [
    { "local_addr": "10.0.0.5:22", "remote_addr": "203.0.113.5:51234", "status": "ESTABLISHED" }
  ],

  // Security: Docker Containers (if installed)
  "docker": [
    { "name": "nginx-proxy", "status": "Up 2 days", "image": "nginx:latest" }
  ],

  // Security: Last 5 SSH Logins
  "ssh_logins": [
    { "user": "root", "ip": "203.0.113.5", "date": "May 8 03:00:01" }
  ],

  // Resource Hogs: Top 10 CPU/RAM Processes
  "processes": [
    { "pid": 1234, "name": "node", "cpu": 5.2, "memory": 2.1 }
  ]
}
```

---

## 5. Action Execution (Zero-Trust)

If the user clicks "Reboot Server" on the website dashboard, the backend MUST queue a `reboot` action for that specific `machine_uuid`.

When that agent next hits the `/api/agent/heartbeat` endpoint, the backend must respond with a cryptographically signed payload.

### The Cryptography
The backend and the agent share a secret `HMAC_KEY` (64+ chars).
To sign an action, the backend must generate a one-time `nonce` (UUID) and hash `action + nonce` using HMAC-SHA256.

**Node.js Example for Backend:**
```javascript
const crypto = require('crypto');
const HMAC_KEY = "your-shared-secret-key";

function signAction(action) {
    const nonce = crypto.randomUUID();
    const hmac = crypto.createHmac('sha256', HMAC_KEY);
    hmac.update(action + nonce);
    return {
        action: action,
        nonce: nonce,
        signature: hmac.digest('hex')
    };
}
```

### The Required HTTP Response
If an action is queued, the heartbeat response MUST look exactly like this:
```json
{
  "message": "ok",
  "action": "reboot",
  "nonce": "d9b2d63d-a233-4123-8478-3a21394c8b82",
  "signature": "e5c6... (64 char hex string)"
}
```
*Note: If the signature is invalid or the nonce was already used, the agent will reject the command and log a security warning.*

---

## 6. Website UI Recommendations

When building the frontend dashboard, you should implement the following components based on the data provided:

1. **Fleet Overview**: A grid or table of all `machine_uuid`s, showing Online/Offline status (calculated by checking if `timestamp` is within the last 30 seconds).
2. **Hardware Visualizations**: Donut charts for `cpu_usage`, `ram_usage`, and `disk_usage`.
3. **Network Throughput**: Line charts tracking `net_rx` and `net_tx` over time (requires the backend to store historical heartbeat data).
4. **Security Audit Panel**: A dedicated tab showing `ssh_logins` and active `connections` to identify unauthorized access.
5. **Docker Health**: A table showing all running containers across the fleet.
6. **Admin Controls**: A "Reboot Server" button that requires a double-confirmation modal before queuing the action in the backend.
