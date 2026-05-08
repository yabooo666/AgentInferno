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
	// Check all known paths for an existing persistent ID
	paths := []string{
		idFilePath,
		"machine_id",
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			id := strings.TrimSpace(string(data))
			if _, err := uuid.Parse(id); err == nil {
				return id, nil
			}
		}
	}

	// Generate new UUID
	id := uuid.New().String()

	// Try primary production path first
	dir := filepath.Dir(idFilePath)
	if err := os.MkdirAll(dir, 0700); err == nil {
		// 0600 = owner read/write only — prevents other users from reading the ID
		if err := os.WriteFile(idFilePath, []byte(id), 0600); err == nil {
			return id, nil
		}
	}

	// Fallback to local directory (development/Windows)
	if err := os.WriteFile("machine_id", []byte(id), 0600); err != nil {
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
