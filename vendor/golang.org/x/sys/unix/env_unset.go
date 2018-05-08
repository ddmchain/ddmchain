
// +build go1.4

package unix

import "syscall"

func Unsetenv(key string) error {

	return syscall.Unsetenv(key)
}
