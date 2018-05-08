
package ddm

import (
	"github.com/ddmchain/go-ddmchain/rhythm"
	"github.com/ddmchain/go-ddmchain/discover"
)

var (
	propTxnInPacketsMeter     = metrics.NewMeter("ddm/prop/txns/in/packets")
	propTxnInTrafficMeter     = metrics.NewMeter("ddm/prop/txns/in/traffic")
	propTxnOutPacketsMeter    = metrics.NewMeter("ddm/prop/txns/out/packets")
	propTxnOutTrafficMeter    = metrics.NewMeter("ddm/prop/txns/out/traffic")
	propHashInPacketsMeter    = metrics.NewMeter("ddm/prop/hashes/in/packets")
	propHashInTrafficMeter    = metrics.NewMeter("ddm/prop/hashes/in/traffic")
	propHashOutPacketsMeter   = metrics.NewMeter("ddm/prop/hashes/out/packets")
	propHashOutTrafficMeter   = metrics.NewMeter("ddm/prop/hashes/out/traffic")
	propBlockInPacketsMeter   = metrics.NewMeter("ddm/prop/blocks/in/packets")
	propBlockInTrafficMeter   = metrics.NewMeter("ddm/prop/blocks/in/traffic")
	propBlockOutPacketsMeter  = metrics.NewMeter("ddm/prop/blocks/out/packets")
	propBlockOutTrafficMeter  = metrics.NewMeter("ddm/prop/blocks/out/traffic")
	reqHeaderInPacketsMeter   = metrics.NewMeter("ddm/req/headers/in/packets")
	reqHeaderInTrafficMeter   = metrics.NewMeter("ddm/req/headers/in/traffic")
	reqHeaderOutPacketsMeter  = metrics.NewMeter("ddm/req/headers/out/packets")
	reqHeaderOutTrafficMeter  = metrics.NewMeter("ddm/req/headers/out/traffic")
	reqBodyInPacketsMeter     = metrics.NewMeter("ddm/req/bodies/in/packets")
	reqBodyInTrafficMeter     = metrics.NewMeter("ddm/req/bodies/in/traffic")
	reqBodyOutPacketsMeter    = metrics.NewMeter("ddm/req/bodies/out/packets")
	reqBodyOutTrafficMeter    = metrics.NewMeter("ddm/req/bodies/out/traffic")
	reqStateInPacketsMeter    = metrics.NewMeter("ddm/req/states/in/packets")
	reqStateInTrafficMeter    = metrics.NewMeter("ddm/req/states/in/traffic")
	reqStateOutPacketsMeter   = metrics.NewMeter("ddm/req/states/out/packets")
	reqStateOutTrafficMeter   = metrics.NewMeter("ddm/req/states/out/traffic")
	reqReceiptInPacketsMeter  = metrics.NewMeter("ddm/req/receipts/in/packets")
	reqReceiptInTrafficMeter  = metrics.NewMeter("ddm/req/receipts/in/traffic")
	reqReceiptOutPacketsMeter = metrics.NewMeter("ddm/req/receipts/out/packets")
	reqReceiptOutTrafficMeter = metrics.NewMeter("ddm/req/receipts/out/traffic")
	miscInPacketsMeter        = metrics.NewMeter("ddm/misc/in/packets")
	miscInTrafficMeter        = metrics.NewMeter("ddm/misc/in/traffic")
	miscOutPacketsMeter       = metrics.NewMeter("ddm/misc/out/packets")
	miscOutTrafficMeter       = metrics.NewMeter("ddm/misc/out/traffic")
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
	switch {
	case msg.Code == BlockHeadersMsg:
		packets, traffic = reqHeaderInPacketsMeter, reqHeaderInTrafficMeter
	case msg.Code == BlockBodiesMsg:
		packets, traffic = reqBodyInPacketsMeter, reqBodyInTrafficMeter

	case rw.version >= ddm63 && msg.Code == NodeDataMsg:
		packets, traffic = reqStateInPacketsMeter, reqStateInTrafficMeter
	case rw.version >= ddm63 && msg.Code == ReceiptsMsg:
		packets, traffic = reqReceiptInPacketsMeter, reqReceiptInTrafficMeter

	case msg.Code == NewBlockHashesMsg:
		packets, traffic = propHashInPacketsMeter, propHashInTrafficMeter
	case msg.Code == NewBlockMsg:
		packets, traffic = propBlockInPacketsMeter, propBlockInTrafficMeter
	case msg.Code == TxMsg:
		packets, traffic = propTxnInPacketsMeter, propTxnInTrafficMeter
	}
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	return msg, err
}

func (rw *meteredMsgReadWriter) WriteMsg(msg p2p.Msg) error {

	packets, traffic := miscOutPacketsMeter, miscOutTrafficMeter
	switch {
	case msg.Code == BlockHeadersMsg:
		packets, traffic = reqHeaderOutPacketsMeter, reqHeaderOutTrafficMeter
	case msg.Code == BlockBodiesMsg:
		packets, traffic = reqBodyOutPacketsMeter, reqBodyOutTrafficMeter

	case rw.version >= ddm63 && msg.Code == NodeDataMsg:
		packets, traffic = reqStateOutPacketsMeter, reqStateOutTrafficMeter
	case rw.version >= ddm63 && msg.Code == ReceiptsMsg:
		packets, traffic = reqReceiptOutPacketsMeter, reqReceiptOutTrafficMeter

	case msg.Code == NewBlockHashesMsg:
		packets, traffic = propHashOutPacketsMeter, propHashOutTrafficMeter
	case msg.Code == NewBlockMsg:
		packets, traffic = propBlockOutPacketsMeter, propBlockOutTrafficMeter
	case msg.Code == TxMsg:
		packets, traffic = propTxnOutPacketsMeter, propTxnOutTrafficMeter
	}
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	return rw.MsgReadWriter.WriteMsg(msg)
}
