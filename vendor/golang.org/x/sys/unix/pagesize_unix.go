
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package unix

import "syscall"

func Getpagesize() int {
	return syscall.Getpagesize()
}
