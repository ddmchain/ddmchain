// +build linux

package reexec

import (
	"os/exec"
	"syscall"
)

func Self() string {
	return "/proc/self/exe"
}

func Command(args ...string) *exec.Cmd {
	return &exec.Cmd{
		Path: Self(),
		Args: args,
		SysProcAttr: &syscall.SysProcAttr{
			Pdeathsig: syscall.SIGTERM,
		},
	}
}
