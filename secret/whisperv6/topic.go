
package whisperv6

import (
	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/general/hexutil"
)

type TopicType [TopicLength]byte

func BytesToTopic(b []byte) (t TopicType) {
	sz := TopicLength
	if x := len(b); x < TopicLength {
		sz = x
	}
	for i := 0; i < sz; i++ {
		t[i] = b[i]
	}
	return t
}

func (t *TopicType) String() string {
	return common.ToHex(t[:])
}

func (t TopicType) MarshalText() ([]byte, error) {
	return hexutil.Bytes(t[:]).MarshalText()
}

func (t *TopicType) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Topic", input, t[:])
}
