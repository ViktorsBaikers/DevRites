//go:build !unix

package lib

import "os/exec"

func configureBoundedCommand(_ *exec.Cmd) {}
