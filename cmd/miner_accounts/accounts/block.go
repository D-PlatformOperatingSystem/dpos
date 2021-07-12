// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package accounts
package accounts

import (
	"fmt"
	"os"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/rpc/jsonclient"
	rpctypes "github.com/D-PlatformOperatingSystem/dpos/rpc/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

// ts/height -> blockHeader
type dplatformos struct {
	lastHeader *rpctypes.Header
	// height -> ts -> header
	Headers   map[int64]*rpctypes.Header
	Height2Ts map[int64]int64
	// new only cache ticket for miner
	accountCache map[int64]*types.Accounts
	Host         string
}

func (b dplatformos) findBlock(ts int64) (int64, *rpctypes.Header) {
	log.Info("show", "utc", ts, "lastBlockTime", b.lastHeader.BlockTime)
	if ts > b.lastHeader.BlockTime {
		ts = b.lastHeader.BlockTime
		return ts, b.lastHeader
	}
	//                block
	//  1.      ，       ，        ts ，
	for delta := int64(1); delta < int64(100); delta++ {

		if _, ok := b.Headers[ts-delta]; ok {
			log.Info("show", "utc", ts, "find", b.Headers[ts-delta])
			return ts - delta, b.Headers[ts-delta]
		}
	}

	return 0, nil
}

func (b dplatformos) getBalance(addrs []string, exec string, height int64) (*rpctypes.Header, []*rpctypes.Account, error) {
	rpcCli, err := jsonclient.NewJSONClient(b.Host)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, nil, err
	}
	hs, err := getHeaders(rpcCli, height, height)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, nil, err
	}

	acc, err := getBalanceAt(rpcCli, addrs, exec, hs.Items[0].StateHash)
	return hs.Items[0], acc, err
}

func (b dplatformos) addBlock(h *rpctypes.Header) error {
	b.Headers[h.BlockTime] = h
	b.Height2Ts[h.Height] = h.BlockTime
	if h.Height > b.lastHeader.Height {
		cache.lastHeader = h
	}

	return nil
}

var cache = dplatformos{
	lastHeader:   &rpctypes.Header{Height: 0},
	Headers:      map[int64]*rpctypes.Header{},
	Height2Ts:    map[int64]int64{},
	accountCache: map[int64]*types.Accounts{},
}

func getLastHeader(cli *jsonclient.JSONClient) (*rpctypes.Header, error) {
	method := "DplatformOS.GetLastHeader"
	var res rpctypes.Header
	err := cli.Call(method, nil, &res)
	return &res, err
}

func getHeaders(cli *jsonclient.JSONClient, start, end int64) (*rpctypes.Headers, error) {
	method := "DplatformOS.GetHeaders"
	params := &types.ReqBlocks{Start: start, End: end, IsDetail: false}
	var res rpctypes.Headers
	err := cli.Call(method, params, &res)
	return &res, err
}

func getBalanceAt(cli *jsonclient.JSONClient, addrs []string, exec, stateHash string) ([]*rpctypes.Account, error) {
	method := "DplatformOS.GetBalance"
	params := &types.ReqBalance{Addresses: addrs, Execer: exec, StateHash: stateHash}
	var res []*rpctypes.Account
	err := cli.Call(method, params, &res)
	return res, err
}

func syncHeaders(host string) {
	rpcCli, err := jsonclient.NewJSONClient(host)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	last, err := getLastHeader(rpcCli)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	curHeight := cache.lastHeader.Height
	lastHeight := last.Height

	//        ， 15000      10M
	if lastHeight-15000 > curHeight { //} && false {
		curHeight = lastHeight - 15000
	}

	for curHeight < lastHeight {
		hs, err := getHeaders(rpcCli, curHeight, curHeight+100)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		for _, h := range hs.Items {
			if h.Height > cache.lastHeader.Height {
				curHeight = h.Height
			}
			cache.addBlock(h)
		}
		fmt.Fprintln(os.Stderr, err, cache.lastHeader.Height)
	}

	fmt.Fprintln(os.Stderr, err, cache.lastHeader.Height)
}

//SyncBlock
func SyncBlock(host string) {
	cache.Host = host
	syncHeaders(host)

	timeout := time.NewTicker(time.Minute)
	for {
		<-timeout.C
		syncHeaders(host)
	}
}
