
package flock

import (
	"os"
	"path/filepath"
)

type Releaser interface {
	Release() error
}

func New(fileName string) (r Releaser, existed bool, err error) {
	if err = os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		return
	}

	_, err = os.Stat(fileName)
	existed = err == nil

	r, err = newLock(fileName)
	return
}
