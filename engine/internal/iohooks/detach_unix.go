//go:build unix

package iohooks

import "syscall"

// detachSysProcAttr starts the refresh worker in its own session so it outlives the
// Stop hook that spawned it — the detached-background contract of refresh-indexes.
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
