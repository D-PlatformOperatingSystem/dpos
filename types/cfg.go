// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

// Config
type Config struct {
	Title          string       `json:"title,omitempty"`
	Version        string       `json:"version,omitempty"`
	Log            *Log         `json:"log,omitempty"`
	Store          *Store       `json:"store,omitempty"`
	Consensus      *Consensus   `json:"consensus,omitempty"`
	Mempool        *Mempool     `json:"memPool,omitempty"`
	BlockChain     *BlockChain  `json:"blockChain,omitempty"`
	Wallet         *Wallet      `json:"wallet,omitempty"`
	P2P            *P2P         `json:"p2p,omitempty"`
	RPC            *RPC         `json:"rpc,omitempty"`
	Exec           *Exec        `json:"exec,omitempty"`
	TestNet        bool         `json:"testNet,omitempty"`
	FixTime        bool         `json:"fixTime,omitempty"`
	TxHeight       bool         `json:"txHeight,omitempty"`
	Pprof          *Pprof       `json:"pprof,omitempty"`
	Fork           *ForkList    `json:"fork,omitempty"`
	Health         *HealthCheck `json:"health,omitempty"`
	CoinSymbol     string       `json:"coinSymbol,omitempty"`
	EnableParaFork bool         `json:"enableParaFork,omitempty"`
	Metrics        *Metrics     `json:"metrics,omitempty"`
}

// ForkList fork
type ForkList struct {
	System map[string]int64
	Sub    map[string]map[string]int64
}

// Log
type Log struct {
	//     ，  debug(dbug)/info/warn/error(eror)/crit
	Loglevel        string `json:"loglevel,omitempty"`
	LogConsoleLevel string `json:"logConsoleLevel,omitempty"`
	//      ，    ，
	LogFile string `json:"logFile,omitempty"`
	//           （  ： ）
	MaxFileSize uint32 `json:"maxFileSize,omitempty"`
	//
	MaxBackups uint32 `json:"maxBackups,omitempty"`
	//            （  ： ）
	MaxAge uint32 `json:"maxAge,omitempty"`
	//              （    UTC  ）
	LocalTime bool `json:"localTime,omitempty"`
	//           （     gz）
	Compress bool `json:"compress,omitempty"`
	//
	CallerFile bool `json:"callerFile,omitempty"`
	//
	CallerFunction bool `json:"callerFunction,omitempty"`
}

// Mempool
type Mempool struct {
	// mempool    ，  ，timeline，score，price
	Name string `json:"name,omitempty"`
	// mempool      ，  10240
	PoolCacheSize int64 `json:"poolCacheSize,omitempty"`
	ForceAccept   bool  `json:"forceAccept,omitempty"`
	//      mempool        ，  100
	MaxTxNumPerAccount int64 `json:"maxTxNumPerAccount,omitempty"`
	MaxTxLast          int64 `json:"maxTxLast,omitempty"`
	IsLevelFee         bool  `json:"isLevelFee,omitempty"`
	//        ，       ，  ，   100000
	MinTxFeeRate int64 `json:"minTxFeeRate,omitempty"`
	//        ,   1e7
	MaxTxFeeRate int64 `json:"maxTxFeeRate,omitempty"`
	//        ,   1e9
	MaxTxFee int64 `json:"maxTxFee,omitempty"`
	//   execCheck    ，      execCheck，
	DisableExecCheck bool `json:"disableExecCheck,omitempty"`
}

// Consensus
type Consensus struct {
	//      ：solo, ticket, raft, tendermint, para
	Name string `json:"name,omitempty"`
	//       (UTC  )
	GenesisBlockTime int64 `json:"genesisBlockTime,omitempty"`
	//       ,
	Minerstart bool `json:"minerstart,omitempty"`
	//
	Genesis     string `json:"genesis,omitempty"`
	HotkeyAddr  string `json:"hotkeyAddr,omitempty"`
	ForceMining bool   `json:"forceMining,omitempty"`
	//
	MinerExecs []string `json:"minerExecs,omitempty"`
	//
	EnableBestBlockCmp bool `json:"enableBestBlockCmp,omitempty"`
}

// Wallet
type Wallet struct {
	//          ，  0.00000001DOM(1e-8),  100000， 0.001DOM
	MinFee int64 `json:"minFee,omitempty"`
	// walletdb
	Driver string `json:"driver,omitempty"`
	// walletdb
	DbPath string `json:"dbPath,omitempty"`
	// walletdb
	DbCache int32 `json:"dbCache,omitempty"`
	//
	SignType string `json:"signType,omitempty"`
	CoinType string `json:"coinType,omitempty"`
}

// Store
type Store struct {
	//         ，    mavl,kvdb,kvmvcc,mpt
	Name string `json:"name,omitempty"`
	//         ，    leveldb,goleveldb,memdb,gobadgerdb,ssdb,pegasus
	Driver string `json:"driver,omitempty"`
	//
	DbPath string `json:"dbPath,omitempty"`
	// Cache
	DbCache int32 `json:"dbCache,omitempty"`
	//
	LocalDBVersion string `json:"localdbVersion,omitempty"`
	//
	StoreDBVersion string `json:"storedbVersion,omitempty"`
}

