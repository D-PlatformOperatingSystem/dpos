package download

import (
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

// task datastruct
type tasks []*taskInfo

type taskInfo struct {
	ID      string        //         ID
	TaskNum int           //
	Pid     peer.ID       //  ID
	Index   int           //           ，       ，
	Latency time.Duration //
	mtx     sync.Mutex
}

//Len size of the Invs data
func (t tasks) Len() int {
	return len(t)
}

//Less Sort from low to high
func (t tasks) Less(a, b int) bool {
	return t[a].Latency < t[b].Latency
}

//Swap  the param
func (t tasks) Swap(a, b int) {
	t[a], t[b] = t[b], t[a]
}

func (t tasks) Remove(task *taskInfo) tasks {
	task.mtx.Lock()
	defer task.mtx.Unlock()
	if task.Index+1 > t.Size() {
		return t
	}

	t = append(t[:task.Index], t[task.Index+1:]...)
	return t
}

func (t tasks) Sort() tasks {
	sort.Sort(t)
	return t
}

func (t tasks) Size() int {
	return len(t)
}

func (d *downloadProtol) initJob(pids []string, jobID string) tasks {
	var JobPeerIds tasks
	var pIDs []peer.ID
	for _, pid := range pids {
		pID, err := peer.IDB58Decode(pid)
		if err != nil {
			log.Error("initJob", "IDB58Decode", err)
			continue
		}
		pIDs = append(pIDs, pID)
	}
	if len(pIDs) == 0 {
		pIDs = d.ConnManager.FetchConnPeers()

	}
	latency := d.GetConnsManager().GetLatencyByPeer(pIDs)
	for _, pID := range pIDs {
		if pID.Pretty() == d.GetHost().ID().Pretty() {
			continue
		}
		var job taskInfo
		job.Pid = pID
		job.ID = jobID
		var ok bool
		latency, ok := latency[job.Pid.Pretty()]
		if ok {
			if latency != 0 {
				job.Latency = latency

			}
		} else { //             ，
			job.Latency = time.Second
		}
		job.TaskNum = 0
		JobPeerIds = append(JobPeerIds, &job)
	}
	return JobPeerIds
}

func (d *downloadProtol) checkTask(taskID string, pids []string, faildJobs map[string]interface{}) {

	select {
	case <-d.Ctx.Done():
		log.Warn("checkTask", "process", "done+++++++")
		return
	default:
		break
	}
	v, ok := faildJobs[taskID]
	if !ok {
		return
	}
	defer delete(faildJobs, taskID)

	faildJob := v.(map[int64]bool)
	for blockheight := range faildJob {
		jobS := d.initJob(pids, taskID)
		log.Warn("checkTask<<<<<<<<<<", "taskID", taskID, "faildJob", blockheight)
		d.downloadBlock(blockheight, jobS)

	}
}

func (d *downloadProtol) availbTask(ts tasks, blockheight int64) *taskInfo {

	var limit int
	if len(ts) > 10 {
		limit = 20 //     10，              20
	} else {
		limit = 50 //     ，              50
	}
	for i, task := range ts {
		//check blockheight
		peerInfo := d.GetPeerInfoManager().GetPeerInfoInMin(task.Pid.Pretty())
		if peerInfo != nil {
			if peerInfo.GetHeader().GetHeight() < blockheight {
				continue
			}
		} else {
			//log.Error("CheckAvailbJob", "PeerInfoManager No ths Peer info...", task.Pid.Pretty())
			continue
		}
		task.mtx.Lock()
		if task.TaskNum < limit {
			task.TaskNum++
			task.Index = i
			log.Debug("getFreeJob", " taskNum", task.TaskNum, "latency", task.Latency, "peerid", task.Pid)
			task.mtx.Unlock()

			return task
		}
		task.mtx.Unlock()
	}

	return nil

}

func (d *downloadProtol) releaseJob(js *taskInfo) {
	js.mtx.Lock()
	defer js.mtx.Unlock()
	js.TaskNum--
	if js.TaskNum < 0 {
		js.TaskNum = 0
	}
}
