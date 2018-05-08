
// +build !linux,!darwin,!windows ios !cgo

package hid

func Supported() bool {
	return false
}

func Enumerate(vendorID uint16, productID uint16) []DeviceInfo {
	return nil
}

type Device struct {
	DeviceInfo 
}

func (info DeviceInfo) Open() (*Device, error) {
	return nil, ErrUnsupportedPlatform
}

func (dev *Device) Close() {}

func (dev *Device) Write(b []byte) (int, error) {
	return 0, ErrUnsupportedPlatform
}

func (dev *Device) Read(b []byte) (int, error) {
	return 0, ErrUnsupportedPlatform
}
