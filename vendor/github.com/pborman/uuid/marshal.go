
package uuid

import (
	"errors"
	"fmt"
)

func (u UUID) MarshalText() ([]byte, error) {
	if len(u) != 16 {
		return nil, nil
	}
	var js [36]byte
	encodeHex(js[:], u)
	return js[:], nil
}

func (u *UUID) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	id := Parse(string(data))
	if id == nil {
		return errors.New("invalid UUID")
	}
	*u = id
	return nil
}

func (u UUID) MarshalBinary() ([]byte, error) {
	return u[:], nil
}

func (u *UUID) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if len(data) != 16 {
		return fmt.Errorf("invalid UUID (got %d bytes)", len(data))
	}
	var id [16]byte
	copy(id[:], data)
	*u = id[:]
	return nil
}

func (u Array) MarshalText() ([]byte, error) {
	var js [36]byte
	encodeHex(js[:], u[:])
	return js[:], nil
}

func (u *Array) UnmarshalText(data []byte) error {
	id := Parse(string(data))
	if id == nil {
		return errors.New("invalid UUID")
	}
	*u = id.Array()
	return nil
}

func (u Array) MarshalBinary() ([]byte, error) {
	return u[:], nil
}

func (u *Array) UnmarshalBinary(data []byte) error {
	if len(data) != 16 {
		return fmt.Errorf("invalid UUID (got %d bytes)", len(data))
	}
	copy(u[:], data)
	return nil
}
