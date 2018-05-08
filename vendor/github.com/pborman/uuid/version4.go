
package uuid

func NewRandom() UUID {
	uuid := make([]byte, 16)
	randomBits([]byte(uuid))
	uuid[6] = (uuid[6] & 0x0f) | 0x40 
	uuid[8] = (uuid[8] & 0x3f) | 0x80 
	return uuid
}
