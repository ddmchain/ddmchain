// +build freebsd solaris darwin

package reexec

import (
	"os/exec"
)

func Self() string {
	return naiveSelf()
}

func Command(args ...string) *exec.Cmd {
	return &exec.Cmd{
		Path: Self(),
		Args: args,
	}
}
