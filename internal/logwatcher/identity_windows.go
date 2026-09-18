//go:build windows

package logwatcher

import (
	"fmt"
	"os"
	"syscall"
)

func encodeFileRef(file *os.File) (string, error) {
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(file.Fd()), &info); err != nil {
		return "", fmt.Errorf("read combat log file identity: %w", err)
	}
	index := (uint64(info.FileIndexHigh) << 32) | uint64(info.FileIndexLow)
	return fmt.Sprintf("win:%08x:%016x", info.VolumeSerialNumber, index), nil
}
