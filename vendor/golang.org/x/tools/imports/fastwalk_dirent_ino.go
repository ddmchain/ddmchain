
// +build linux,!appengine darwin

package imports

import "syscall"

func direntInode(dirent *syscall.Dirent) uint64 {
	return uint64(dirent.Ino)
}
