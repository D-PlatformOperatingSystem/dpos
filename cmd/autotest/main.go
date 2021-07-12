// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package            ，              ，
//                    。
//             ，         ，           。
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/D-PlatformOperatingSystem/dpos/cmd/autotest/testflow"
	//       dapp AutoTest
	_ "github.com/D-PlatformOperatingSystem/dpos/system"
)

var (
	configFile string
	logFile    string
)

func init() {

	flag.StringVar(&configFile, "f", "autotest.toml", "-f configFile")
	flag.StringVar(&logFile, "l", "autotest.log", "-l logFile")
	flag.Parse()
}

func main() {

	testflow.InitFlowConfig(configFile, logFile)

	if testflow.StartAutoTest() {

		fmt.Println("========================================Succeed!============================================")
		os.Exit(0)
	} else {
		fmt.Println("==========================================Failed!============================================")
		os.Exit(1)
	}
}
