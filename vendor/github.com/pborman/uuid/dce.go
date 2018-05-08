
package uuid

import (
	"encoding/binary"
	"fmt"
	"os"
)

type Domain byte

const (
	Person = Domain(0)
	Group  = Domain(1)
	Org    = Domain(2)
)

func NewDCESecurity(domain Domain, id uint32) UUID {
	uuid := NewUUID()
	if uuid != nil {
		uuid[6] = (uuid[6] & 0x0f) | 0x20 
		uuid[9] = byte(domain)
		binary.BigEndian.PutUint32(uuid[0:], id)
	}
	return uuid
}

func NewDCEPerson() UUID {
	return NewDCESecurity(Person, uint32(os.Getuid()))
}

func NewDCEGroup() UUID {
	return NewDCESecurity(Group, uint32(os.Getgid()))
}

func (uuid UUID) Domain() (Domain, bool) {
	if v, _ := uuid.Version(); v != 2 {
		return 0, false
	}
	return Domain(uuid[9]), true
}

func (uuid UUID) Id() (uint32, bool) {
	if v, _ := uuid.Version(); v != 2 {
		return 0, false
	}
	return binary.BigEndian.Uint32(uuid[0:4]), true
}

func (d Domain) String() string {
	switch d {
	case Person:
		return "Person"
	case Group:
		return "Group"
	case Org:
		return "Org"
	}
	return fmt.Sprintf("Domain%d", int(d))
}
