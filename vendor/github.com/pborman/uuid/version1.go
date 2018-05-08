
package uuid

import (
	"encoding/binary"
)

func NewUUID() UUID {
	if nodeID == nil {
		SetNodeInterface("")
	}

	now, seq, err := GetTime()
	if err != nil {
		return nil
	}

	uuid := make([]byte, 16)

	time_low := uint32(now & 0xffffffff)
	time_mid := uint16((now >> 32) & 0xffff)
	time_hi := uint16((now >> 48) & 0x0fff)
	time_hi |= 0x1000 

	binary.BigEndian.PutUint32(uuid[0:], time_low)
	binary.BigEndian.PutUint16(uuid[4:], time_mid)
	binary.BigEndian.PutUint16(uuid[6:], time_hi)
	binary.BigEndian.PutUint16(uuid[8:], seq)
	copy(uuid[10:], nodeID)

	return uuid
}
