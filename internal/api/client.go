package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yabooo666/AgentInferno/internal/config"
	"github.com/yabooo666/AgentInferno/internal/logger"
	"go.uber.org/zap"
)

type Client struct {
	config *config.Config
	http   *http.Client
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		config: cfg,
		http: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}
}

func (c *Client) Register(ctx context.Context, payload interface{}) (map[string]interface{}, error) {
	return c.post(ctx, "/api/agent/register", payload)
}

func (c *Client) Heartbeat(ctx context.Context, payload interface{}) (map[string]interface{}, error) {
	return c.post(ctx, "/api/agent/heartbeat", payload)
}

func (c *Client) post(ctx context.Context, endpoint string, payload interface{}) (map[string]interface{}, error) {
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
		respBody, _ := io.ReadAll(resp.Body)
		logger.Log.Warn("backend returned error status",
			zap.Int("status", resp.StatusCode),
			zap.String("endpoint", endpoint),
			zap.String("response", string(respBody)),
		)
		return nil, fmt.Errorf("backend returned status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil // Silently ignore if not JSON
	}

	return result, nil
}
