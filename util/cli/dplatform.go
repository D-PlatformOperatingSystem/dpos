// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build go1.8

// package cli RunDplatformOS         ，
//          。
//         ，
//             。
//rpc

package cli

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof" //
	"os"
	"path/filepath"
	"runtime"

	"github.com/D-PlatformOperatingSystem/dpos/p2p"

	"github.com/D-PlatformOperatingSystem/dpos/metrics"

	"time"

	"github.com/D-PlatformOperatingSystem/dpos/blockchain"
	"github.com/D-PlatformOperatingSystem/dpos/util"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/common/limits"
	clog "github.com/D-PlatformOperatingSystem/dpos/common/log"
	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/common/version"
	"github.com/D-PlatformOperatingSystem/dpos/consensus"
	"github.com/D-PlatformOperatingSystem/dpos/executor"
	"github.com/D-PlatformOperatingSystem/dpos/mempool"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/rpc"
	"github.com/D-PlatformOperatingSystem/dpos/store"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/wallet"
	"google.golang.org/grpc/grpclog"
)

var (
	cpuNum      = runtime.NumCPU()
	configPath  = flag.String("f", "", "configfile")
	datadir     = flag.String("datadir", "", "data dir of dplatformos, include logs and datas")
	versionCmd  = flag.Bool("v", false, "version")
	fixtime     = flag.Bool("fixtime", false, "fix time")
	waitPid     = flag.Bool("waitpid", false, "p2p stuck until seed save info wallet & wallet unlock")
	rollback    = flag.Int64("rollback", 0, "rollback block")
	save        = flag.Bool("save", false, "rollback save temporary block")
	importFile  = flag.String("import", "", "import block file name")
	exportTitle = flag.String("export", "", "export block title name")
	fileDir     = flag.String("filedir", "", "import/export block file dir,defalut current path")
	startHeight = flag.Int64("startheight", 0, "export block start height")
)

