// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package accounts

import (
	"time"

	l "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"

	"github.com/D-PlatformOperatingSystem/dpos/types"

	//"encoding/json"
	//"io/ioutil"
	"fmt"
	"strconv"

	rpctypes "github.com/D-PlatformOperatingSystem/dpos/rpc/types"
)

const secondsPerBlock = 5
const dposPreBlock = 5
const baseInterval = 3600
const maxInterval = 15 * 24 * 3600
const monitorDposLowLimit = 3 * 1e7 * types.Coin

var log = l.New("module", "accounts")

//ShowMinerAccount
type ShowMinerAccount struct {
	DataDir string
	Addrs   []string
}

//Echo
func (*ShowMinerAccount) Echo(in *string, out *interface{}) error {
	if in == nil {
		return types.ErrInvalidParam
	}
	*out = *in
	return nil
}

//TimeAt time
type TimeAt struct {
	// YYYY-mm-dd-HH
	TimeAt string   `json:"timeAt,omitempty"`
	Addrs  []string `json:"addrs,omitempty"`
}

//Get get
func (show *ShowMinerAccount) Get(in *TimeAt, out *interface{}) error {
	if in == nil {
		log.Error("show", "in", "nil")
		return types.ErrInvalidParam
	}
	addrs := show.Addrs
	if in.Addrs != nil && len(in.Addrs) > 0 {
		addrs = in.Addrs
	}
	log.Info("show", "miners", addrs)

	height, err := toBlockHeight(in.TimeAt)
	if err != nil {
		return err
	}
	log.Info("show", "header", height)

	header, curAcc, err := cache.getBalance(addrs, "ticket", height)
	if err != nil {
		log.Error("show", "getBalance failed", err, "height", height)
		return nil
	}

	totalDpos := int64(0)
	for _, acc := range curAcc {
		totalDpos += acc.Frozen
	}
	log.Info("show 1st balance", "utc", header.BlockTime, "total", totalDpos)

	monitorInterval := calcMoniterInterval(totalDpos)
	log.Info("show", "monitor Interval", monitorInterval)

	lastHourHeader, lastAcc, err := cache.getBalance(addrs, "ticket", header.Height-monitorInterval)
	if err != nil {
		log.Error("show", "getBalance failed", err, "ts", header.BlockTime-monitorInterval)
		return nil
	}
	fmt.Print(curAcc, lastAcc)
	log.Info("show 2nd balance", "utc", *lastHourHeader)

	miner := &MinerAccounts{}
	miner.Seconds = header.BlockTime - lastHourHeader.BlockTime
	miner.Blocks = header.Height - lastHourHeader.Height
	miner.ExpectBlocks = miner.Seconds / secondsPerBlock

	miner = calcIncrease(miner, curAcc, lastAcc, header)
	*out = &miner

	return nil
}

//            ，
func toBlockHeight(timeAt string) (int64, error) {
	seconds := time.Now().Unix()
	if len(timeAt) != 0 {
		tm, err := time.Parse("2006-01-02-15", timeAt)
		if err != nil {
			log.Error("show", "in.TimeAt Parse", err)
			return 0, types.ErrInvalidParam
		}
		seconds = tm.Unix()
	}
	log.Info("show", "utc-init", seconds)

	realTs, header := cache.findBlock(seconds)
	if realTs == 0 || header == nil {
		log.Error("show", "findBlock", "nil")
		return 0, types.ErrNotFound
	}
	return header.Height, nil
}

//
//         ，        ，  9000    138
func calcMoniterInterval(totalDpos int64) int64 {
	monitorInterval := int64(baseInterval)
	if totalDpos < monitorDposLowLimit && totalDpos > 0 {
		monitorInterval = int64(float64(monitorDposLowLimit) / float64(totalDpos) * float64(baseInterval))
	}
	if monitorInterval > maxInterval {
		monitorInterval = maxInterval
	}
	log.Info("show", "monitor Interval", monitorInterval)
	return monitorInterval / secondsPerBlock
}

