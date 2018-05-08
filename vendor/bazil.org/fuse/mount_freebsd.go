package fuse

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

func handleMountFusefsStderr(errCh chan<- error) func(line string) (ignore bool) {
	return func(line string) (ignore bool) {
		const (
			noMountpointPrefix = `mount_fusefs: `
			noMountpointSuffix = `: No such file or directory`
		)
		if strings.HasPrefix(line, noMountpointPrefix) && strings.HasSuffix(line, noMountpointSuffix) {

			mountpoint := line[len(noMountpointPrefix) : len(line)-len(noMountpointSuffix)]
			err := &MountpointDoesNotExistError{
				Path: mountpoint,
			}
			select {
			case errCh <- err:
				return true
			default:

				return false
			}
		}

		return false
	}
}

func isBoringMountFusefsError(err error) bool {
	if err, ok := err.(*exec.ExitError); ok && err.Exited() {
		if status, ok := err.Sys().(syscall.WaitStatus); ok && status.ExitStatus() == 1 {
			return true
		}
	}
	return false
}

func mount(dir string, conf *mountConfig, ready chan<- struct{}, errp *error) (*os.File, error) {
	for k, v := range conf.options {
		if strings.Contains(k, ",") || strings.Contains(v, ",") {

			return nil, fmt.Errorf("mount options cannot contain commas on FreeBSD: %q=%q", k, v)
		}
	}

	f, err := os.OpenFile("/dev/fuse", os.O_RDWR, 0000)
	if err != nil {
		*errp = err
		return nil, err
	}

	cmd := exec.Command(
		"/sbin/mount_fusefs",
		"--safe",
		"-o", conf.getOptions(),
		"3",
		dir,
	)
	cmd.ExtraFiles = []*os.File{f}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("setting up mount_fusefs stderr: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("setting up mount_fusefs stderr: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mount_fusefs: %v", err)
	}
	helperErrCh := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(2)
	go lineLogger(&wg, "mount helper output", neverIgnoreLine, stdout)
	go lineLogger(&wg, "mount helper error", handleMountFusefsStderr(helperErrCh), stderr)
	wg.Wait()
	if err := cmd.Wait(); err != nil {

		select {
		case helperErr := <-helperErrCh:

			if !isBoringMountFusefsError(err) {
				log.Printf("mount helper failed: %v", err)
			}

			return nil, helperErr
		default:

		}
		return nil, fmt.Errorf("mount_fusefs: %v", err)
	}

	close(ready)
	return f, nil
}
