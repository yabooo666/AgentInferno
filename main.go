package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/yabooo666/AgentInferno/internal/config"
	"github.com/yabooo666/AgentInferno/internal/logger"
	"github.com/yabooo666/AgentInferno/internal/service"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Initialize Logger
	logger.Init()
	defer logger.Sync()

	// Load Config
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Log.Fatal("failed to load configuration", zap.Error(err))
	}

	// Initialize Agent
	agent, err := service.NewAgent(cfg)
	if err != nil {
		logger.Log.Fatal("failed to initialize agent", zap.Error(err))
	}

	// Context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Run Agent
	if err := agent.Run(ctx); err != nil {
		logger.Log.Fatal("agent execution failed", zap.Error(err))
	}
}
