// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import "errors"

//TTL
const (
	//              ttl，    1
	defaultLtTxBroadCastTTL = 3
	//            ttl，
	defaultMaxTxBroadCastTTL = 25
	//             ， 100KB
	defaultMinLtBlockSize = 100
	//              ， 50M
	defaultLtBlockCacheSize = 50
)

// P2pCacheTxSize p2pcache size of transaction
const (
	//             mempool
	txRecvFilterCacheNum    = 10240
	blockRecvFilterCacheNum = 1024
	//               ,          ,
	txSendFilterCacheNum    = 500
	blockSendFilterCacheNum = 50
	ltBlockCacheNum         = 1000
)

//
var (
	errQueryMempool     = errors.New("errQueryMempool")
	errQueryBlockChain  = errors.New("errQueryBlockChain")
	errRecvBlockChain   = errors.New("errRecvBlockChain")
	errRecvMempool      = errors.New("errRecvMempool")
	errSendPeer         = errors.New("errSendPeer")
	errSendBlockChain   = errors.New("errSendBlockChain")
	errBuildBlockFailed = errors.New("errBuildBlockFailed")
	errLtBlockNotExist  = errors.New("errLtBlockNotExist")
)

//

const (
	bcTopic      = "broadcast"
	broadcastTag = "broadcast-tag"
)

// pubsub error
var (
	errSnappyDecode = errors.New("errSnappyDecode")
)
