package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	BackendURL        string `json:"backend_url"`
	AgentToken        string `json:"agent_token"`
	HeartbeatInterval int    `json:"heartbeat_interval"`
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = "/etc/agentinferno/config.json"
		// Fallback for development if /etc doesn't exist or is not readable
		if _, err := os.Stat(path); os.IsNotExist(err) {
			path = "configs/config.json"
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	if cfg.BackendURL == "" {
		return nil, fmt.Errorf("backend_url is required")
	}
	if cfg.AgentToken == "" {
		return nil, fmt.Errorf("agent_token is required")
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 10
	}

	return &cfg, nil
}
