
package whisperv2

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/general/math"
	"github.com/ddmchain/go-ddmchain/black"
	"github.com/ddmchain/go-ddmchain/black/ecies"
	"github.com/ddmchain/go-ddmchain/ptl"
)

type Envelope struct {
	Expiry uint32 
	TTL    uint32 
	Topics []Topic
	Data   []byte
	Nonce  uint32

	hash common.Hash 
}

func NewEnvelope(ttl time.Duration, topics []Topic, msg *Message) *Envelope {
	return &Envelope{
		Expiry: uint32(time.Now().Add(ttl).Unix()),
		TTL:    uint32(ttl.Seconds()),
		Topics: topics,
		Data:   msg.bytes(),
		Nonce:  0,
	}
}

func (self *Envelope) Seal(pow time.Duration) {
	d := make([]byte, 64)
	copy(d[:32], self.rlpWithoutNonce())

	finish, bestBit := time.Now().Add(pow).UnixNano(), 0
	for nonce := uint32(0); time.Now().UnixNano() < finish; {
		for i := 0; i < 1024; i++ {
			binary.BigEndian.PutUint32(d[60:], nonce)

			d := new(big.Int).SetBytes(crypto.Keccak256(d))
			firstBit := math.FirstBitSet(d)
			if firstBit > bestBit {
				self.Nonce, bestBit = nonce, firstBit
			}
			nonce++
		}
	}
}

func (self *Envelope) rlpWithoutNonce() []byte {
	enc, _ := rlp.EncodeToBytes([]interface{}{self.Expiry, self.TTL, self.Topics, self.Data})
	return enc
}

func (self *Envelope) Open(key *ecdsa.PrivateKey) (msg *Message, err error) {

	data := self.Data

	message := &Message{
		Flags: data[0],
		Sent:  time.Unix(int64(self.Expiry-self.TTL), 0),
		TTL:   time.Duration(self.TTL) * time.Second,
		Hash:  self.Hash(),
	}
	data = data[1:]

	if message.Flags&signatureFlag == signatureFlag {
		if len(data) < signatureLength {
			return nil, fmt.Errorf("unable to open envelope. First bit set but len(data) < len(signature)")
		}
		message.Signature, data = data[:signatureLength], data[signatureLength:]
	}
	message.Payload = data

	if key == nil {
		return message, nil
	}
	err = message.decrypt(key)
	switch err {
	case nil:
		return message, nil

	case ecies.ErrInvalidPublicKey: 
		return message, err

	default:
		return nil, fmt.Errorf("unable to open envelope, decrypt failed: %v", err)
	}
}

func (self *Envelope) Hash() common.Hash {
	if (self.hash == common.Hash{}) {
		enc, _ := rlp.EncodeToBytes(self)
		self.hash = crypto.Keccak256Hash(enc)
	}
	return self.hash
}

func (self *Envelope) DecodeRLP(s *rlp.Stream) error {
	raw, err := s.Raw()
	if err != nil {
		return err
	}

	type rlpenv Envelope
	if err := rlp.DecodeBytes(raw, (*rlpenv)(self)); err != nil {
		return err
	}
	self.hash = crypto.Keccak256Hash(raw)
	return nil
}
