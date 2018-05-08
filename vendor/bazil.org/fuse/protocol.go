package fuse

import (
	"fmt"
)

type Protocol struct {
	Major uint32
	Minor uint32
}

func (p Protocol) String() string {
	return fmt.Sprintf("%d.%d", p.Major, p.Minor)
}

func (a Protocol) LT(b Protocol) bool {
	return a.Major < b.Major ||
		(a.Major == b.Major && a.Minor < b.Minor)
}

func (a Protocol) GE(b Protocol) bool {
	return a.Major > b.Major ||
		(a.Major == b.Major && a.Minor >= b.Minor)
}

func (a Protocol) is79() bool {
	return a.GE(Protocol{7, 9})
}

func (a Protocol) HasAttrBlockSize() bool {
	return a.is79()
}

func (a Protocol) HasReadWriteFlags() bool {
	return a.is79()
}

func (a Protocol) HasGetattrFlags() bool {
	return a.is79()
}

func (a Protocol) is710() bool {
	return a.GE(Protocol{7, 10})
}

func (a Protocol) HasOpenNonSeekable() bool {
	return a.is710()
}

func (a Protocol) is712() bool {
	return a.GE(Protocol{7, 12})
}

func (a Protocol) HasUmask() bool {
	return a.is712()
}

func (a Protocol) HasInvalidate() bool {
	return a.is712()
}
