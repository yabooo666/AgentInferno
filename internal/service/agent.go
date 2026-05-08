package service

import (
	"context"
	"os/exec"
	"runtime"
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
			a.processHeartbeat(ctx)
		}
	}
}

func (a *Agent) mustRegister(ctx context.Context) {
	backoff := 1 * time.Second
	for {
		stats, _ := metrics.Collect(ctx, a.machineID)
		_, err := a.apiClient.Register(ctx, stats)
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

func (a *Agent) processHeartbeat(ctx context.Context) {
	stats, err := metrics.Collect(ctx, a.machineID)
	if err != nil {
		logger.Log.Error("failed to collect metrics", zap.Error(err))
		return
	}

	resp, err := a.apiClient.Heartbeat(ctx, stats)
	if err != nil {
		logger.Log.Warn("heartbeat failed", zap.Error(err))
		return
	}

	if action, ok := resp["action"].(string); ok {
		a.handleAction(action)
	}
}

func (a *Agent) handleAction(action string) {
	switch action {
	case "reboot":
		logger.Log.Warn("received REBOOT command from backend")
		a.executeReboot()
	default:
		logger.Log.Info("received unknown action", zap.String("action", action))
	}
}

func (a *Agent) executeReboot() {
	// Securely execute ONLY the reboot command
	// For Linux: reboot
	// For Windows (dev): shutdown /r /t 0
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("shutdown", "/r", "/t", "5") // 5 sec delay for safety
	} else {
		cmd = exec.Command("reboot")
	}

	logger.Log.Info("executing system reboot...")
	if err := cmd.Run(); err != nil {
		logger.Log.Error("reboot command failed", zap.Error(err))
	}
}
