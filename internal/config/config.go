package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

var (
	// Build-time injected variables
	BackendURL        string
	AgentToken        string
	HeartbeatInterval string // String because ldflags only supports strings
)

type Config struct {
	BackendURL        string `json:"backend_url"`
	AgentToken        string `json:"agent_token"`
	HeartbeatInterval int    `json:"heartbeat_interval"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		BackendURL:        BackendURL,
		AgentToken:        AgentToken,
		HeartbeatInterval: 10,
	}

	// Parse heartbeat interval if injected
	if HeartbeatInterval != "" {
		if val, err := strconv.Atoi(HeartbeatInterval); err == nil {
			cfg.HeartbeatInterval = val
		}
	}

	// If hardcoded config is present, we can skip file loading if path is empty
	if cfg.BackendURL != "" && cfg.AgentToken != "" && path == "" {
		return cfg, nil
	}

	// Fallback to file loading if provided or if hardcoded is missing
	if path == "" {
		path = "/etc/agentinferno/config.json"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			path = "configs/config.json"
		}
	}

	if _, err := os.Stat(path); err == nil {
		file, err := os.Open(path)
		if err == nil {
			defer file.Close()
			var fileCfg Config
			if err := json.NewDecoder(file).Decode(&fileCfg); err == nil {
				// File overrides build-time if both exist
				if fileCfg.BackendURL != "" {
					cfg.BackendURL = fileCfg.BackendURL
				}
				if fileCfg.AgentToken != "" {
					cfg.AgentToken = fileCfg.AgentToken
				}
				if fileCfg.HeartbeatInterval > 0 {
					cfg.HeartbeatInterval = fileCfg.HeartbeatInterval
				}
			}
		}
	}

	if cfg.BackendURL == "" || cfg.AgentToken == "" {
		return nil, fmt.Errorf("backend_url and agent_token must be provided via build flags or config file")
	}

	return cfg, nil
}
