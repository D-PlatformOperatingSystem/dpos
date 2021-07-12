// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"runtime"
	//log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
)

var (
	poolCacheSize          int64 = 10240 // mempool
	mempoolExpiredInterval int64 = 600   // mempool       ，10
	maxTxNumPerAccount     int64 = 100   // TODO      mempool       ，10
	maxTxLast              int64 = 10
	processNum             int
)

// TODO
func init() {
	processNum = runtime.NumCPU()
}
