
package whisperv2

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/general/hexutil"
	"github.com/ddmchain/go-ddmchain/black"
)

type PublicWhisperAPI struct {
	w *Whisper

	messagesMu sync.RWMutex
	messages   map[hexutil.Uint]*whisperFilter
}

type whisperOfflineError struct{}

func (e *whisperOfflineError) Error() string {
	return "whisper is offline"
}

var whisperOffLineErr = new(whisperOfflineError)

func NewPublicWhisperAPI(w *Whisper) *PublicWhisperAPI {
	return &PublicWhisperAPI{w: w, messages: make(map[hexutil.Uint]*whisperFilter)}
}

func (s *PublicWhisperAPI) Version() (hexutil.Uint, error) {
	if s.w == nil {
		return 0, whisperOffLineErr
	}
	return hexutil.Uint(s.w.Version()), nil
}

func (s *PublicWhisperAPI) HasIdentity(identity string) (bool, error) {
	if s.w == nil {
		return false, whisperOffLineErr
	}
	return s.w.HasIdentity(crypto.ToECDSAPub(common.FromHex(identity))), nil
}

func (s *PublicWhisperAPI) NewIdentity() (string, error) {
	if s.w == nil {
		return "", whisperOffLineErr
	}

	identity := s.w.NewIdentity()
	return common.ToHex(crypto.FromECDSAPub(&identity.PublicKey)), nil
}

type NewFilterArgs struct {
	To     string
	From   string
	Topics [][][]byte
}

func (s *PublicWhisperAPI) NewFilter(args NewFilterArgs) (hexutil.Uint, error) {
	if s.w == nil {
		return 0, whisperOffLineErr
	}

	var id hexutil.Uint
	filter := Filter{
		To:     crypto.ToECDSAPub(common.FromHex(args.To)),
		From:   crypto.ToECDSAPub(common.FromHex(args.From)),
		Topics: NewFilterTopics(args.Topics...),
		Fn: func(message *Message) {
			wmsg := NewWhisperMessage(message)
			s.messagesMu.RLock() 
			defer s.messagesMu.RUnlock()
			if s.messages[id] != nil {
				s.messages[id].insert(wmsg)
			}
		},
	}
	id = hexutil.Uint(s.w.Watch(filter))

	s.messagesMu.Lock()
	s.messages[id] = newWhisperFilter(id, s.w)
	s.messagesMu.Unlock()

	return id, nil
}

func (s *PublicWhisperAPI) GetFilterChanges(filterId hexutil.Uint) []WhisperMessage {
	s.messagesMu.RLock()
	defer s.messagesMu.RUnlock()

	if s.messages[filterId] != nil {
		if changes := s.messages[filterId].retrieve(); changes != nil {
			return changes
		}
	}
	return returnWhisperMessages(nil)
}

func (s *PublicWhisperAPI) UninstallFilter(filterId hexutil.Uint) bool {
	s.messagesMu.Lock()
	defer s.messagesMu.Unlock()

	if _, ok := s.messages[filterId]; ok {
		delete(s.messages, filterId)
		return true
	}
	return false
}

func (s *PublicWhisperAPI) GetMessages(filterId hexutil.Uint) []WhisperMessage {

	s.messagesMu.RLock()
	defer s.messagesMu.RUnlock()

	var messages []*Message
	if s.messages[filterId] != nil {
		messages = s.messages[filterId].messages()
	}

	return returnWhisperMessages(messages)
}

func returnWhisperMessages(messages []*Message) []WhisperMessage {
	msgs := make([]WhisperMessage, len(messages))
	for i, msg := range messages {
		msgs[i] = NewWhisperMessage(msg)
	}
	return msgs
}

type PostArgs struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Topics   [][]byte `json:"topics"`
	Payload  string   `json:"payload"`
	Priority int64    `json:"priority"`
	TTL      int64    `json:"ttl"`
}

func (s *PublicWhisperAPI) Post(args PostArgs) (bool, error) {
	if s.w == nil {
		return false, whisperOffLineErr
	}

	message := NewMessage(common.FromHex(args.Payload))
	options := Options{
		To:     crypto.ToECDSAPub(common.FromHex(args.To)),
		TTL:    time.Duration(args.TTL) * time.Second,
		Topics: NewTopics(args.Topics...),
	}

	if len(args.From) > 0 {
		if key := s.w.GetIdentity(crypto.ToECDSAPub(common.FromHex(args.From))); key != nil {
			options.From = key
		} else {
			return false, fmt.Errorf("unknown identity to send from: %s", args.From)
		}
	}

	pow := time.Duration(args.Priority) * time.Millisecond
	envelope, err := message.Wrap(pow, options)
	if err != nil {
		return false, err
	}

	return true, s.w.Send(envelope)
}

