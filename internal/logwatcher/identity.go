package logwatcher

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

const defaultHeadSize = 4096

// Identity captures stable file identity and a head fingerprint for replacement detection.
type Identity struct {
	Path            string
	DeviceInode     string
	HeadFingerprint string
}

// StorageKey returns the persisted file_identity value for storage.
func (id Identity) StorageKey() string {
	return id.DeviceInode
}

// ResolveIdentity reads the file head and encodes device/inode identity.
func ResolveIdentity(path string) (Identity, os.FileInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return Identity{}, nil, fmt.Errorf("open combat log: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return Identity{}, nil, fmt.Errorf("stat combat log: %w", err)
	}

	deviceInode, err := encodeFileRef(file)
	if err != nil {
		return Identity{}, nil, err
	}

	head, err := readHead(file, info.Size())
	if err != nil {
		return Identity{}, nil, err
	}

	return Identity{
		Path:            path,
		DeviceInode:     deviceInode,
		HeadFingerprint: fingerprint(head),
	}, info, nil
}

func readHead(file *os.File, size int64) ([]byte, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek combat log head: %w", err)
	}
	n := int64(defaultHeadSize)
	if size < n {
		n = size
	}
	if n == 0 {
		return nil, nil
	}
	buf := make([]byte, n)
	read, err := io.ReadFull(file, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("read combat log head: %w", err)
	}
	return buf[:read], nil
}

func fingerprint(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func isReplacement(committed Identity, current Identity, lastInfo, currentInfo os.FileInfo, committedOffset int64) bool {
	if committed.DeviceInode == "" {
		return false
	}
	if committed.DeviceInode != current.DeviceInode {
		return true
	}
	if lastInfo != nil && currentInfo != nil && !os.SameFile(lastInfo, currentInfo) {
		return true
	}
	if committed.HeadFingerprint == "" || current.HeadFingerprint == "" {
		return false
	}
	if committed.HeadFingerprint == current.HeadFingerprint {
		return false
	}
	// Growing appends change the head of files smaller than the fingerprint
	// window. Once we have consumed bytes, only treat a *shorter* file with a
	// different head as replacement — Linux often reuses the inode after
	// unlink+create, which otherwise looks like an in-place truncation.
	if committedOffset > 0 {
		return currentInfo != nil && currentInfo.Size() < committedOffset
	}
	return true
}
