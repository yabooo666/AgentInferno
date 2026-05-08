package service

import (
	"context"
	"time"

	"github.com/yabooo666/AgentInferno/internal/api"
	"github.com/yabooo666/AgentInferno/internal/config"
	"github.com/yabooo666/AgentInferno/internal/logger"
	"github.com/yabooo666/AgentInferno/internal/machine"
	"github.com/yabooo666/AgentInferno/internal/metrics"
	"go.uber.org/zap"
)

type Agent struct {
	config    *config.Config
	apiClient *api.Client
	machineID string
}

func NewAgent(cfg *config.Config) (*Agent, error) {
	id, err := machine.GetMachineID()
	if err != nil {
		return nil, err
	}

	return &Agent{
		config:    cfg,
		apiClient: api.NewClient(cfg),
		machineID: id,
	}, nil
}

func (a *Agent) Run(ctx context.Context) error {
	logger.Log.Info("starting AgentInferno service",
		zap.String("machine_id", a.machineID),
		zap.String("backend_url", a.config.BackendURL),
	)

	// 1. Initial Registration
	a.mustRegister(ctx)

	// 2. Heartbeat loop
	ticker := time.NewTicker(time.Duration(a.config.HeartbeatInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("stopping service loop")
			return nil
		case <-ticker.C:
			a.sendHeartbeat(ctx)
		}
	}
}

func (a *Agent) mustRegister(ctx context.Context) {
	backoff := 1 * time.Second
	for {
		stats, _ := metrics.Collect(ctx, a.machineID)
		err := a.apiClient.Register(ctx, stats)
		if err == nil {
			logger.Log.Info("agent registered successfully")
			return
		}

		logger.Log.Error("registration failed, retrying...", zap.Error(err), zap.Duration("backoff", backoff))
		
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
			backoff *= 2
			if backoff > 1*time.Minute {
				backoff = 1 * time.Minute
			}
		}
	}
}

func (a *Agent) sendHeartbeat(ctx context.Context) {
	stats, err := metrics.Collect(ctx, a.machineID)
	if err != nil {
		logger.Log.Error("failed to collect metrics", zap.Error(err))
		return
	}

	err = a.apiClient.Heartbeat(ctx, stats)
	if err != nil {
		logger.Log.Warn("heartbeat failed", zap.Error(err))
	}
}
