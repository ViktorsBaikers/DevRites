//go:build !unix

package lib

import "os"

func openSequenceBrief(path string) (*os.File, error) {
	return os.Open(path)
}
