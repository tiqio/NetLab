package reconcile

import (
	"os"
	"path/filepath"
	"strings"
)

func DetectHostRestart(stateDir string) (bool, error) {
	current, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return false, err
	}
	currentID := strings.TrimSpace(string(current))
	path := filepath.Join(stateDir, "host-boot-id")
	previous, readErr := os.ReadFile(path)
	if readErr != nil && !os.IsNotExist(readErr) {
		return false, readErr
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, err
	}
	if err = os.WriteFile(path, []byte(currentID+"\n"), 0o600); err != nil {
		return false, err
	}
	return len(previous) > 0 && strings.TrimSpace(string(previous)) != currentID, nil
}