type WhisperMessage struct {
	ref *Message

	Payload string `json:"payload"`
	To      string `json:"to"`
	From    string `json:"from"`
	Sent    int64  `json:"sent"`
	TTL     int64  `json:"ttl"`
	Hash    string `json:"hash"`
}

func (args *PostArgs) UnmarshalJSON(data []byte) (err error) {
	var obj struct {
		From     string         `json:"from"`
		To       string         `json:"to"`
		Topics   []string       `json:"topics"`
		Payload  string         `json:"payload"`
		Priority hexutil.Uint64 `json:"priority"`
		TTL      hexutil.Uint64 `json:"ttl"`
	}

	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	args.From = obj.From
	args.To = obj.To
	args.Payload = obj.Payload
	args.Priority = int64(obj.Priority) 
	args.TTL = int64(obj.TTL)           

	args.Topics = make([][]byte, len(obj.Topics))
	for i, topic := range obj.Topics {
		args.Topics[i] = common.FromHex(topic)
	}

	return nil
}

func (args *NewFilterArgs) UnmarshalJSON(b []byte) (err error) {

	var obj struct {
		To     interface{} `json:"to"`
		From   interface{} `json:"from"`
		Topics interface{} `json:"topics"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}

	if obj.To == nil {
		args.To = ""
	} else {
		argstr, ok := obj.To.(string)
		if !ok {
			return fmt.Errorf("to is not a string")
		}
		args.To = argstr
	}
	if obj.From == nil {
		args.From = ""
	} else {
		argstr, ok := obj.From.(string)
		if !ok {
			return fmt.Errorf("from is not a string")
		}
		args.From = argstr
	}

	if obj.Topics != nil {

		list, ok := obj.Topics.([]interface{})
		if !ok {
			return fmt.Errorf("topics is not an array")
		}

		topics := make([][]string, len(list))
		for idx, field := range list {
			switch value := field.(type) {
			case nil:
				topics[idx] = []string{}

			case string:
				topics[idx] = []string{value}

			case []interface{}:
				topics[idx] = make([]string, len(value))
				for i, nested := range value {
					switch value := nested.(type) {
					case nil:
						topics[idx][i] = ""

					case string:
						topics[idx][i] = value

					default:
						return fmt.Errorf("topic[%d][%d] is not a string", idx, i)
					}
				}
			default:
				return fmt.Errorf("topic[%d] not a string or array", idx)
			}
		}

		topicsDecoded := make([][][]byte, len(topics))
		for i, condition := range topics {
			topicsDecoded[i] = make([][]byte, len(condition))
			for j, topic := range condition {
				topicsDecoded[i][j] = common.FromHex(topic)
			}
		}

		args.Topics = topicsDecoded
	}
	return nil
}

type whisperFilter struct {
	id  hexutil.Uint 
	ref *Whisper     

	cache  []WhisperMessage         
	skip   map[common.Hash]struct{} 
	update time.Time                

	lock sync.RWMutex 
}

func (w *whisperFilter) messages() []*Message {
	w.lock.Lock()
	defer w.lock.Unlock()

	w.cache = nil
	w.update = time.Now()

	w.skip = make(map[common.Hash]struct{})
	messages := w.ref.Messages(int(w.id))
	for _, message := range messages {
		w.skip[message.Hash] = struct{}{}
	}
	return messages
}

func (w *whisperFilter) insert(messages ...WhisperMessage) {
	w.lock.Lock()
	defer w.lock.Unlock()

	for _, message := range messages {
		if _, ok := w.skip[message.ref.Hash]; !ok {
			w.cache = append(w.cache, messages...)
		}
	}
}

func (w *whisperFilter) retrieve() (messages []WhisperMessage) {
	w.lock.Lock()
	defer w.lock.Unlock()

	messages, w.cache = w.cache, nil
	w.update = time.Now()

	return
}

func newWhisperFilter(id hexutil.Uint, ref *Whisper) *whisperFilter {
	return &whisperFilter{
		id:     id,
		ref:    ref,
		update: time.Now(),
		skip:   make(map[common.Hash]struct{}),
	}
}

func NewWhisperMessage(message *Message) WhisperMessage {
	return WhisperMessage{
		ref: message,

		Payload: common.ToHex(message.Payload),
		From:    common.ToHex(crypto.FromECDSAPub(message.Recover())),
		To:      common.ToHex(crypto.FromECDSAPub(message.To)),
		Sent:    message.Sent.Unix(),
		TTL:     int64(message.TTL / time.Second),
		Hash:    common.ToHex(message.Hash.Bytes()),
	}
}
