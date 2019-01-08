
// +build none

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/black"
	"github.com/ddmchain/go-ddmchain/signger"
	"github.com/ddmchain/go-ddmchain/discover"
	"github.com/ddmchain/go-ddmchain/discover/nat"
	"github.com/ddmchain/go-ddmchain/secret"
)

func main() {
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.InfoLevel))

	key, err := crypto.GenerateKey()
	if err != nil {
		fmt.Printf("Failed to generate peer key: %v.\n", err)
		os.Exit(-1)
	}
	name := common.MakeName("whisper-go", "1.0")
	shh := whisper.New()

	server := p2p.Server{
		PrivateKey: key,
		MaxPeers:   10,
		Name:       name,
		Protocols:  []p2p.Protocol{shh.Protocol()},
		ListenAddr: ":30300",
		NAT:        nat.Any(),
	}
	fmt.Println("Starting DDMchain peer...")
	if err := server.Start(); err != nil {
		fmt.Printf("Failed to start DDMchain peer: %v.\n", err)
		os.Exit(1)
	}

	payload := fmt.Sprintf("Hello world, this is %v. In case you're wondering, the time is %v", name, time.Now())
	if err := selfSend(shh, []byte(payload)); err != nil {
		fmt.Printf("Failed to self message: %v.\n", err)
		os.Exit(-1)
	}
}

func selfSend(shh *whisper.Whisper, payload []byte) error {
	ok := make(chan struct{})

	id := shh.NewIdentity()
	shh.Watch(whisper.Filter{
		To: &id.PublicKey,
		Fn: func(msg *whisper.Message) {
			fmt.Printf("Message received: %s, signed with 0x%x.\n", string(msg.Payload), msg.Signature)
			close(ok)
		},
	})

	msg := whisper.NewMessage(payload)
	envelope, err := msg.Wrap(whisper.DefaultPoW, whisper.Options{
		From: id,
		To:   &id.PublicKey,
		TTL:  whisper.DefaultTTL,
	})
	if err != nil {
		return fmt.Errorf("failed to seal message: %v", err)
	}

	if err := shh.Send(envelope); err != nil {
		return fmt.Errorf("failed to send self-message: %v", err)
	}
	select {
	case <-ok:
	case <-time.After(time.Second):
		return fmt.Errorf("failed to receive message in time")
	}
	return nil
}
