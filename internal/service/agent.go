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

// Allowed actions — hardcoded whitelist. No dynamic action names are accepted.
var allowedActions = map[string]bool{
	"reboot": true,
}

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
		zap.Bool("dev_mode", a.config.DevMode),
	)

	// 1. Initial Registration with retry
	a.mustRegister(ctx)

	// 2. Heartbeat loop
	ticker := time.NewTicker(time.Duration(a.config.HeartbeatInterval) * time.Second)
	defer ticker.Stop()

	heartbeatFailures := 0

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("stopping service loop")
			return nil
		case <-ticker.C:
			if err := a.processHeartbeat(ctx); err != nil {
				heartbeatFailures++
				// Exponential backoff on repeated failures (cap at 2 min)
				if heartbeatFailures > 3 {
					backoff := time.Duration(heartbeatFailures*10) * time.Second
					if backoff > 2*time.Minute {
						backoff = 2 * time.Minute
					}
					logger.Log.Warn("heartbeat failing repeatedly, backing off",
						zap.Int("failures", heartbeatFailures),
						zap.Duration("backoff", backoff),
					)
					time.Sleep(backoff)
				}
			} else {
				heartbeatFailures = 0
			}
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

		logger.Log.Error("registration failed, retrying...",
			zap.Error(err),
			zap.Duration("backoff", backoff),
		)

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

func (a *Agent) processHeartbeat(ctx context.Context) error {
	stats, err := metrics.Collect(ctx, a.machineID)
	if err != nil {
		logger.Log.Error("failed to collect metrics", zap.Error(err))
		return err
	}

	resp, err := a.apiClient.Heartbeat(ctx, stats)
	if err != nil {
		logger.Log.Warn("heartbeat failed", zap.Error(err))
		return err
	}

	// Only process actions that are HMAC-verified by the API client
	if resp.Action != "" {
		a.handleVerifiedAction(resp)
	}

	return nil
}

func (a *Agent) handleVerifiedAction(resp *api.ActionResponse) {
	// Step 1: Check whitelist — reject unknown action names entirely
	if !allowedActions[resp.Action] {
		logger.Log.Warn("received unknown/disallowed action — ignoring",
			zap.String("action", resp.Action),
		)
		return
	}

	// Step 2: Verify HMAC signature (zero-trust — backend must prove identity)
	if !a.apiClient.VerifyAction(resp) {
		logger.Log.Error("action REJECTED — HMAC verification failed",
			zap.String("action", resp.Action),
		)
		return
	}

	// Step 3: Execute only verified + whitelisted actions
	switch resp.Action {
	case "reboot":
		logger.Log.Warn("executing VERIFIED reboot command")
		a.executeReboot()
	}
}

func (a *Agent) executeReboot() {
	// This is the ONLY os/exec call in the entire agent.
	// It executes a hardcoded command — no user input is involved.
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("shutdown", "/r", "/t", "5")
	} else {
		cmd = exec.Command("/sbin/reboot")
	}

	logger.Log.Info("executing system reboot...")
	if err := cmd.Run(); err != nil {
		logger.Log.Error("reboot command failed", zap.Error(err))
	}
}
