package download

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	//protobufCodec "github.com/multiformats/go-multicodec/protobuf"

	"github.com/D-PlatformOperatingSystem/dpos/common/log/log15"

	core "github.com/libp2p/go-libp2p-core"

	prototypes "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol/types"
	uuid "github.com/google/uuid"

	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

var (
	log = log15.New("module", "p2p.download")
)

func init() {
	prototypes.RegisterProtocol(protoTypeID, &downloadProtol{})
	prototypes.RegisterStreamHandler(protoTypeID, downloadBlockReq, &downloadHander{})
}

const (
	protoTypeID      = "DownloadProtocolType"
	downloadBlockReq = "/dplatformos/downloadBlockReq/1.0.0"
)

//type Istream
type downloadProtol struct {
	*prototypes.BaseProtocol
}

func (d *downloadProtol) InitProtocol(env *prototypes.P2PEnv) {
	d.P2PEnv = env
	//
	prototypes.RegisterEventHandler(types.EventFetchBlocks, d.handleEvent)

}

type downloadHander struct {
	*prototypes.BaseStreamHandler
}

//Handle
func (d *downloadHander) Handle(stream core.Stream) {
	protocol := d.GetProtocol().(*downloadProtol)

	//
	if stream.Protocol() == downloadBlockReq {
		var data types.MessageGetBlocksReq
		err := prototypes.ReadStream(&data, stream)
		if err != nil {
			log.Error("Handle", "err", err)
			return
		}
		recvData := data.Message
		protocol.onReq(data.GetMessageData().GetId(), recvData, stream)
	}

}

func (d *downloadProtol) processReq(id string, message *types.P2PGetBlocks) (*types.MessageGetBlocksResp, error) {

	//            256
	if message.GetEndHeight()-message.GetStartHeight() > 256 || message.GetEndHeight() < message.GetStartHeight() {
		return nil, errors.New("param error")

	}
	//

	reqblock := &types.ReqBlocks{Start: message.GetStartHeight(), End: message.GetEndHeight()}

	resp, err := d.QueryBlockChain(types.EventGetBlocks, reqblock)
	if err != nil {
		log.Error("sendToBlockChain", "Error", err.Error())
		return nil, err
	}

	blockDetails := resp.(*types.BlockDetails)
	var p2pInvData = make([]*types.InvData, 0)
	var invdata types.InvData
	for _, item := range blockDetails.Items {
		invdata.Reset()
		invdata.Ty = 2 //2 block,1 tx
		invdata.Value = &types.InvData_Block{Block: item.Block}
		p2pInvData = append(p2pInvData, &invdata)
	}

	peerID := d.GetHost().ID()
	pubkey, _ := d.GetHost().Peerstore().PubKey(peerID).Bytes()
	blocksResp := &types.MessageGetBlocksResp{MessageData: d.NewMessageCommon(id, peerID.Pretty(), pubkey, false),
		Message: &types.InvDatas{Items: p2pInvData}}

	return blocksResp, nil

}

func (d *downloadProtol) onReq(id string, message *types.P2PGetBlocks, s core.Stream) {
	log.Debug("OnReq", "start", message.GetStartHeight(), "end", message.GetStartHeight(), "remoteId", s.Conn().RemotePeer().String(), "id", id)

	blockdata, err := d.processReq(id, message)
	if err != nil {
		log.Error("processReq", "err", err, "pid", s.Conn().RemotePeer().String())
		return
	}
	err = prototypes.WriteStream(blockdata, s)
	if err != nil {
		log.Error("WriteStream", "err", err, "pid", s.Conn().RemotePeer().String())
		return
	}

	log.Debug("OnReq", "Send Block Height+++++++", blockdata.Message.GetItems()[0].GetBlock().GetHeight(), "send  to", s.Conn().RemotePeer().String())

}