//RunDplatformOS : run DplatformOS
func RunDplatformOS(name, defCfg string) {
	flag.Parse()
	if *versionCmd {
		fmt.Println(version.GetVersion())
		return
	}
	if *configPath == "" {
		if name == "" {
			*configPath = "dplatformos.toml"
		} else {
			*configPath = name + ".toml"
		}
	}
	d, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	log.Info("current dir:", "dir", d)
	err = os.Chdir(pwd())
	if err != nil {
		panic(err)
	}
	d, err = os.Getwd()
	if err != nil {
		panic(err)
	}
	log.Info("current dir:", "dir", d)
	err = limits.SetLimits()
	if err != nil {
		panic(err)
	}
	//set config: dom   dplatformos.toml
	dplatformosCfg := types.NewDplatformOSConfig(types.MergeCfg(types.ReadFile(*configPath), defCfg))
	cfg := dplatformosCfg.GetModuleConfig()
	if *datadir != "" {
		util.ResetDatadir(cfg, *datadir)
	}
	if *fixtime {
		cfg.FixTime = *fixtime
	}
	if *waitPid {
		cfg.P2P.WaitPid = *waitPid
	}

	if cfg.FixTime {
		go fixtimeRoutine()
	}
	//compare minFee in wallet, mempool, exec
	//set file log
	clog.SetFileLog(cfg.Log)
	//set grpc log
	f, err := createFile(cfg.P2P.GrpcLogFile)
	if err != nil {
		glogv2 := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout)
		grpclog.SetLoggerV2(glogv2)
	} else {
		glogv2 := grpclog.NewLoggerV2WithVerbosity(f, f, f, 10)
		grpclog.SetLoggerV2(glogv2)
	}
	//set watching
	t := time.Tick(10 * time.Second)
	go func() {
		for range t {
			watching()
		}
	}()
	//set pprof
	go func() {
		if cfg.Pprof != nil {
			err := http.ListenAndServe(cfg.Pprof.ListenAddr, nil)
			if err != nil {
				log.Info("ListenAndServe", "listen addr", cfg.Pprof.ListenAddr, "err", err)
			}
		} else {
			err := http.ListenAndServe("localhost:6060", nil)
			if err != nil {
				log.Info("ListenAndServe", "listen addr localhost:6060 err", err)
			}
		}
	}()
	//set maxprocs
	runtime.GOMAXPROCS(cpuNum)
	//
	//channel, rabitmq
	version.SetLocalDBVersion(cfg.Store.LocalDBVersion)
	version.SetStoreDBVersion(cfg.Store.StoreDBVersion)
	version.SetAppVersion(cfg.Version)
	log.Info(cfg.Title + "-app:" + version.GetAppVersion() + " dplatformos:" + version.GetVersion() + " localdb:" + version.GetLocalDBVersion() + " statedb:" + version.GetStoreDBVersion())
	log.Info("loading queue")
	q := queue.New("channel")
	q.SetConfig(dplatformosCfg)

	log.Info("loading mempool module")
	mem := mempool.New(dplatformosCfg)
	mem.SetQueueClient(q.Client())

	log.Info("loading execs module")
	exec := executor.New(dplatformosCfg)
	exec.SetQueueClient(q.Client())

	log.Info("loading blockchain module")
	cfg.BlockChain.RollbackBlock = *rollback
	cfg.BlockChain.RollbackSave = *save
	chain := blockchain.New(dplatformosCfg)
	chain.SetQueueClient(q.Client())

	log.Info("loading store module")
	s := store.New(dplatformosCfg)
	s.SetQueueClient(q.Client())

	chain.Upgrade()

	log.Info("loading consensus module")
	cs := consensus.New(dplatformosCfg)
	cs.SetQueueClient(q.Client())

	//jsonrpc, grpc, channel
	rpcapi := rpc.New(dplatformosCfg)
	rpcapi.SetQueueClient(q.Client())

	log.Info("loading wallet module")
	walletm := wallet.New(dplatformosCfg)
	walletm.SetQueueClient(q.Client())

	chain.Rollbackblock()
	//  /      title
	if *importFile != "" {
		chain.ImportBlockProc(*importFile, *fileDir)
	}
	if *exportTitle != "" {
		chain.ExportBlockProc(*exportTitle, *fileDir, *startHeight)
	}
	log.Info("loading p2p module")
	var network queue.Module
	if cfg.P2P.Enable {
		network = p2p.NewP2PMgr(dplatformosCfg)
	} else {
		network = &util.MockModule{Key: "p2p"}
	}
	network.SetQueueClient(q.Client())

	health := util.NewHealthCheckServer(q.Client())
	health.Start(cfg.Health)
	metrics.StartMetrics(dplatformosCfg)
	defer func() {
		//close all module,clean some resource
		log.Info("begin close health module")
		health.Close()
		log.Info("begin close blockchain module")
		chain.Close()
		log.Info("begin close mempool module")
		mem.Close()
		log.Info("begin close P2P module")
		network.Close()
		log.Info("begin close execs module")
		exec.Close()
		log.Info("begin close store module")
		s.Close()
		log.Info("begin close consensus module")
		cs.Close()
		log.Info("begin close rpc module")
		rpcapi.Close()
		log.Info("begin close wallet module")
		walletm.Close()
		log.Info("begin close queue module")
		q.Close()

	}()
	q.Start()
}

func createFile(filename string) (*os.File, error) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func watching() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Info("info:", "NumGoroutine:", runtime.NumGoroutine())
	log.Info("info:", "Mem:", m.Sys/(1024*1024))
	log.Info("info:", "HeapAlloc:", m.HeapAlloc/(1024*1024))
}

func pwd() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	return dir
}

func fixtimeRoutine() {
	hosts := types.NtpHosts
	for i := 0; i < len(hosts); i++ {
		t, err := common.GetNtpTime(hosts[i])
		if err == nil {
			log.Info("time", "host", hosts[i], "now", t)
		} else {
			log.Error("time", "err", err)
		}
	}
	t := common.GetRealTimeRetry(hosts, 10)
	if !t.IsZero() {
		//update
		types.SetTimeDelta(int64(time.Until(t)))
		log.Info("change time", "delta", time.Until(t), "real.now", types.Now())
	}
	//        :
	ticket := time.NewTicker(time.Minute * 1)
	for range ticket.C {
		t = common.GetRealTimeRetry(hosts, 10)
		if !t.IsZero() {
			//update
			log.Info("change time", "delta", time.Until(t))
			types.SetTimeDelta(int64(time.Until(t)))
		}
	}
}
