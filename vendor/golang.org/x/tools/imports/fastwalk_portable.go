
// +build appengine !linux,!darwin,!freebsd,!openbsd,!netbsd

package imports

import (
	"io/ioutil"
	"os"
)

func readDir(dirName string, fn func(dirName, entName string, typ os.FileMode) error) error {
	fis, err := ioutil.ReadDir(dirName)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		if err := fn(dirName, fi.Name(), fi.Mode()&os.ModeType); err != nil {
			return err
		}
	}
	return nil
}
