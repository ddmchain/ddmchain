
// +build windows
// +build go1.4

package windows

import "syscall"

func Unsetenv(key string) error {

	return syscall.Unsetenv(key)
}
