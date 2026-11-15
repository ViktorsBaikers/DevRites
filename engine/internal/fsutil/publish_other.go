//go:build (!linux && !darwin && !windows) || (!amd64 && !arm64)

package fsutil

import (
	"errors"
	"os"
)

func nativePublishOpen(parent *os.File, name string, directory, movable bool) (*os.File, error) {
	return nil, errors.ErrUnsupported
}

func nativePublishRename(sourceParent, stage, targetParent *os.File, from, to string) error {
	return errors.ErrUnsupported
}

func nativePublishSync(file *os.File) error {
	return errors.ErrUnsupported
}
