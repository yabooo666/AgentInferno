package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Build-time injected variables via ldflags.
// These are the ONLY trusted values — they are sealed into the binary at compile time.
var (
	BackendURL        string // HTTPS URL of the trusted backend
	AgentToken        string // Initial bootstrap token for first registration
	HMACKey           string // Shared HMAC-SHA256 key for verifying backend action commands
	HeartbeatInterval string // Heartbeat interval in seconds (string because ldflags)
	DevMode           string // Set to "true" to allow HTTP in development ONLY
)

type Config struct {
	BackendURL        string
	AgentToken        string
	HMACKey           string
	HeartbeatInterval int
	DevMode           bool
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		BackendURL:        BackendURL,
		AgentToken:        AgentToken,
		HMACKey:           HMACKey,
		HeartbeatInterval: 10,
		DevMode:           DevMode == "true",
	}

	// Parse heartbeat interval from ldflags
	if HeartbeatInterval != "" {
		if val, err := strconv.Atoi(HeartbeatInterval); err == nil && val > 0 {
			cfg.HeartbeatInterval = val
		}
	}

	// If build-time config is complete and no explicit path given, skip file loading
	if cfg.BackendURL != "" && cfg.AgentToken != "" && cfg.HMACKey != "" && path == "" {
		return cfg, cfg.validate()
	}

	// Fallback: try to load from file for values not set at build time
	if path == "" {
		path = "/etc/agentinferno/config.json"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			path = "configs/config.json"
		}
	}

	if _, err := os.Stat(path); err == nil {
		if err := cfg.loadFromFile(path); err != nil {
			return nil, fmt.Errorf("config file error: %w", err)
		}
	}

	return cfg, cfg.validate()
}

func (c *Config) loadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var fileCfg struct {
		BackendURL        string `json:"backend_url"`
		AgentToken        string `json:"agent_token"`
		HMACKey           string `json:"hmac_key"`
		HeartbeatInterval int    `json:"heartbeat_interval"`
		DevMode           bool   `json:"dev_mode"`
	}

	if err := json.NewDecoder(file).Decode(&fileCfg); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	// File values override build-time defaults
	if fileCfg.BackendURL != "" {
		c.BackendURL = fileCfg.BackendURL
	}
	if fileCfg.AgentToken != "" {
		c.AgentToken = fileCfg.AgentToken
	}
	if fileCfg.HMACKey != "" {
		c.HMACKey = fileCfg.HMACKey
	}
	if fileCfg.HeartbeatInterval > 0 {
		c.HeartbeatInterval = fileCfg.HeartbeatInterval
	}
	if fileCfg.DevMode {
		c.DevMode = true
	}

	return nil
}

func (c *Config) validate() error {
	if c.BackendURL == "" {
		return fmt.Errorf("backend_url is required")
	}
	if c.AgentToken == "" {
		return fmt.Errorf("agent_token is required")
	}
	if c.HMACKey == "" {
		return fmt.Errorf("hmac_key is required for zero-trust action verification")
	}

	// Parse and validate the backend URL
	u, err := url.Parse(c.BackendURL)
	if err != nil {
		return fmt.Errorf("invalid backend_url: %w", err)
	}

	// ENFORCE HTTPS unless explicitly in dev mode
	if !c.DevMode && u.Scheme != "https" {
		return fmt.Errorf("backend_url MUST use https:// in production (got %s://). Set dev_mode=true for local testing", u.Scheme)
	}

	if u.Host == "" {
		return fmt.Errorf("backend_url has no host")
	}

	// Strip trailing slash for consistency
	c.BackendURL = strings.TrimRight(c.BackendURL, "/")

	return nil
}
