
package mmap

import (
	"errors"
	"os"
	"sync"
	"syscall"
)

var handleLock sync.Mutex
var handleMap = map[uintptr]syscall.Handle{}

func mmap(len int, prot, flags, hfile uintptr, off int64) ([]byte, error) {
	flProtect := uint32(syscall.PAGE_READONLY)
	dwDesiredAccess := uint32(syscall.FILE_MAP_READ)
	switch {
	case prot&COPY != 0:
		flProtect = syscall.PAGE_WRITECOPY
		dwDesiredAccess = syscall.FILE_MAP_COPY
	case prot&RDWR != 0:
		flProtect = syscall.PAGE_READWRITE
		dwDesiredAccess = syscall.FILE_MAP_WRITE
	}
	if prot&EXEC != 0 {
		flProtect <<= 4
		dwDesiredAccess |= syscall.FILE_MAP_EXECUTE
	}

	maxSizeHigh := uint32((off + int64(len)) >> 32)
	maxSizeLow := uint32((off + int64(len)) & 0xFFFFFFFF)

	h, errno := syscall.CreateFileMapping(syscall.Handle(hfile), nil, flProtect, maxSizeHigh, maxSizeLow, nil)
	if h == 0 {
		return nil, os.NewSyscallError("CreateFileMapping", errno)
	}

	fileOffsetHigh := uint32(off >> 32)
	fileOffsetLow := uint32(off & 0xFFFFFFFF)
	addr, errno := syscall.MapViewOfFile(h, dwDesiredAccess, fileOffsetHigh, fileOffsetLow, uintptr(len))
	if addr == 0 {
		return nil, os.NewSyscallError("MapViewOfFile", errno)
	}
	handleLock.Lock()
	handleMap[addr] = h
	handleLock.Unlock()

	m := MMap{}
	dh := m.header()
	dh.Data = addr
	dh.Len = len
	dh.Cap = dh.Len

	return m, nil
}

func flush(addr, len uintptr) error {
	errno := syscall.FlushViewOfFile(addr, len)
	if errno != nil {
		return os.NewSyscallError("FlushViewOfFile", errno)
	}

	handleLock.Lock()
	defer handleLock.Unlock()
	handle, ok := handleMap[addr]
	if !ok {

		return errors.New("unknown base address")
	}

	errno = syscall.FlushFileBuffers(handle)
	return os.NewSyscallError("FlushFileBuffers", errno)
}

func lock(addr, len uintptr) error {
	errno := syscall.VirtualLock(addr, len)
	return os.NewSyscallError("VirtualLock", errno)
}

func unlock(addr, len uintptr) error {
	errno := syscall.VirtualUnlock(addr, len)
	return os.NewSyscallError("VirtualUnlock", errno)
}

func unmap(addr, len uintptr) error {
	flush(addr, len)

	handleLock.Lock()
	defer handleLock.Unlock()
	err := syscall.UnmapViewOfFile(addr)
	if err != nil {
		return err
	}

	handle, ok := handleMap[addr]
	if !ok {

		return errors.New("unknown base address")
	}
	delete(handleMap, addr)

	e := syscall.CloseHandle(syscall.Handle(handle))
	return os.NewSyscallError("CloseHandle", e)
}
