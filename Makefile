BINARY_NAME=agentinferno
VERSION=1.0.0
BUILD_DIR=bin

.PHONY: all build clean install test

all: build

build:
	@echo "Building AgentInferno..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./main.go

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
