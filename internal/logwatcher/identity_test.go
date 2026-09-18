package logwatcher

import (
	"io/fs"
	"testing"
	"time"
)

type stubFileInfo struct {
	size    int64
	name    string
	modTime time.Time
}

func (s stubFileInfo) Name() string       { return s.name }
func (s stubFileInfo) Size() int64        { return s.size }
func (s stubFileInfo) Mode() fs.FileMode  { return 0o644 }
func (s stubFileInfo) ModTime() time.Time { return s.modTime }
func (s stubFileInfo) IsDir() bool        { return false }
func (s stubFileInfo) Sys() any           { return nil }

func TestIsReplacementInodeReuseWithShorterDifferentHead(t *testing.T) {
	before := fingerprint([]byte("before\n"))
	after := fingerprint([]byte("after\n"))
	committed := Identity{DeviceInode: "unix:1:1", HeadFingerprint: before}
	current := Identity{DeviceInode: "unix:1:1", HeadFingerprint: after}

	if !isReplacement(committed, current, nil, stubFileInfo{size: 6}, 7) {
		t.Fatal("unlink+create with inode reuse must count as replacement")
	}
}

func TestIsReplacementAppendDoesNotCountAsReplacement(t *testing.T) {
	head := fingerprint([]byte("before\npartial"))
	grown := fingerprint([]byte("before\npartial-more"))
	committed := Identity{DeviceInode: "unix:1:1", HeadFingerprint: head}
	current := Identity{DeviceInode: "unix:1:1", HeadFingerprint: grown}

	if isReplacement(committed, current, nil, stubFileInfo{size: 20}, 7) {
		t.Fatal("append must not count as replacement")
	}
}

func TestIsReplacementDifferentInode(t *testing.T) {
	head := fingerprint([]byte("before\n"))
	committed := Identity{DeviceInode: "unix:1:1", HeadFingerprint: head}
	current := Identity{DeviceInode: "unix:1:2", HeadFingerprint: head}

	if !isReplacement(committed, current, nil, stubFileInfo{size: 20}, 7) {
		t.Fatal("different inode must count as replacement")
	}
}

func TestIsReplacementEmptyDeviceInodeIsNotReplacement(t *testing.T) {
	head := fingerprint([]byte("after\n"))
	committed := Identity{HeadFingerprint: head}
	current := Identity{DeviceInode: "unix:1:1", HeadFingerprint: head}

	if isReplacement(committed, current, nil, stubFileInfo{size: 6}, 7) {
		t.Fatal("unset committed identity must not count as replacement")
	}
}
