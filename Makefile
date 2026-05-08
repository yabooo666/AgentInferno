BINARY_NAME=agentinferno
VERSION=1.0.0
BUILD_DIR=bin

.PHONY: all build clean install test

all: build

# Load .env file if it exists
ifneq ("$(wildcard .env)","")
    include .env
    export
endif

# Injection targets
PKG=github.com/yabooo666/AgentInferno/internal/config
LDFLAGS=-X '$(PKG).BackendURL=$(BACKEND_URL)' -X '$(PKG).AgentToken=$(AGENT_TOKEN)' -X '$(PKG).HeartbeatInterval=$(HEARTBEAT_INTERVAL)'

build:
	@echo "Building AgentInferno with embedded config..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w $(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./main.go

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)

test:
	go test ./...

install: build
	@echo "Installing AgentInferno..."
	@install -m 755 $(BUILD_DIR)/$(BINARY_NAME) /usr/bin/$(BINARY_NAME)
	@mkdir -p /etc/agentinferno
	@mkdir -p /var/lib/agentinferno
	@cp configs/config.json /etc/agentinferno/config.json.example
	@cp deployments/agentinferno.service /etc/systemd/system/agentinferno.service
	@systemctl daemon-reload
	@echo "Done. Please edit /etc/agentinferno/config.json and start the service."
