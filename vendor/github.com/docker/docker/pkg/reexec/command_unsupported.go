// +build !linux,!windows,!freebsd,!solaris,!darwin

package reexec

import (
	"os/exec"
)

func Command(args ...string) *exec.Cmd {
	return nil
}
