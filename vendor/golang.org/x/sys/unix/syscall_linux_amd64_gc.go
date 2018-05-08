
// +build amd64,linux
// +build !gccgo

package unix

import "syscall"

//go:noescape
func gettimeofday(tv *Timeval) (err syscall.Errno)
