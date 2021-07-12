// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/D-PlatformOperatingSystem/dpos/common/address"
)

//OneStepSendCmd send cmd
func OneStepSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "send",
		Short:              "send tx in one step",
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			oneStepSend(cmd, os.Args[0], args)
		},
	}
	cmd.Flags().StringP("key", "k", "", "private key or from address for sign tx")
	//cmd.MarkFlagRequired("key")
	return cmd
}

// one step send
func oneStepSend(cmd *cobra.Command, cmdName string, params []string) {
	if len(params) < 1 || params[0] == "help" || params[0] == "--help" || params[0] == "-h" {
		loadSendHelp()
		return
	}

	var createParams, keyParams []string
	//  send   key  ,            
	for i, v := range params {
		if strings.HasPrefix(v, "-k=") || strings.HasPrefix(v, "--key=") {
			keyParams = append(keyParams, v)
			createParams = append(params[:i], params[i+1:]...)
			break
		} else if (v == "-k" || v == "--key") && i < len(params)-1 {
			keyParams = append(keyParams, v, params[i+1])
			createParams = append(params[:i], params[i+2:]...)
			break
		}
	}
	//  send  parse    key  
	err := cmd.Flags().Parse(keyParams)
	key, _ := cmd.Flags().GetString("key")
	if len(key) < 32 || err != nil {
		loadSendHelp()
		fmt.Fprintln(os.Stderr, "Error: required flag(s) \"key\" not proper set")
		return
	}

	//       key  
	if address.CheckAddress(key) == nil {
		keyParams = append([]string{}, "-a", key)
	} else {
		keyParams = append([]string{}, "-k", key)
	}
	//      
	cmdCreate := exec.Command(cmdName, createParams...)
	createRes, err := execCmd(cmdCreate)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	//cli          ,       
	if strings.Contains(createRes, "\n") {
		fmt.Println(createRes)
		return
	}
	//           ,  rpc_laddr    
	createCmd, createFlags, _ := cmd.Root().Traverse(createParams)
	_ = createCmd.ParseFlags(createFlags)
	rpcAddr, _ := createCmd.Flags().GetString("rpc_laddr")
	//        
	signParams := []string{"wallet", "sign", "-d", createRes, "--rpc_laddr", rpcAddr}
	cmdSign := exec.Command(cmdName, append(signParams, keyParams...)...)
	signRes, err := execCmd(cmdSign)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	//        
	sendParams := []string{"wallet", "send", "-d", signRes, "--rpc_laddr", rpcAddr}
	cmdSend := exec.Command(cmdName, sendParams...)
	sendRes, err := execCmd(cmdSend)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(sendRes)
}

func execCmd(c *exec.Cmd) (string, error) {
	var outBuf, errBuf bytes.Buffer
	c.Stderr = &errBuf
	c.Stdout = &outBuf

	if err := c.Run(); err != nil {
		return "", errors.New(err.Error() + "\n" + errBuf.String())
	}
	if len(errBuf.String()) > 0 {
		return "", errors.New(errBuf.String())
	}
	outBytes := outBuf.Bytes()
	return string(outBytes[:len(outBytes)-1]), nil
}

func loadSendHelp() {
	help := `[Integrate create/sign/send transaction operations in one command]
Usage:
  -cli send [flags]

Examples:
cli send coins transfer -a 1 -n note -t toAddr -k [privateKey | fromAddr]

equivalent to three steps: 
1. cli coins transfer -a 1 -n note -t toAddr   //create raw tx
2. cli wallet sign -d rawTx -k privateKey      //sign raw tx
3. cli wallet send -d signTx                   //send tx to block chain

Flags:
  -h, --help         help for send
  -k, --key			 private key or from address for sign tx, required`
	fmt.Println(help)
}
