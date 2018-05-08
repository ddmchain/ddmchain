
// +build linux,cgo darwin,!ios,cgo windows,cgo

package hid

/*
#cgo CFLAGS: -I./hidapi/hidapi

#cgo linux CFLAGS: -I./libusb/libusb -DDEFAULT_VISIBILITY="" -DOS_LINUX -D_GNU_SOURCE -DPOLL_NFDS_TYPE=int
#cgo linux,!android LDFLAGS: -lrt
#cgo darwin CFLAGS: -DOS_DARWIN
#cgo darwin LDFLAGS: -framework CoreFoundation -framework IOKit
#cgo windows CFLAGS: -DOS_WINDOWS
#cgo windows LDFLAGS: -lsetupapi

#ifdef OS_LINUX
	#include <sys/poll.h>
	#include "os/threads_posix.c"
	#include "os/poll_posix.c"

	#include "os/linux_usbfs.c"
	#include "os/linux_netlink.c"

	#include "core.c"
	#include "descriptor.c"
	#include "hotplug.c"
	#include "io.c"
	#include "strerror.c"
	#include "sync.c"

	#include "hidapi/libusb/hid.c"
#elif OS_DARWIN
	#include "hidapi/mac/hid.c"
#elif OS_WINDOWS
	#include "hidapi/windows/hid.c"
#endif
*/
import "C"
import (
	"errors"
	"runtime"
	"sync"
	"unsafe"
)

var enumerateLock sync.Mutex

func init() {

	C.hid_init()
}

func Supported() bool {
	return true
}

func Enumerate(vendorID uint16, productID uint16) []DeviceInfo {
	enumerateLock.Lock()
	defer enumerateLock.Unlock()

	head := C.hid_enumerate(C.ushort(vendorID), C.ushort(productID))
	if head == nil {
		return nil
	}
	defer C.hid_free_enumeration(head)

	var infos []DeviceInfo
	for ; head != nil; head = head.next {
		info := DeviceInfo{
			Path:      C.GoString(head.path),
			VendorID:  uint16(head.vendor_id),
			ProductID: uint16(head.product_id),
			Release:   uint16(head.release_number),
			UsagePage: uint16(head.usage_page),
			Usage:     uint16(head.usage),
			Interface: int(head.interface_number),
		}
		if head.serial_number != nil {
			info.Serial, _ = wcharTToString(head.serial_number)
		}
		if head.product_string != nil {
			info.Product, _ = wcharTToString(head.product_string)
		}
		if head.manufacturer_string != nil {
			info.Manufacturer, _ = wcharTToString(head.manufacturer_string)
		}
		infos = append(infos, info)
	}
	return infos
}

func (info DeviceInfo) Open() (*Device, error) {
	path := C.CString(info.Path)
	defer C.free(unsafe.Pointer(path))

	device := C.hid_open_path(path)
	if device == nil {
		return nil, errors.New("hidapi: failed to open device")
	}
	return &Device{
		DeviceInfo: info,
		device:     device,
	}, nil
}

type Device struct {
	DeviceInfo 

	device *C.hid_device 
	lock   sync.Mutex
}

func (dev *Device) Close() {
	dev.lock.Lock()
	defer dev.lock.Unlock()

	if dev.device != nil {
		C.hid_close(dev.device)
		dev.device = nil
	}
}

func (dev *Device) Write(b []byte) (int, error) {

	if len(b) == 0 {
		return 0, nil
	}

	dev.lock.Lock()
	device := dev.device
	dev.lock.Unlock()

	if device == nil {
		return 0, ErrDeviceClosed
	}

	var report []byte
	if runtime.GOOS == "windows" {
		report = append([]byte{0x00}, b...)
	} else {
		report = b
	}

	written := int(C.hid_write(device, (*C.uchar)(&report[0]), C.size_t(len(report))))
	if written == -1 {

		dev.lock.Lock()
		device = dev.device
		dev.lock.Unlock()

		if device == nil {
			return 0, ErrDeviceClosed
		}

		message := C.hid_error(device)
		if message == nil {
			return 0, errors.New("hidapi: unknown failure")
		}
		failure, _ := wcharTToString(message)
		return 0, errors.New("hidapi: " + failure)
	}
	return written, nil
}

func (dev *Device) Read(b []byte) (int, error) {

	if len(b) == 0 {
		return 0, nil
	}

	dev.lock.Lock()
	device := dev.device
	dev.lock.Unlock()

	if device == nil {
		return 0, ErrDeviceClosed
	}

	read := int(C.hid_read(device, (*C.uchar)(&b[0]), C.size_t(len(b))))
	if read == -1 {

		dev.lock.Lock()
		device = dev.device
		dev.lock.Unlock()

		if device == nil {
			return 0, ErrDeviceClosed
		}

		message := C.hid_error(device)
		if message == nil {
			return 0, errors.New("hidapi: unknown failure")
		}
		failure, _ := wcharTToString(message)
		return 0, errors.New("hidapi: " + failure)
	}
	return read, nil
}
