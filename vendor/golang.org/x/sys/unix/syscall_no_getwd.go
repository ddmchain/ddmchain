
// +build dragonfly freebsd netbsd openbsd

package unix

const ImplementsGetwd = false

func Getwd() (string, error) { return "", ENOTSUP }
