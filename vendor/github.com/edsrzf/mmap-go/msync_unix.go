
// +build darwin dragonfly freebsd linux openbsd solaris

package mmap

import (
	"syscall"
)

const _SYS_MSYNC = syscall.SYS_MSYNC
const _MS_SYNC = syscall.MS_SYNC
