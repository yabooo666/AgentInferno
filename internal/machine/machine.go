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
	// List of potential paths to check (Production first, then Development fallback)
	paths := []string{
		idFilePath,
		"machine_id",
	}

	// 1. Try to read from any existing persistent storage
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			id := strings.TrimSpace(string(data))
			if _, err := uuid.Parse(id); err == nil {
				return id, nil
			}
		}
	}

	// 2. Generate new one if none found
	id := uuid.New().String()

	// 3. Try to persist to the primary production path
	dir := filepath.Dir(idFilePath)
	if err := os.MkdirAll(dir, 0755); err == nil {
		err = os.WriteFile(idFilePath, []byte(id), 0644)
		if err == nil {
			return id, nil
		}
	}

	// 4. Final fallback to local directory (Development/Windows)
	err := os.WriteFile("machine_id", []byte(id), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to persist machine ID to local fallback: %w", err)
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
