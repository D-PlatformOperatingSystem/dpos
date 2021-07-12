// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package consensus
package consensus

import (
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

//Create
type Create func(cfg *types.Consensus, sub []byte) queue.Module

var regConsensus = make(map[string]Create)

//QueryData
var QueryData = types.NewQueryData("Query_")

//Reg ...
func Reg(name string, create Create) {
	if create == nil {
		panic("Consensus: Register driver is nil")
	}
	if _, dup := regConsensus[name]; dup {
		panic("Consensus: Register called twice for driver " + name)
	}
	regConsensus[name] = create
}

//Load
func Load(name string) (create Create, err error) {
	if driver, ok := regConsensus[name]; ok {
		return driver, nil
	}
	return nil, types.ErrNotFound
}
