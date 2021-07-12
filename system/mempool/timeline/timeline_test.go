// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timeline

import (
	"encoding/json"
	"testing"

	"github.com/D-PlatformOperatingSystem/dpos/system/mempool"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

func TestNewMempool(t *testing.T) {
	sub, _ := json.Marshal(&mempool.SubConfig{PoolCacheSize: 2})
	module := New(&types.Mempool{}, sub)
	mem := module.(*mempool.Mempool)
	mem.Close()
}
