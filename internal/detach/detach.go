package detach

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogsDir returns the directory where detached agent logs are stored.
func LogsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	logsDir := filepath.Join(homeDir, ".swarm", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create logs directory: %w", err)
	}

	return logsDir, nil
}

// LogFilePath generates a log file path for a detached agent.
func LogFilePath(id string) (string, error) {
	logsDir, err := LogsDir()
	if err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.log", timestamp, id)
	return filepath.Join(logsDir, filename), nil
}
