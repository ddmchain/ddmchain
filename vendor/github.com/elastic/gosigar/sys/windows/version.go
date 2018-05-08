// +build windows

package windows

import (
	"fmt"
	"syscall"
)

type Version struct {
	Major int
	Minor int
	Build int
}

func GetWindowsVersion() Version {

	ver, err := syscall.GetVersion()
	if err != nil {

		panic(fmt.Errorf("GetVersion failed: %v", err))
	}

	return Version{
		Major: int(ver & 0xFF),
		Minor: int(ver >> 8 & 0xFF),
		Build: int(ver >> 16),
	}
}

func (v Version) IsWindowsVistaOrGreater() bool {

	return v.Major >= 6 && v.Minor >= 0
}
