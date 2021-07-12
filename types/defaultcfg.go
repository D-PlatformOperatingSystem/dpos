// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

var cfgstring = `
Title="local"
TestNet=true
FixTime=false
TxHeight=false
CoinSymbol="dpos"

[log]
#     ，  debug(dbug)/info/warn/error(eror)/crit
loglevel = "debug"
logConsoleLevel = "info"
#      ，    ，                
logFile = "logs/dplatformos.log"
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
defCacheSize=128
maxFetchBlockNum=128
timeoutSeconds=5
batchBlockNum=128
driver="leveldb"
dbPath="datadir"
dbCache=64
isStrongConsistency=false
singleMode=true
batchsync=false
isRecordBlockSequence=true
isParaChain=false
enableTxQuickIndex=true
txHeight=true

#     localdb
enableReduceLocaldb=false
#       ,  false       ;                  true
disableShard=false
#                
chunkblockNum=1000
#    P2pStore     
enableFetchP2pstore=false
#              ,      
enableIfDelLocalChunk=false

enablePushSubscribe=true
maxActiveBlockNum=1024
maxActiveBlockSize=100


[p2p]
enable=false
driver="leveldb"
dbPath="datadir/addrbook"
dbCache=4
grpcLogFile="grpc.log"

[rpc]
jrpcBindAddr="localhost:0"
grpcBindAddr="localhost:0"
whitelist=["127.0.0.1"]
jrpcFuncWhitelist=["*"]
grpcFuncWhitelist=["*"]

[mempool]
name="timeline"
poolCacheSize=102400
#          ，       ，  ，   0.0001 coins
minTxFeeRate=10000
maxTxNumPerAccount=100000

[consensus]
name="solo"
minerstart=true
genesisBlockTime=1514533394
genesis="16ERTbYtKKQ64wMthAY9J4La4nAiidG45A"
minerExecs=["ticket", "autonomy"]

[mver.consensus]
fundKeyAddr = "1CQXE6TxaYCG5mADtWij4AxhZCUTpoABb3"
powLimitBits = "0x1f00ffff"
maxTxNumber = 10000

[mver.consensus.ForkChainParamV1]
maxTxNumber = 10000

[mver.consensus.ForkChainParamV2]
powLimitBits = "0x1f2fffff"

[mver.consensus.ForkTicketFundAddrV1]
fundKeyAddr = "1Ji3W12KGScCM7C2p8bg635sNkayDM8MGY"

[mver.consensus.ticket]
coinReward = 18
coinDevFund = 12
ticketPrice = 10000
retargetAdjustmentFactor = 4
futureBlockTime = 16
ticketFrozenTime = 5
ticketWithdrawTime = 10
ticketMinerWaitTime = 2
targetTimespan = 2304
targetTimePerBlock = 16

[mver.consensus.ticket.ForkChainParamV1]
targetTimespan = 288 #only for test
targetTimePerBlock = 2

[consensus.sub.para]
ParaRemoteGrpcClient="localhost:28804"
#             
startHeight=345850
#      ，   
writeBlockSeconds=2
#               ，         
emptyBlockInterval=50
#    ，             ，          ，       
authAccount=""
#                    ，         ，   2
waitBlocks4CommitMsg=2
searchHashMatchedBlockDepth=100

[consensus.sub.solo]
genesis="16ERTbYtKKQ64wMthAY9J4La4nAiidG45A"
genesisBlockTime=1514533394
waitTxMs=1

[consensus.sub.ticket]
genesisBlockTime=1514533394
[[consensus.sub.ticket.genesis]]
minerAddr="12oupcayRT7LvaC4qW4avxsTE7U41cKSio"
returnAddr="16ERTbYtKKQ64wMthAY9J4La4nAiidG45A"
count=10000

[[consensus.sub.ticket.genesis]]
minerAddr="1PUiGcbsccfxW3zuvHXZBJfznziph5miAo"
returnAddr="1EbDHAXpoiewjPLX9uqoz38HsKqMXayZrF"
count=1000

[[consensus.sub.ticket.genesis]]
minerAddr="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
returnAddr="1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa"
count=1000

[store]
name="mavl"
driver="leveldb"
dbPath="datadir/mavltree"
dbCache=128

[store.sub.mavl]
enableMavlPrefix=false
enableMVCC=false

[wallet]
minFee=100000
driver="leveldb"
dbPath="wallet"
dbCache=16
signType="secp256k1"
coinType="dpos"

[wallet.sub.ticket]
minerdisable=false
minerwhitelist=["*"]

[exec]
enableStat=false
enableMVCC=false
alias=["token1:token","token2:token","token3:token"]

[exec.sub.token]
saveTokenTxList=true
tokenApprs = [
	"1Bsg9j6gW83sShoee1fZAt9TkUjcrCgA9S",
	"1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK",
	"1LY8GFia5EiyoTodMLfkB5PHNNpXRqxhyB",
	"1GCzJDS6HbgTQ2emade7mEJGGWFfA15pS9",
	"1JYB8sxi4He5pZWHCd3Zi2nypQ4JMB6AxN",
	"12oupcayRT7LvaC4qW4avxsTE7U41cKSio",
]

[exec.sub.relay]
genesis="16ERTbYtKKQ64wMthAY9J4La4nAiidG45A"

[exec.sub.cert]
#            
enable=false
#       
cryptoPath="authdir/crypto"
#        ，  "auth_ecdsa", "auth_sm2"
signType="auth_ecdsa"

[exec.sub.manage]
superManager=[
    "1Bsg9j6gW83sShoee1fZAt9TkUjcrCgA9S", 
    "12oupcayRT7LvaC4qW4avxsTE7U41cKSio", 
    "1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK"
]
[exec.sub.autonomy]
total="16ctbivFSGPosEWKRwxhSUt6gg9zhPu3jQ"
useBalance=false

[exec.sub.jvm]
jdkPath="../../../../build/j2sdk-image"
`

//GetDefaultCfgstring ...
func GetDefaultCfgstring() string {
	return cfgstring
}
