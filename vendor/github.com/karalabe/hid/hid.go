
package hid

import "errors"

var ErrDeviceClosed = errors.New("hid: device closed")

var ErrUnsupportedPlatform = errors.New("hid: unsupported platform")

type DeviceInfo struct {
	Path         string 
	VendorID     uint16 
	ProductID    uint16 
	Release      uint16 
	Serial       string 
	Manufacturer string 
	Product      string 
	UsagePage    uint16 
	Usage        uint16 

	Interface int
}
