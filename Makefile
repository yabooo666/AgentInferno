BINARY_NAME=agentinferno
VERSION=1.0.0
BUILD_DIR=bin

# Load .env file if it exists
ifneq ("$(wildcard .env)","")
    include .env
    export
endif

# Injection targets
PKG=github.com/yabooo666/AgentInferno/internal/config
LDFLAGS=-X '$(PKG).BackendURL=$(BACKEND_URL)' \
        -X '$(PKG).AgentToken=$(AGENT_TOKEN)' \
        -X '$(PKG).HMACKey=$(HMAC_KEY)' \
        -X '$(PKG).HeartbeatInterval=$(HEARTBEAT_INTERVAL)' \
        -X '$(PKG).DevMode=$(DEV_MODE)'

.PHONY: all build clean install test

all: build

build:
	@echo "Building AgentInferno with embedded zero-trust config..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w $(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) .

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)

test:
	go test ./...

install: build
	@echo "Installing AgentInferno..."
	@install -m 755 $(BUILD_DIR)/$(BINARY_NAME) /usr/bin/$(BINARY_NAME)
	@mkdir -p /var/lib/agentinferno
	@useradd -r -s /bin/false agentinferno 2>/dev/null || true
	@chown agentinferno:agentinferno /var/lib/agentinferno
	@cp deployments/agentinferno.service /etc/systemd/system/agentinferno.service
	@systemctl daemon-reload
	@echo ""
	@echo "Done. Run: sudo systemctl enable --now agentinferno"
