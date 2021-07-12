package p2pstore

import (
	"context"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol"
	types2 "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

func (p *Protocol) republish() {
	//    p2pstore    chunk,    republish
	if p.SubConfig.IsFullNode {
		return
	}
	m := make(map[string]LocalChunkInfo)
	p.localChunkInfoMutex.RLock()
	for k, v := range p.localChunkInfo {
		m[k] = v
	}
	p.localChunkInfoMutex.RUnlock()
	invertedIndex := make(map[peer.ID][]*types.ChunkInfoMsg)
	for hash, info := range m {
		if time.Since(info.Time) > types2.ExpiredTime {
			log.Info("republish deleteChunkBlock", "hash", hash, "start", info.Start)
			if err := p.deleteChunkBlock(info.ChunkHash); err != nil {
				log.Error("republish deleteChunkBlock error", "hash", hash, "error", err)
			}
			continue
		}
		if time.Since(info.Time) > types2.RefreshInterval*11/10 {
			continue
		}
		log.Info("local chunk", "hash", hash, "start", info.Start)
		peers := p.healthyRoutingTable.NearestPeers(genDHTID(info.ChunkHash), Backup-1)
		for _, pid := range peers {
			invertedIndex[pid] = append(invertedIndex[pid], info.ChunkInfoMsg)
		}
	}

	log.Info("republish", "invertedIndex length", len(invertedIndex))
	for pid, infos := range invertedIndex {
		log.Info("republish", "pid", pid, "info len", len(infos))
		if err := p.storeChunksOnPeer(pid, infos...); err != nil {
			log.Error("republish", "storeChunksOnPeer error", err, "pid", pid)
		}
	}
	log.Info("republish ok")
}

//       *BackUp-1*        ，      Backup
func (p *Protocol) notifyStoreChunk(req *types.ChunkInfoMsg) {
	peers := p.healthyRoutingTable.NearestPeers(genDHTID(req.ChunkHash), Backup-1)
	for _, pid := range peers {
		err := p.storeChunksOnPeer(pid, req)
		if err != nil {
			log.Error("notifyStoreChunk", "peer id", pid, "error", err)
		}
	}
}

func (p *Protocol) storeChunksOnPeer(pid peer.ID, reqs ...*types.ChunkInfoMsg) error {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Minute)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, pid, protocol.StoreChunk)
	if err != nil {
		log.Error("new stream error when store chunk", "peer id", pid, "error", err)
		return err
	}
	defer protocol.CloseStream(stream)
	msg := types.P2PRequest{}
	msg.Request = &types.P2PRequest_ChunkInfoList{
		ChunkInfoList: &types.ChunkInfoList{
			Items: reqs,
		}}
	return protocol.SignAndWriteStream(&msg, stream)
}
