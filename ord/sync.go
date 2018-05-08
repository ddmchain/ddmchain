
package les

import (
	"context"
	"time"

	"github.com/ddmchain/go-ddmchain/major"
	"github.com/ddmchain/go-ddmchain/ddm/downloader"
	"github.com/ddmchain/go-ddmchain/thin"
)

const (

	minDesiredPeerCount = 5 
)

func (pm *ProtocolManager) syncer() {

	defer pm.downloader.Terminate()

	for {
		select {
		case <-pm.newPeerCh:

		case <-pm.noMorePeers:
			return
		}
	}
}

func (pm *ProtocolManager) needToSync(peerHead blockInfo) bool {
	head := pm.blockchain.CurrentHeader()
	currentTd := core.GetTd(pm.chainDb, head.Hash(), head.Number.Uint64())
	return currentTd != nil && peerHead.Td.Cmp(currentTd) > 0
}

func (pm *ProtocolManager) synchronise(peer *peer) {

	if peer == nil {
		return
	}

	if !pm.needToSync(peer.headBlockInfo()) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	pm.blockchain.(*light.LightChain).SyncCht(ctx)
	pm.downloader.Synchronise(peer.id, peer.Head(), peer.Td(), downloader.LightSync)
}
