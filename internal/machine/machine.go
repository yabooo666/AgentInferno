package machine

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/host"
)

const idFilePath = "/var/lib/agentinferno/machine_id"

func GetMachineID() (string, error) {
	// 1. Try to read from our persistent storage
	data, err := os.ReadFile(idFilePath)
	if err == nil {
		id := strings.TrimSpace(string(data))
		if _, err := uuid.Parse(id); err == nil {
			return id, nil
		}
	}

	// 2. Generate new one if missing or invalid
	id := uuid.New().String()

	// Ensure directory exists
	dir := filepath.Dir(idFilePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// If we can't create /var/lib/agentinferno, fallback to current dir for development
		idFilePathDev := "machine_id"
		_ = os.WriteFile(idFilePathDev, []byte(id), 0644)
		return id, nil
	}

	err = os.WriteFile(idFilePath, []byte(id), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to persist machine ID: %w", err)
	}

	return id, nil
}

func GetFingerprint() string {
	info, _ := host.Info()
	raw := fmt.Sprintf("%s-%s-%s-%s", info.Hostname, info.OS, runtime.GOARCH, info.Platform)
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", hash)
}

func GetOSInfo() string {
	info, _ := host.Info()
	return fmt.Sprintf("%s %s (%s)", info.Platform, info.PlatformVersion, info.KernelVersion)
}

func GetHostname() string {
	h, _ := os.Hostname()
	return h
}
