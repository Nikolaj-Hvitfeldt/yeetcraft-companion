//go:build !windows

package logwatcher

import (
	"fmt"
	"os"
	"syscall"
)

func encodeFileRef(file *os.File) (string, error) {
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("stat combat log file identity: %w", err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", fmt.Errorf("read combat log file identity: unsupported file stat")
	}
	return fmt.Sprintf("unix:%d:%d", stat.Dev, stat.Ino), nil
}
