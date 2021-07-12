/*
 * Copyright D-Platform Corp. 2018 All Rights Reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

package strategy

//const
const (
	//   main.go
	CpftMainGo = `package main

import (
	_ "github.com/D-PlatformOperatingSystem/dpos/system"
	_ "${PROJECTPATH}/plugin"

	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util/cli"
)

func main() {
	types.S("cfg.${PROJECTNAME}", ${PROJECTNAME})
	cli.RunDplatformOS("${PROJECTNAME}")
}
`

	//           xxx.toml
	CpftCfgToml = `
Title="${PROJECTNAME}"
FixTime=false

[log]
#     ，  debug(dbug)/info/warn/error(eror)/crit
loglevel = "debug"
logConsoleLevel = "info"
#      ，    ，                
logFile = "logs/${PROJECTNAME}.log"
#           （  ： ）
maxFileSize = 300
#              
maxBackups = 100
#            （  ： ）
maxAge = 28
#              （    UTC  ）
localTime = true
#           （     gz）
compress = true
#             
callerFile = false
#         
callerFunction = false

[blockchain]
dbPath="datadir"
dbCache=64
batchsync=false
isRecordBlockSequence=false
enableTxQuickIndex=false

[p2p]
seeds=[]
isSeed=false
innerSeedEnable=true
useGithub=true
innerBounds=300
dbPath="datadir/addrbook"
dbCache=4
grpcLogFile="grpc.log"

[rpc]
jrpcBindAddr="localhost:28803"
grpcBindAddr="localhost:28804"
whitelist=["127.0.0.1"]
jrpcFuncWhitelist=["*"]
grpcFuncWhitelist=["*"]

[mempool]
maxTxNumPerAccount=100

[store]
dbPath="datadir/mavltree"
dbCache=128
enableMavlPrefix=false
enableMVCC=false
enableMavlPrune=false
pruneHeight=10000

[wallet]
dbPath="wallet"
dbCache=16

[wallet.sub.ticket]
minerdisable=false
minerwhitelist=["*"]

[exec]
enableStat=false
enableMVCC=false

[exec.sub.token]
saveTokenTxList=false
`

	CpftRunmainBlock = `package main

var ${PROJECTNAME} = `

	//              xxx.go
	//        package main
	CpftRunMain = `TestNet=false
[blockchain]
defCacheSize=128
maxFetchBlockNum=128
timeoutSeconds=5
batchBlockNum=128
driver="leveldb"
isStrongConsistency=false
singleMode=false
[p2p]
enable=true
serverStart=true
msgCacheSize=10240
driver="leveldb"
[mempool]
poolCacheSize=102400
minTxFeeRate=10000
[consensus]
name="ticket"
minerstart=true
genesisBlockTime=1514533394
genesis="16ERTbYtKKQ64wMthAY9J4La4nAiidG45A"
[mver.consensus]
fundKeyAddr = "1OmCaA6un9CFYEWPnGRi7uyXY1KhTJxJEc"
coinReward = 18
coinDevFund = 12
ticketPrice = 10000
powLimitBits = "0x1f00ffff"
retargetAdjustmentFactor = 4
futureBlockTime = 15
ticketFrozenTime = 43200
ticketWithdrawTime = 172800
ticketMinerWaitTime = 7200
maxTxNumber = 1500
targetTimespan = 2160
targetTimePerBlock = 15
[consensus.sub.ticket]
genesisBlockTime=1526486816
[[consensus.sub.ticket.genesis]]
minerAddr="184wj4nsgVxKyz2NhM3Yb5RK5Ap6AFRFq2"
returnAddr="1FB8L3DykVF7Y78bRfUrRcMZwesKue7CyR"
count=3000
[[consensus.sub.ticket.genesis]]
minerAddr="1M4ns1eGHdHak3SNc2UTQB75vnXyJQd91s"
returnAddr="1Lw6QLShKVbKM6QvMaCQwTh5Uhmy4644CG"
count=3000
[[consensus.sub.ticket.genesis]]
minerAddr="19ozyoUGPAQ9spsFiz9CJfnUCFeszpaFuF"
returnAddr="1PSYYfCbtSeT1vJTvSKmQvhz8y6VhtddWi"
count=3000
[[consensus.sub.ticket.genesis]]
minerAddr="1MoEnCDhXZ6Qv5fNDGYoW6MVEBTBK62HP2"
returnAddr="1BG9ZoKtgU5bhKLpcsrncZ6xdzFCgjrZud"
count=3000
[[consensus.sub.ticket.genesis]]
minerAddr="1FjKcxY7vpmMH6iB5kxNYLvJkdkQXddfrp"
returnAddr="1G7s64AgX1ySDcUdSW5vDa8jTYQMnZktCd"
count=3000
[[consensus.sub.ticket.genesis]]
minerAddr="12T8QfKbCRBhQdRfnAfFbUwdnH7TDTm4vx"
returnAddr="1FiDC6XWHLe7fDMhof8wJ3dty24f6aKKjK"
count=3000
[[consensus.sub.ticket.genesis]]
minerAddr="1bgg6HwQretMiVcSWvayPRvVtwjyKfz1J"
returnAddr="1AMvuuQ7V7FPQ4hkvHQdgNWy8wVL4d4hmp"
count=3000
[[consensus.sub.ticket.genesis]]
minerAddr="1EwkKd9iU1pL2ZwmRAC5RrBoqFD1aMrQ2"
returnAddr="1ExRRLoJXa8LzXdNxnJvBkVNZpVw3QWMi4"
count=3000
[[consensus.sub.ticket.genesis]]
minerAddr="1HFUhgxarjC7JLru1FLEY6aJbQvCSL58CB"
returnAddr="1KNGHukhbBnbWWnMYxu1C7YMoCj45Z3amm"
count=3000
[[consensus.sub.ticket.genesis]]
minerAddr="1C9M1RCv2e9b4GThN9ddBgyxAphqMgh5zq"
returnAddr="1AH9HRd4WBJ824h9PP1jYpvRZ4BSA4oN6Y"
count=4733
[store]
name="mavl"
driver="leveldb"
[wallet]
minFee=100000
driver="leveldb"
signType="secp256k1"
[exec]

[exec.sub.token]
#      ，         
tokenApprs = []
[exec.sub.relay]
genesis="16ERTbYtKKQ64wMthAY9J4La4nAiidG45A"
[exec.sub.manage]
superManager=[
"1OmCaA6un9CFYEWPnGRi7uyXY1KhTJxJEc",
]
#      fork,   dplatformos      
#        
[fork.system]
ForkChainParamV1= 0
ForkCheckTxDup=0
ForkBlockHash= 1
ForkMinerTime= 0
ForkTransferExec= 100000
ForkExecKey= 200000
ForkTxGroup= 200000
ForkResetTx0= 200000
ForkWithdraw= 200000
ForkExecRollback= 450000
ForkCheckBlockTime=1200000
ForkMultiSignAddress=1298600
ForkTxHeight= -1
ForkTxGroupPara= -1
ForkChainParamV2= -1
ForkBase58AddressCheck=1800000
ForkTicketFundAddrV1=-1
ForkRootHash=1
[fork.sub.coins]
Enable=0
[fork.sub.ticket]
Enable=0
ForkTicketId = 1200000
[fork.sub.retrieve]
Enable=0
ForkRetrive=0
[fork.sub.hashlock]
Enable=0
[fork.sub.manage]
Enable=0
ForkManageExec=100000
[fork.sub.token]
Enable=0
ForkTokenBlackList= 0
ForkBadTokenSymbol= 0
ForkTokenPrice= 300000
[fork.sub.trade]
Enable=0
ForkTradeBuyLimit= 0
ForkTradeAsset= -1
`

	//     Makefile      Makefile
	CpftMakefile = `
DplatformOS=github.com/D-PlatformOperatingSystem/dpos
DPOS_PATH=vendor/${DPOS}
all: vendor proto build

build:
	go build -i -o ${PROJECTNAME}
	go build -i -o ${PROJECTNAME}-cli ${PROJECTPATH}/cli

vendor:
	make update
	make updatevendor

proto: 
	cd ${GOPATH}/src/${PROJECTPATH}/plugin/dapp/${EXECNAME}/proto && sh create_protobuf.sh

update:
	go get -u -v github.com/kardianos/govendor
	rm -rf ${DPOS_PATH}
	git clone --depth 1 -b master https://${DPOS}.git ${DPOS_PATH}
	rm -rf vendor/${DPOS}/.git
	rm -rf vendor/${DPOS}/vendor/github.com/apache/thrift/tutorial/erl/
	cp -Rf vendor/${DPOS}/vendor/* vendor/
	rm -rf vendor/${DPOS}/vendor
	govendor init
	go build -i -o tool github.com/D-PlatformOperatingSystem/plugin/vendor/github.com/D-PlatformOperatingSystem/dpos/cmd/tools
	./tool import --path "plugin" --packname "${PROJECTPATH}/plugin" --conf "plugin/plugin.toml"

updatevendor:
	govendor add +e
	govendor fetch -v +m

clean:
	@rm -rf vendor
	@rm -rf datadir
	@rm -rf logs
	@rm -rf wallet
	@rm -rf grpc.log
	@rm -rf ${PROJECTNAME}
	@rm -rf ${PROJECTNAME}-cli
	@rm -rf tool
	@rm -rf plugin/init.go
	@rm -rf plugin/consensus/init
	@rm -rf plugin/dapp/init
	@rm -rf plugin/crypto/init
	@rm -rf plugin/store/init
	@rm -rf plugin/mempool/init
`

	//    .travis.yml
	CpftTravisYml = `
language: go

go:
  - "1.9"
  - master
`

	//    plugin/plugin.toml
	CpftPluginToml = `
# type      consensus  dapp store mempool
[dapp-ticket]
gitrepo = "github.com/D-PlatformOperatingSystem/plugin/plugin/dapp/ticket"

[consensus-ticket]
gitrepo = "github.com/D-PlatformOperatingSystem/plugin/plugin/consensus/ticket"

[dapp-retrieve]
gitrepo = "github.com/D-PlatformOperatingSystem/plugin/plugin/dapp/retrieve"

[dapp-hashlock]
gitrepo = "github.com/D-PlatformOperatingSystem/plugin/plugin/dapp/hashlock"

[dapp-token]
gitrepo = "github.com/D-PlatformOperatingSystem/plugin/plugin/dapp/token"

[dapp-trade]
gitrepo = "github.com/D-PlatformOperatingSystem/plugin/plugin/dapp/trade"

[mempool-price]
gitrepo = "github.com/D-PlatformOperatingSystem/plugin/plugin/mempool/price"

[mempool-score]
gitrepo = "github.com/D-PlatformOperatingSystem/plugin/plugin/mempool/score"
`
	//    cli/main.go
	CpftCliMain = `package main

import (
	_ "${PROJECTPATH}/plugin"
	_ "github.com/D-PlatformOperatingSystem/dpos/system"
	"github.com/D-PlatformOperatingSystem/dpos/util/cli"
)

func main() {
	cli.Run("", "")
}
`
	// plugin/dapp/xxxx/commands/cmd.go     c
	CpftDappCommands = `package commands

import (
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	return nil
}`

	// plugin/dapp/xxxx/plugin.go
	CpftDappPlugin = `package ${PROJECTNAME}

import (
	"github.com/D-PlatformOperatingSystem/dpos/pluginmgr"
	"${PROJECTPATH}/plugin/dapp/${PROJECTNAME}/commands"
	"${PROJECTPATH}/plugin/dapp/${PROJECTNAME}/executor"
	"${PROJECTPATH}/plugin/dapp/${PROJECTNAME}/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     types.${EXECNAME_FB}X,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.Cmd,
		RPC:      nil,
	})
}
`

	// plugin/dapp/xxxx/executor/xxxx.go
	CpftDappExec = `package executor

import (
	log "github.com/inconshreveable/log15"
	drivers "github.com/D-PlatformOperatingSystem/dpos/system/dapp"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

var clog = log.New("module", "execs.${EXECNAME}")
var driverName = "${EXECNAME}"

func init() {
	ety := types.LoadExecutorType(driverName)
	if ety != nil {
		ety.InitFuncList(types.ListMethod(&${CLASSNAME}{}))
	}
}

func Init(name string, sub []byte) {
	ety := types.LoadExecutorType(driverName)
	if ety != nil {
		ety.InitFuncList(types.ListMethod(&${CLASSNAME}{}))
	}

	clog.Debug("register ${EXECNAME} execer")
	drivers.Register(GetName(), new${CLASSNAME}, types.GetDappFork(driverName, "Enable"))
}

func GetName() string {
	return new${CLASSNAME}().GetName()
}

type ${CLASSNAME} struct {
	drivers.DriverBase
}

func new${CLASSNAME}() drivers.Driver {
	n := &${CLASSNAME}{}
	n.SetChild(n)
	n.SetIsFree(true)
	n.SetExecutorType(types.LoadExecutorType(driverName))
	return n
}

func (this *${CLASSNAME}) GetDriverName() string {
	return driverName
}

func (this *${CLASSNAME}) CheckTx(tx *types.Transaction, index int) error {
	return nil
}
`
	// plugin/dapp/xxxx/proto/create_protobuf.sh
	CpftDappCreatepb = `#!/bin/sh
protoc --go_out=plugins=grpc:../types ./*.proto --proto_path=. 
`

	// plugin/dapp/xxxx/proto/Makefile
	CpftDappMakefile = `all:
	sh ./create_protobuf.sh
`

	// plugin/dapp/xxxx/proto/xxxx.proto
	CpftDappProto = `syntax = "proto3";
package types;

message ${ACTIONNAME} {
    int32 ty = 1;
	oneof value {
		${ACTIONNAME}None none = 2;
	}
}

message ${ACTIONNAME}None {
}
`

	// plugin/dapp/xxxx/types/types.go     cd
	CpftDappTypefile = `package types

import (
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

var (
	${EXECNAME_FB}X      = "${EXECNAME}"
	Execer${EXECNAME_FB} = []byte(${EXECNAME_FB}X)
	actionName  = map[string]int32{}
	logmap = map[int64]*types.LogInfo{}
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, Execer${EXECNAME_FB})
	types.RegistorExecutor("${EXECNAME}", NewType())

	types.RegisterDappFork(${EXECNAME_FB}X, "Enable", 0)
}

type ${CLASSTYPENAME} struct {
	types.ExecTypeBase
}

func NewType() *${CLASSTYPENAME} {
	c := &${CLASSTYPENAME}{}
	c.SetChild(c)
	return c
}

func (coins *${CLASSTYPENAME}) GetPayload() types.Message {
	return &${ACTIONNAME}{}
}

func (coins *${CLASSTYPENAME}) GetName() string {
	return ${EXECNAME_FB}X
}

func (coins *${CLASSTYPENAME}) GetLogMap() map[int64]*types.LogInfo {
	return logmap
}

func (c *${CLASSTYPENAME}) GetTypeMap() map[string]int32 {
	return actionName
}
`
)