func calcIncrease(miner *MinerAccounts, acc1, acc2 []*rpctypes.Account, header *rpctypes.Header) *MinerAccounts {
	type minerAt struct {
		addr    string
		curAcc  *rpctypes.Account
		lastAcc *rpctypes.Account
	}
	miners := map[string]*minerAt{}
	for _, a := range acc1 {
		miners[a.Addr] = &minerAt{a.Addr, a, nil}
	}
	for _, a := range acc2 {
		if _, ok := miners[a.Addr]; ok {
			miners[a.Addr].lastAcc = a
		}
	}

	totalIncrease := float64(0)
	expectTotalIncrease := float64(0)
	totalFrozen := float64(0)
	for _, v := range miners {
		if v.lastAcc != nil && v.curAcc != nil {
			totalFrozen += float64(v.curAcc.Frozen) / float64(types.Coin)
		}
	}
	ticketTotal := float64(30000 * 10000)
	_, ticketAcc, err := cache.getBalance([]string{"16ctbivFSGPosEWKRwxhSUt6gg9zhPu3jQ"}, "coins", header.Height)
	if err == nil && len(ticketAcc) == 1 {
		ticketTotal = float64(ticketAcc[0].Balance+ticketAcc[0].Frozen) / float64(types.Coin)
	}
	for k, v := range miners {
		if v.lastAcc != nil && v.curAcc != nil {
			total := v.curAcc.Balance + v.curAcc.Frozen
			increase := total - v.lastAcc.Balance - v.lastAcc.Frozen
			expectIncrease := float64(miner.Blocks) * float64(dposPreBlock) * (float64(v.curAcc.Frozen) / float64(types.Coin)) / ticketTotal

			m := &MinerAccount{
				Addr:           k,
				Total:          strconv.FormatFloat(float64(total)/float64(types.Coin), 'f', 4, 64),
				Increase:       strconv.FormatFloat(float64(increase)/float64(types.Coin), 'f', 4, 64),
				Frozen:         strconv.FormatFloat(float64(v.curAcc.Frozen)/float64(types.Coin), 'f', 4, 64),
				ExpectIncrease: strconv.FormatFloat(expectIncrease, 'f', 4, 64),
			}

			//if m.Addr == "1Lw6QLShKVbKM6QvMaCQwTh5Uhmy4644CG" {
			//	log.Info("acount", "Increase", m.Increase, "expectIncrease", m.ExpectIncrease)
			//	fmt.Println(os.Stderr, "Increase", m.Increase, "expectIncrease", m.ExpectIncrease)
			//}

			//           ，        ，             。
			//          ，
			expectBlocks := (expectIncrease / dposPreBlock)                              //
			expectMinerInterval := float64(miner.Seconds/secondsPerBlock) / expectBlocks //
			moniterInterval := int64(2*expectMinerInterval) + 1

			m.ExpectMinerBlocks = strconv.FormatFloat(expectBlocks, 'f', 4, 64)
			_, acc, err := cache.getBalance([]string{m.Addr}, "ticket", header.Height-moniterInterval)
			if err != nil || len(acc) == 0 {
				m.MinerDposDuring = "0.0000"
			} else {
				minerDelta := total - acc[0].Balance - acc[0].Frozen
				m.MinerDposDuring = strconv.FormatFloat(float64(minerDelta)/float64(types.Coin), 'f', 4, 64)
			}

			miner.MinerAccounts = append(miner.MinerAccounts, m)
			totalIncrease += float64(increase) / float64(types.Coin)
			expectTotalIncrease += expectIncrease
		}
	}
	miner.TotalIncrease = strconv.FormatFloat(totalIncrease, 'f', 4, 64)
	miner.ExpectTotalIncrease = strconv.FormatFloat(expectTotalIncrease, 'f', 4, 64)

	return miner

}