//GetBlocks     dplatformos blockchain
func (d *downloadProtol) handleEvent(msg *queue.Message) {

	req := msg.GetData().(*types.ReqBlocks)
	if req.GetStart() > req.GetEnd() {
		log.Error("handleEvent", "download start", req.GetStart(), "download end", req.GetEnd())
		msg.Reply(d.GetQueueClient().NewMessage("blockchain", types.EventReply, types.Reply{Msg: []byte("start>end")}))
		return
	}
	pids := req.GetPid()
	if len(pids) == 0 { //     pidlist      block header
		log.Debug("GetBlocks:pid is nil")
		msg.Reply(d.GetQueueClient().NewMessage("blockchain", types.EventReply, types.Reply{Msg: []byte("no pid")}))
		return
	}

	msg.Reply(d.GetQueueClient().NewMessage("blockchain", types.EventReply, types.Reply{IsOk: true, Msg: []byte("ok")}))
	var taskID = uuid.New().String() + "+" + fmt.Sprintf("%d-%d", req.GetStart(), req.GetEnd())

	log.Debug("handleEvent", "taskID", taskID, "download start", req.GetStart(), "download end", req.GetEnd(), "pids", pids)

	//
	jobS := d.initJob(pids, taskID)
	log.Debug("handleEvent", "jobs", jobS)
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var maxgoroutin int32
	var reDownload = make(map[string]interface{})
	var startTime = time.Now().UnixNano()

	for height := req.GetStart(); height <= req.GetEnd(); height++ {
		wg.Add(1)
	Wait:
		if atomic.LoadInt32(&maxgoroutin) > 50 {
			time.Sleep(time.Millisecond * 200)
			goto Wait
		}
		atomic.AddInt32(&maxgoroutin, 1)
		go func(blockheight int64, tasks tasks) {
			err := d.downloadBlock(blockheight, tasks)
			if err != nil {
				mutex.Lock()
				defer mutex.Unlock()

				if err == d.Ctx.Err() {
					log.Error("syncDownloadBlock", "err", err.Error())
					return
				}

				log.Error("syncDownloadBlock", "downloadBlock err", err.Error())
				v, ok := reDownload[taskID]
				if ok {
					faildJob := v.(map[int64]bool)
					faildJob[blockheight] = false
					reDownload[taskID] = faildJob

				} else {
					var faildJob = make(map[int64]bool)
					faildJob[blockheight] = false
					reDownload[taskID] = faildJob

				}
			}
			wg.Done()
			atomic.AddInt32(&maxgoroutin, -1)

		}(height, jobS)

	}

	wg.Wait()
	d.checkTask(taskID, pids, reDownload)
	log.Debug("Download Job Complete!", "TaskID++++++++++++++", taskID,
		"cost time", fmt.Sprintf("cost time:%d ms", (time.Now().UnixNano()-startTime)/1e6),
		"from", pids)

}

func (d *downloadProtol) downloadBlock(blockheight int64, tasks tasks) error {

	var retryCount uint
	tasks.Sort() //           ，
ReDownload:
	select {
	case <-d.Ctx.Done():
		log.Warn("downloadBlock", "process", "done")
		return d.Ctx.Err()
	default:
		break
	}

	if tasks.Size() == 0 {
		return errors.New("no peer for download")
	}

	retryCount++
	if retryCount > 50 {
		return errors.New("beyound max try count 50")
	}

	task := d.availbTask(tasks, blockheight)
	if task == nil {
		time.Sleep(time.Millisecond * 400)
		goto ReDownload
	}

	var downloadStart = time.Now().UnixNano()

	getblocks := &types.P2PGetBlocks{StartHeight: blockheight, EndHeight: blockheight,
		Version: 0}

	peerID := d.GetHost().ID()
	pubkey, _ := d.GetHost().Peerstore().PubKey(peerID).Bytes()
	blockReq := &types.MessageGetBlocksReq{MessageData: d.NewMessageCommon(uuid.New().String(), peerID.Pretty(), pubkey, false),
		Message: getblocks}

	req := &prototypes.StreamRequest{
		PeerID: task.Pid,
		Data:   blockReq,
		MsgID:  []core.ProtocolID{downloadBlockReq},
	}
	var resp types.MessageGetBlocksResp
	err := d.SendRecvPeer(req, &resp)
	if err != nil {
		log.Error("handleEvent", "SendRecvPeer", err, "pid", task.Pid)
		d.releaseJob(task)
		tasks = tasks.Remove(task)
		goto ReDownload
	}

	block := resp.GetMessage().GetItems()[0].GetBlock()
	remotePid := task.Pid.Pretty()
	costTime := (time.Now().UnixNano() - downloadStart) / 1e6

	log.Debug("download+++++", "from", remotePid, "blockheight", block.GetHeight(),
		"blockSize (bytes)", block.Size(), "costTime ms", costTime)

	client := d.GetQueueClient()
	newmsg := client.NewMessage("blockchain", types.EventSyncBlock, &types.BlockPid{Pid: remotePid, Block: block}) //       )
	client.SendTimeout(newmsg, false, 10*time.Second)
	d.releaseJob(task)

	return nil

}
