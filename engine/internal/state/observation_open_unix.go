//go:build unix

package state

import (
	"os"
	"syscall"
)

func openObservedFile(root *os.Root, name string) (*os.File, error) {
	return root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
