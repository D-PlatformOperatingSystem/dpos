// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"sync"

	"github.com/D-PlatformOperatingSystem/dpos/client"
	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

//Query
type Query struct {
	db        dbm.DB
	stateHash []byte
	client    queue.Client
	mu        sync.Mutex
	api       client.QueueProtocolAPI
}

//NewQuery new
func NewQuery(db dbm.DB, qclient queue.Client, stateHash []byte) *Query {
	var err error
	query := &Query{db: db, client: qclient, stateHash: stateHash}
	query.api, err = client.New(qclient, nil)
	if err != nil {
		panic("NewQuery client.New err")
	}
	return query
}

//Query
func (q *Query) Query(driver string, funcname string, param types.Message) (types.Message, error) {
	query := &types.ChainExecutor{
		Driver:    driver,
		FuncName:  funcname,
		Param:     types.Encode(param),
		StateHash: q.getStateHash(),
	}
	return q.api.QueryChain(query)
}

func (q *Query) updateStateHash(stateHash []byte) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.stateHash = stateHash
}

func (q *Query) getStateHash() (stateHash []byte) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.stateHash
}
