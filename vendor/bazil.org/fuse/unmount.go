package fuse

func Unmount(dir string) error {
	return unmount(dir)
}
