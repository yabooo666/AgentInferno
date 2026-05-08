const express = require('express');
const cors = require('cors');
const bodyParser = require('body-parser');
const crypto = require('crypto');
const path = require('path');

const app = express();
const PORT = 3000;

// This MUST match the HMAC_KEY in .env used to build the agent
const HMAC_KEY = 'dev-hmac-secret-replace-in-production';

app.use(cors());
app.use(bodyParser.json({ limit: '1mb' }));

// In-memory store
let agents = {};
let pendingActions = {};

// --- Helper: Sign an action with HMAC-SHA256 ---
function signAction(action) {
    const nonce = crypto.randomUUID();
    const hmac = crypto.createHmac('sha256', HMAC_KEY);
    hmac.update(action + nonce);
    const signature = hmac.digest('hex');
    return { action, nonce, signature };
}

// --- API Endpoints ---

// Agent Registration
app.post('/api/agent/register', (req, res) => {
    const data = req.body;
    const agentId = data.machine_uuid;

    if (!agentId) {
        return res.status(400).json({ error: 'machine_uuid is required' });
    }

    console.log(`[REGISTER] Agent ${agentId} from ${data.local_ip}`);

    agents[agentId] = {
        ...data,
        last_seen: new Date().toISOString(),
        status: 'online'
    };

    res.status(200).json({ message: 'registered' });
});

// Agent Heartbeat
app.post('/api/agent/heartbeat', (req, res) => {
    const data = req.body;
    const agentId = data.machine_uuid;

    if (!agentId) {
        return res.status(400).json({ error: 'machine_uuid is required' });
    }

    agents[agentId] = {
        ...data,
        last_seen: new Date().toISOString(),
        status: 'online'
    };

    // Check for pending actions — sign them with HMAC
    let response = { message: 'ok' };
    if (pendingActions[agentId]) {
        const signed = signAction(pendingActions[agentId]);
        response = { ...response, ...signed };
        delete pendingActions[agentId];
        console.log(`[ACTION] Sent HMAC-signed "${signed.action}" to ${agentId}`);
    }

    res.status(200).json(response);
});

// Admin: Request Reboot (HMAC-signed)
app.post('/api/admin/reboot/:id', (req, res) => {
    const agentId = req.params.id;
    if (!agents[agentId]) {
        return res.status(404).json({ error: 'agent not found' });
    }
    pendingActions[agentId] = 'reboot';
    console.log(`[ADMIN] Reboot queued for ${agentId} (will be HMAC-signed on next heartbeat)`);
    res.json({ message: 'reboot command queued (HMAC-signed)' });
});

// Admin: Get all agents
app.get('/api/admin/agents', (req, res) => {
    res.json(Object.values(agents));
});

// Frontend
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Mark agents offline after 30s without heartbeat
setInterval(() => {
    const now = new Date();
    for (let id in agents) {
        const lastSeen = new Date(agents[id].last_seen);
        if ((now - lastSeen) > 30000) {
            agents[id].status = 'offline';
        }
    }
}, 5000);

app.listen(PORT, () => {
    console.log(`\n🔥 AgentInferno Test Backend (Zero-Trust) running at http://localhost:${PORT}`);
    console.log(`🔑 HMAC Key: ${HMAC_KEY.substring(0, 8)}...`);
    console.log(`🚀 Waiting for agents...\n`);
});
