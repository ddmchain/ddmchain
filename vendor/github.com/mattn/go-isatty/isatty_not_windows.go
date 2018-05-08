// +build !windows appengine

package isatty

func IsCygwinTerminal(fd uintptr) bool {
	return false
}
