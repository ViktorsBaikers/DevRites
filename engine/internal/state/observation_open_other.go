//go:build !unix

package state

import "os"

func openObservedFile(root *os.Root, name string) (*os.File, error) {
	return root.OpenFile(name, os.O_RDONLY, 0)
}
