
package whisperv6

import (
	"fmt"
	"time"
)

const (
	ProtocolVersion    = uint64(6) 
	ProtocolVersionStr = "6.0"     
	ProtocolName       = "shh"     

	statusCode           = 0   
	messagesCode         = 1   
	powRequirementCode   = 2   
	bloomFilterExCode    = 3   
	p2pRequestCode       = 126 
	p2pMessageCode       = 127 
	NumberOfMessageCodes = 128

	SizeMask      = byte(3) 
	signatureFlag = byte(4)

	TopicLength     = 4  
	signatureLength = 65 
	aesKeyLength    = 32 
	aesNonceLength  = 12 
	keyIDSize       = 32 
	bloomFilterSize = 64 
	flagsLength     = 1

	EnvelopeHeaderLength = 20

	MaxMessageSize        = uint32(10 * 1024 * 1024) 
	DefaultMaxMessageSize = uint32(1024 * 1024)
	DefaultMinimumPoW     = 0.2

	padSizeLimit      = 256 
	messageQueueLimit = 1024

	expirationCycle   = time.Second
	transmissionCycle = 300 * time.Millisecond

	DefaultTTL           = 50 
	DefaultSyncAllowance = 10 
)

type unknownVersionError uint64

func (e unknownVersionError) Error() string {
	return fmt.Sprintf("invalid envelope version %d", uint64(e))
}

type MailServer interface {
	Archive(env *Envelope)
	DeliverMail(whisperPeer *Peer, request *Envelope)
}
