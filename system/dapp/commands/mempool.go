// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"github.com/D-PlatformOperatingSystem/dpos/rpc/jsonclient"
	rpctypes "github.com/D-PlatformOperatingSystem/dpos/rpc/types"
	"github.com/D-PlatformOperatingSystem/dpos/system/dapp/commands/types"
	ctypes "github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/spf13/cobra"
)

// MempoolCmd mempool command
func MempoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mempool",
		Short: "Mempool management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		GetMempoolCmd(),
		GetLastMempoolCmd(),
		GetProperFeeCmd(),
	)

	return cmd
}

// GetMempoolCmd get mempool
func GetMempoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List mempool txs",
		Run:   listMempoolTxs,
	}
	addGetMempoolFlags(cmd)
	return cmd
}

func addGetMempoolFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("all", "a", false, "show all tx in mempool")
}

func listMempoolTxs(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	isAll, _ := cmd.Flags().GetBool("all")
	params := &ctypes.ReqGetMempool{
		IsAll: isAll,
	}
	var res rpctypes.ReplyTxList
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "DplatformOS.GetMempool", params, &res)
	ctx.SetResultCb(parseListMempoolTxsRes)
	ctx.Run()
}

func parseListMempoolTxsRes(arg interface{}) (interface{}, error) {
	res := arg.(*rpctypes.ReplyTxList)
	var result types.TxListResult
	for _, v := range res.Txs {
		result.Txs = append(result.Txs, types.DecodeTransaction(v))
	}
	return result, nil
}

// GetLastMempoolCmd  get last 10 txs of mempool
func GetLastMempoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "last_txs",
		Short: "Get latest mempool txs",
		Run:   lastMempoolTxs,
	}
	return cmd
}

func lastMempoolTxs(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res rpctypes.ReplyTxList
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "DplatformOS.GetLastMemPool", nil, &res)
	ctx.SetResultCb(parselastMempoolTxsRes)
	ctx.Run()
}

func parselastMempoolTxsRes(arg interface{}) (interface{}, error) {
	res := arg.(*rpctypes.ReplyTxList)
	var result types.TxListResult
	for _, v := range res.Txs {
		result.Txs = append(result.Txs, types.DecodeTransaction(v))
	}
	return result, nil
}

// GetProperFeeCmd  get last proper fee
func GetProperFeeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proper_fee",
		Short: "Get latest proper fee",
		Run:   properFee,
	}
	return cmd
}

func properFee(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res rpctypes.ReplyProperFee
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "DplatformOS.GetProperFee", nil, &res)
	ctx.SetResultCb(nil)
	ctx.Run()
}
