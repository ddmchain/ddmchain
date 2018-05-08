
package mmap

import (
	"errors"
	"os"
	"reflect"
	"unsafe"
)

const (

	RDONLY = 0

	RDWR = 1 << iota

	COPY

	EXEC
)

const (

	ANON = 1 << iota
)

type MMap []byte

func Map(f *os.File, prot, flags int) (MMap, error) {
	return MapRegion(f, -1, prot, flags, 0)
}

func MapRegion(f *os.File, length int, prot, flags int, offset int64) (MMap, error) {
	var fd uintptr
	if flags&ANON == 0 {
		fd = uintptr(f.Fd())
		if length < 0 {
			fi, err := f.Stat()
			if err != nil {
				return nil, err
			}
			length = int(fi.Size())
		}
	} else {
		if length <= 0 {
			return nil, errors.New("anonymous mapping requires non-zero length")
		}
		fd = ^uintptr(0)
	}
	return mmap(length, uintptr(prot), uintptr(flags), fd, offset)
}

func (m *MMap) header() *reflect.SliceHeader {
	return (*reflect.SliceHeader)(unsafe.Pointer(m))
}

func (m MMap) Lock() error {
	dh := m.header()
	return lock(dh.Data, uintptr(dh.Len))
}

func (m MMap) Unlock() error {
	dh := m.header()
	return unlock(dh.Data, uintptr(dh.Len))
}

func (m MMap) Flush() error {
	dh := m.header()
	return flush(dh.Data, uintptr(dh.Len))
}

func (m *MMap) Unmap() error {
	dh := m.header()
	err := unmap(dh.Data, uintptr(dh.Len))
	*m = nil
	return err
}
