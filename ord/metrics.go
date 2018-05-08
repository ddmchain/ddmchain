
package les

import (
	"github.com/ddmchain/go-ddmchain/rhythm"
	"github.com/ddmchain/go-ddmchain/discover"
)

var (

	miscInPacketsMeter  = metrics.NewMeter("les/misc/in/packets")
	miscInTrafficMeter  = metrics.NewMeter("les/misc/in/traffic")
	miscOutPacketsMeter = metrics.NewMeter("les/misc/out/packets")
	miscOutTrafficMeter = metrics.NewMeter("les/misc/out/traffic")
)

type meteredMsgReadWriter struct {
	p2p.MsgReadWriter     
	version           int 
}

func newMeteredMsgWriter(rw p2p.MsgReadWriter) p2p.MsgReadWriter {
	if !metrics.Enabled {
		return rw
	}
	return &meteredMsgReadWriter{MsgReadWriter: rw}
}

func (rw *meteredMsgReadWriter) Init(version int) {
	rw.version = version
}

func (rw *meteredMsgReadWriter) ReadMsg() (p2p.Msg, error) {

	msg, err := rw.MsgReadWriter.ReadMsg()
	if err != nil {
		return msg, err
	}

	packets, traffic := miscInPacketsMeter, miscInTrafficMeter
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	return msg, err
}

func (rw *meteredMsgReadWriter) WriteMsg(msg p2p.Msg) error {

	packets, traffic := miscOutPacketsMeter, miscOutTrafficMeter
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	return rw.MsgReadWriter.WriteMsg(msg)
}
