package healthy

import (
	"sync/atomic"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol"
	types2 "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

const (
	// MaxQuery means maximum of peers to query.
	MaxQuery = 50
)

var log = log15.New("module", "protocol.healthy")

//Protocol ....
type Protocol struct {
	*protocol.P2PEnv       //
	fallBehind       int64 //      ，          0
}

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

//InitProtocol initials the protocol.
func InitProtocol(env *protocol.P2PEnv) {
	p := Protocol{
		P2PEnv:     env,
		fallBehind: 1<<63 - 1,
	}
	p.Host.SetStreamHandler(protocol.IsSync, protocol.HandlerWithRW(p.handleStreamIsSync))
	p.Host.SetStreamHandler(protocol.IsHealthy, protocol.HandlerWithRW(p.handleStreamIsHealthy))
	p.Host.SetStreamHandler(protocol.GetLastHeader, protocol.HandlerWithRW(p.handleStreamLastHeader))

	//          ，          。
	go func() {
		ticker1 := time.NewTicker(types2.CheckHealthyInterval)
		for {
			select {
			case <-ticker1.C:
				p.updateFallBehind()
			case <-p.Ctx.Done():
				return
			}
		}
	}()

}

// handleStreamIsSync
func (p *Protocol) handleStreamIsSync(_ *types.P2PRequest, res *types.P2PResponse) error {
	maxHeight := p.queryMaxHeight()
	if maxHeight == -1 {
		return types2.ErrUnknown
	}
	header, err := p.getLastHeaderFromBlockChain()
	if err != nil {
		log.Error("handleStreamIsSync", "getLastHeaderFromBlockchain error", err)
		return types2.ErrUnknown
	}

	var isSync bool
	//
	if header.Height >= maxHeight {
		isSync = true
	}
	res.Response = &types.P2PResponse_Reply{
		Reply: &types.Reply{
			IsOk: isSync,
		},
	}

	atomic.StoreInt64(&p.fallBehind, maxHeight-header.Height)
	return nil
}

// handleStreamIsHealthy      ，
func (p *Protocol) handleStreamIsHealthy(req *types.P2PRequest, res *types.P2PResponse) error {
	maxFallBehind := req.Request.(*types.P2PRequest_HealthyHeight).HealthyHeight

	var isHealthy bool
	if atomic.LoadInt64(&p.fallBehind) <= maxFallBehind {
		isHealthy = true
	}
	res.Response = &types.P2PResponse_Reply{
		Reply: &types.Reply{
			IsOk: isHealthy,
		},
	}
	log.Info("handleStreamIsHealthy", "isHealthy", isHealthy)
	return nil
}

func (p *Protocol) handleStreamLastHeader(_ *types.P2PRequest, res *types.P2PResponse) error {
	header, err := p.getLastHeaderFromBlockChain()
	if err != nil {
		return err
	}
	res.Response = &types.P2PResponse_LastHeader{
		LastHeader: header,
	}
	return nil
}
