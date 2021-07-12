// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package commands    dapp
package commands

import (
	"github.com/D-PlatformOperatingSystem/dpos/rpc/jsonclient"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/spf13/cobra"
)

// VersionCmd version command
func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Get node version",
		Run:   version,
	}

	return cmd
}

func version(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res types.VersionInfo
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "DplatformOS.Version", nil, &res)
	ctx.Run()

}
