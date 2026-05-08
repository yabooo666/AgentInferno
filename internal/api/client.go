package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yabooo666/AgentInferno/internal/config"
	"github.com/yabooo666/AgentInferno/internal/logger"
	"go.uber.org/zap"
)

// Maximum allowed response body sizes to prevent memory exhaustion attacks
const (
	maxResponseSize = 4096  // 4KB for normal API responses
	maxErrorSize    = 2048  // 2KB for error response bodies
)

// ActionResponse represents a verified backend response that may contain an action
type ActionResponse struct {
	Message   string `json:"message"`
	Action    string `json:"action,omitempty"`
	Signature string `json:"signature,omitempty"` // HMAC-SHA256 of the action
	Nonce     string `json:"nonce,omitempty"`     // One-time nonce to prevent replay
}

type Client struct {
	config    *config.Config
	http      *http.Client
	usedNonces map[string]time.Time // Replay protection
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		config: cfg,
		http: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
				MaxIdleConns:          10,
				// Node.js Express defaults to a 5-second Keep-Alive timeout.
				// If we set IdleConnTimeout to >= 5s and heartbeat is 5s, we hit a race condition
				// where the server closes the connection exactly as we try to reuse it, causing an EOF.
				// Setting this to 3s forces the Go client to safely close the connection first.
				IdleConnTimeout:       3 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
		usedNonces: make(map[string]time.Time),
	}
}

func (c *Client) Register(ctx context.Context, payload interface{}) (*ActionResponse, error) {
	return c.post(ctx, "/api/agent/register", payload)
}

func (c *Client) Heartbeat(ctx context.Context, payload interface{}) (*ActionResponse, error) {
	return c.post(ctx, "/api/agent/heartbeat", payload)
}

func (c *Client) post(ctx context.Context, endpoint string, payload interface{}) (*ActionResponse, error) {
	url := fmt.Sprintf("%s%s", c.config.BackendURL, endpoint)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.AgentToken))
	req.Header.Set("User-Agent", "AgentInferno/1.0.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read limited error body to prevent memory exhaustion
		limitedReader := io.LimitReader(resp.Body, maxErrorSize)
		respBody, _ := io.ReadAll(limitedReader)
		logger.Log.Warn("backend returned error",
			zap.Int("status", resp.StatusCode),
			zap.String("endpoint", endpoint),
			zap.String("response", string(respBody)),
		)
		return nil, fmt.Errorf("backend returned status: %d", resp.StatusCode)
	}

	// Read limited response body
	limitedReader := io.LimitReader(resp.Body, maxResponseSize)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result ActionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		// Not JSON — return empty response (no action)
		return &ActionResponse{Message: "ok"}, nil
	}

	return &result, nil
}

// VerifyAction checks the HMAC signature on an action command from the backend.
// This is the core zero-trust mechanism: the backend must prove it holds the shared key.
func (c *Client) VerifyAction(resp *ActionResponse) bool {
	if resp.Action == "" {
		return false // No action to verify
	}

	if resp.Signature == "" || resp.Nonce == "" {
		logger.Log.Warn("received action WITHOUT signature — rejecting",
			zap.String("action", resp.Action),
		)
		return false
	}

	// Replay protection: reject already-used nonces
	if _, used := c.usedNonces[resp.Nonce]; used {
		logger.Log.Warn("replay attack detected — nonce already used",
			zap.String("nonce", resp.Nonce),
		)
		return false
	}

	// Verify HMAC-SHA256: sign(action + nonce) must match signature
	mac := hmac.New(sha256.New, []byte(c.config.HMACKey))
	mac.Write([]byte(resp.Action + resp.Nonce))
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedMAC), []byte(resp.Signature)) {
		logger.Log.Warn("HMAC signature verification FAILED — rejecting action",
			zap.String("action", resp.Action),
		)
		return false
	}

	// Store nonce to prevent replay (clean old ones periodically)
	c.usedNonces[resp.Nonce] = time.Now()
	c.cleanOldNonces()

	logger.Log.Info("action signature verified successfully",
		zap.String("action", resp.Action),
	)
	return true
}

// cleanOldNonces removes nonces older than 5 minutes to prevent memory leak
func (c *Client) cleanOldNonces() {
	cutoff := time.Now().Add(-5 * time.Minute)
	for nonce, ts := range c.usedNonces {
		if ts.Before(cutoff) {
			delete(c.usedNonces, nonce)
		}
	}
}
