// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

func init() {
	RegisterPlugin("stat", &statPlugin{})
}

type statPlugin struct {
	pluginBase
}

func (p *statPlugin) CheckEnable(executor *executor, enable bool) (kvs []*types.KeyValue, ok bool, err error) {
	kvs, ok, err = p.checkFlag(executor, types.StatisticFlag(), enable)
	if err == types.ErrDBFlag {
		panic("stat config is enable, it must be synchronized from 0 height ")
	}
	return kvs, ok, err
}

func (p *statPlugin) ExecLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error) {
	return countInfo(executor, data)
}

func (p *statPlugin) ExecDelLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error) {
	return delCountInfo(executor, data)
}

func countInfo(ex *executor, b *types.BlockDetail) ([]*types.KeyValue, error) {
	var kvset types.LocalDBSet
	//
	ticketkv, err := countTicket(ex, b)
	if err != nil {
		return nil, err
	}
	if ticketkv == nil {
		return nil, nil
	}
	kvset.KV = ticketkv.KV
	return kvset.KV, nil
}

func delCountInfo(ex *executor, b *types.BlockDetail) ([]*types.KeyValue, error) {
	var kvset types.LocalDBSet
	//
	ticketkv, err := delCountTicket(ex, b)
	if err != nil {
		return nil, err
	}
	if ticketkv == nil {
		return nil, nil
	}
	kvset.KV = ticketkv.KV
	return kvset.KV, nil
}

//           ticket    。
//          ，      。       0
func countTicket(ex *executor, b *types.BlockDetail) (*types.LocalDBSet, error) {
	return nil, nil
}

func delCountTicket(ex *executor, b *types.BlockDetail) (*types.LocalDBSet, error) {
	return nil, nil
}