// BlockChain
type BlockChain struct {
	//
	ChunkblockNum int64 `json:"chunkblockNum,omitempty"`
	//
	DefCacheSize int64 `json:"defCacheSize,omitempty"`
	//
	MaxFetchBlockNum int64 `json:"maxFetchBlockNum,omitempty"`
	//
	TimeoutSeconds int64 `json:"timeoutSeconds,omitempty"`
	BatchBlockNum  int64 `json:"batchBlockNum,omitempty"`
	//
	Driver string `json:"driver,omitempty"`
	//
	DbPath string `json:"dbPath,omitempty"`
	//
	DbCache             int32 `json:"dbCache,omitempty"`
	IsStrongConsistency bool  `json:"isStrongConsistency,omitempty"`
	//
	SingleMode bool `json:"singleMode,omitempty"`
	//            ，         ，             false，
	Batchsync bool `json:"batchsync,omitempty"`
	//                ，         ，          ，     true
	IsRecordBlockSequence bool `json:"isRecordBlockSequence,omitempty"`
	//
	IsParaChain        bool `json:"isParaChain,omitempty"`
	EnableTxQuickIndex bool `json:"enableTxQuickIndex,omitempty"`
	//   storedb      localdb
	EnableReExecLocal bool `json:"enableReExecLocal,omitempty"`
	//
	RollbackBlock int64 `json:"rollbackBlock,omitempty"`
	//
	RollbackSave bool `json:"rollbackSave,omitempty"`
	//           ，   。
	OnChainTimeout int64 `json:"onChainTimeout,omitempty"`
	//     localdb
	EnableReduceLocaldb bool `json:"enableReduceLocaldb,omitempty"`
	//       ,         false;                  true
	DisableShard bool `protobuf:"varint,19,opt,name=disableShard" json:"disableShard,omitempty"`
	//    P2pStore
	EnableFetchP2pstore bool `json:"enableFetchP2pstore,omitempty"`
	//              ,
	EnableIfDelLocalChunk bool `json:"enableIfDelLocalChunk,omitempty"`
	//         、
	EnablePushSubscribe bool `json:"EnablePushSubscribe,omitempty"`
	//
	MaxActiveBlockNum int `json:"maxActiveBlockNum,omitempty"`
	//            M
	MaxActiveBlockSize int `json:"maxActiveBlockSize,omitempty"`

	//HighAllowPackHeight      High
	HighAllowPackHeight int64 `json:"highAllowPackHeight,omitempty"`

	//LowAllowPackHeight      low
	LowAllowPackHeight int64 `json:"lowAllowPackHeight,omitempty"`
}

// P2P
type P2P struct {
	//
	Driver string `json:"driver,omitempty"`
	//
	DbPath string `json:"dbPath,omitempty"`
	//
	DbCache int32 `json:"dbCache,omitempty"`
	// GRPC
	GrpcLogFile string `json:"grpcLogFile,omitempty"`
	//     P2P
	Enable bool `json:"enable,omitempty"`
	//    Pid
	WaitPid bool `json:"waitPid,omitempty"`
	//  p2p  ,   gossip, dht
	Types []string `json:"types,omitempty"`
}

// RPC
type RPC struct {
	// jrpc
	JrpcBindAddr string `json:"jrpcBindAddr,omitempty"`
	// grpc
	GrpcBindAddr string `json:"grpcBindAddr,omitempty"`
	//      ，     IP  ，   “*”，    IP
	Whitlist  []string `json:"whitlist,omitempty"`
	Whitelist []string `json:"whitelist,omitempty"`
	// jrpc       ，   “*”，      RPC
	JrpcFuncWhitelist []string `json:"jrpcFuncWhitelist,omitempty"`
	// grpc       ，   “*”，      RPC
	GrpcFuncWhitelist []string `json:"grpcFuncWhitelist,omitempty"`
	// jrpc       ，           rpc  ，          ，
	JrpcFuncBlacklist []string `json:"jrpcFuncBlacklist,omitempty"`
	// grpc       ，           rpc  ，          ，
	GrpcFuncBlacklist []string `json:"grpcFuncBlacklist,omitempty"`
	//     https
	EnableTLS   bool `json:"enableTLS,omitempty"`
	EnableTrace bool `json:"enableTrace,omitempty"`
	//     ，          cli
	CertFile string `json:"certFile,omitempty"`
	//
	KeyFile string `json:"keyFile,omitempty"`
}

// Exec
type Exec struct {
	//     stat
	EnableStat bool `json:"enableStat,omitempty"`
	//     MVCC
	EnableMVCC       bool     `json:"enableMVCC,omitempty"`
	DisableAddrIndex bool     `json:"disableAddrIndex,omitempty"`
	Alias            []string `json:"alias,omitempty"`
	//     token
	SaveTokenTxList bool `json:"saveTokenTxList,omitempty"`
}

// Pprof
type Pprof struct {
	ListenAddr string `json:"listenAddr,omitempty"`
}

// HealthCheck
type HealthCheck struct {
	ListenAddr     string `json:"listenAddr,omitempty"`
	CheckInterval  uint32 `json:"checkInterval,omitempty"`
	UnSyncMaxTimes uint32 `json:"unSyncMaxTimes,omitempty"`
}

// Metrics
type Metrics struct {
	EnableMetrics bool   `json:"enableMetrics,omitempty"`
	DataEmitMode  string `json:"dataEmitMode,omitempty"`
	Duration      int64  `json:"duration,omitempty"`
	URL           string `json:"url,omitempty"`
	DatabaseName  string `json:"databaseName,omitempty"`
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
}
