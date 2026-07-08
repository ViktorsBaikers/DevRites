//go:build !unix

package iohooks

import "syscall"

// detachSysProcAttr is a no-op on platforms without setsid (e.g. Windows); the
// worker still starts, it just is not put in a new session.
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
