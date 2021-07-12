// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	//	"encoding/json"

	"strconv"
	"strings"

	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	"github.com/D-PlatformOperatingSystem/dpos/rpc/jsonclient"
	rpctypes "github.com/D-PlatformOperatingSystem/dpos/rpc/types"
	cty "github.com/D-PlatformOperatingSystem/dpos/system/dapp/coins/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/spf13/cobra"

	// TODO:              ，

	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"
)

// DecodeTransaction decode transaction function
func DecodeTransaction(tx *rpctypes.Transaction) *TxResult {
	result := &TxResult{
		Execer:     tx.Execer,
		Payload:    tx.Payload,
		RawPayload: tx.RawPayload,
		Signature:  tx.Signature,
		Fee:        tx.FeeFmt,
		Amount:     tx.AmountFmt,
		Expire:     tx.Expire,
		Nonce:      tx.Nonce,
		To:         tx.To,
		From:       tx.From,
		GroupCount: tx.GroupCount,
		Header:     tx.Header,
		Next:       tx.Next,
		Hash:       tx.Hash,
	}
	return result
}

// DecodeAccount decode account func
func DecodeAccount(acc *types.Account, precision int64) *AccountResult {
	balanceResult := strconv.FormatFloat(float64(acc.GetBalance())/float64(precision), 'f', 4, 64)
	frozenResult := strconv.FormatFloat(float64(acc.GetFrozen())/float64(precision), 'f', 4, 64)
	accResult := &AccountResult{
		Addr:     acc.GetAddr(),
		Currency: acc.GetCurrency(),
		Balance:  balanceResult,
		Frozen:   frozenResult,
	}
	return accResult
}

// SendToAddress send to address func
func SendToAddress(rpcAddr string, from string, to string, amount int64, note string, isToken bool, tokenSymbol string, isWithdraw bool) {
	amt := amount
	if isWithdraw {
		amt = -amount
	}
	params := types.ReqWalletSendToAddress{From: from, To: to, Amount: amt, Note: note}
	if !isToken {
		params.IsToken = false
	} else {
		params.IsToken = true
		params.TokenSymbol = tokenSymbol
	}

	var res rpctypes.ReplyHash
	ctx := jsonclient.NewRPCCtx(rpcAddr, "DplatformOS.SendToAddress", params, &res)
	ctx.Run()
}

// CreateRawTx create rawtransaction func
func CreateRawTx(cmd *cobra.Command, to string, amount float64, note string, isWithdraw bool, tokenSymbol, execName string) (string, error) {
	title, _ := cmd.Flags().GetString("title")
	cfg := types.GetCliSysParam(title)

	if amount < 0 {
		return "", types.ErrAmount
	}
	if float64(types.MaxCoin/types.Coin) < amount {
		return "", types.ErrAmount
	}
	//  to
	if to != "" {
		if err := address.CheckAddress(to); err != nil {
			return "", types.ErrInvalidAddress
		}
	}
	paraName, _ := cmd.Flags().GetString("paraName")
	amountInt64 := int64(math.Trunc((amount+0.0000001)*1e4)) * 1e4
	if execName != "" && !types.IsAllowExecName([]byte(execName), []byte(execName)) {
		return "", types.ErrExecNameNotMatch
	}
	var tx *types.Transaction
	transfer := &cty.CoinsAction{}
	if !isWithdraw {
		if execName != "" {
			v := &cty.CoinsAction_TransferToExec{TransferToExec: &types.AssetsTransferToExec{Amount: amountInt64, Note: []byte(note), ExecName: execName, To: to}}
			transfer.Value = v
			transfer.Ty = cty.CoinsActionTransferToExec
		} else {
			v := &cty.CoinsAction_Transfer{Transfer: &types.AssetsTransfer{Amount: amountInt64, Note: []byte(note), To: to}}
			transfer.Value = v
			transfer.Ty = cty.CoinsActionTransfer
		}
	} else {
		v := &cty.CoinsAction_Withdraw{Withdraw: &types.AssetsWithdraw{Amount: amountInt64, Note: []byte(note), ExecName: execName, To: to}}
		transfer.Value = v
		transfer.Ty = cty.CoinsActionWithdraw
	}
	execer := []byte(getRealExecName(paraName, "coins"))
	if paraName == "" {
		tx = &types.Transaction{Execer: execer, Payload: types.Encode(transfer), To: to}
	} else {
		tx = &types.Transaction{Execer: execer, Payload: types.Encode(transfer), To: address.ExecAddress(string(execer))}
	}
	tx, err := types.FormatTx(cfg, string(execer), tx)
	if err != nil {
		return "", err
	}
	txHex := types.Encode(tx)
	return hex.EncodeToString(txHex), nil
}

// GetExecAddr get exec address func
func GetExecAddr(exec string) (string, error) {
	if ok := types.IsAllowExecName([]byte(exec), []byte(exec)); !ok {
		return "", types.ErrExecNameNotAllow
	}

	addrResult := address.ExecAddress(exec)
	result := addrResult
	return result, nil
}

// FormatAmountValue2Display    、   amount
func FormatAmountValue2Display(amount int64) string {
	return strconv.FormatFloat(float64(amount)/float64(types.Coin), 'f', 4, 64)
}

// FormatAmountDisplay2Value    、   amount       、
func FormatAmountDisplay2Value(amount float64) int64 {
	return int64(amount*types.InputPrecision) * types.Multiple1E4
}

// GetAmountValue       amount    int64
func GetAmountValue(cmd *cobra.Command, field string) int64 {
	amount, _ := cmd.Flags().GetFloat64(field)
	return FormatAmountDisplay2Value(amount)
}

func getRealExecName(paraName string, name string) string {
	if strings.HasPrefix(name, "user.p.") {
		return name
	}
	return paraName + name
}

func parseTxHeight(expire string) error {
	if len(expire) == 0 {
		return errors.New("expire string should not be empty")
	}

	if expire[0] == 'H' && expire[1] == ':' {
		txHeight, err := strconv.Atoi(expire[2:])
		if err != nil {
			return err
		}
		if txHeight <= 0 {
			//fmt.Printf("txHeight should be grate to 0")
			return errors.New("txHeight should be grate to 0")
		}

		return nil
	}

	return errors.New("Invalid expire format. Should be one of {time:\"3600s/1min/1h\" block:\"123\" txHeight:\"H:123\"}")
}

// CheckExpireOpt parse expire option in command
func CheckExpireOpt(expire string) (string, error) {
	//    123s/1m/1h
	expireTime, err := time.ParseDuration(expire)
	if err == nil {
		if expireTime < time.Minute*2 && expireTime != time.Second*0 {
			expire = "120s"
			fmt.Println("expire time must longer than 2 minutes, changed expire time into 2 minutes")
		}

		return expire, nil
	}

	//      ，123
	blockInt, err := strconv.Atoi(expire)
	if err == nil {
		if blockInt <= 0 {
			fmt.Printf("block height should be grate to 0")
			return "", errors.New("block height should be grate to 0")
		}
		return expire, nil
	}

	//Txheight  ，H:123
	err = parseTxHeight(expire)
	if err != nil {
		return "", err
	}

	return expire, err
}