// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build go1.8

// package main          MAVL
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	clog "github.com/D-PlatformOperatingSystem/dpos/common/log"
	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	mavl "github.com/D-PlatformOperatingSystem/dpos/system/store/mavl/db"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

var (
	queryHashCount = flag.Bool("qh", false, "  hash    ")
	queryLeafCount = flag.Bool("ql", false, "        ")
	queryAllCount  = flag.Bool("qa", false, "        ")

	queryLeafCountCount    = flag.Bool("qlc", false, "          ")
	queryOldLeafCountCount = flag.Bool("qolc", false, "            ")
	pruneTreeHeight        = flag.Int64("pth", 0, "     ")

	querySameLeafCountKeyPri = flag.String("qslc", "", "         key     ")
	queryHashNode            = flag.String("qhn", "", "  hash       list     ")

	//queryLeafNodeParent
	key    = flag.String("k", "", "            ：key")
	hash   = flag.String("h", "", "            ：hash")
	height = flag.Int64("hei", 0, "            ：height")
)

func main() {
	flag.Parse()
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(dir)
	str := "dbug"

	log1 := &types.Log{
		Loglevel:        str,
		LogConsoleLevel: "info",
		LogFile:         "logs/syc.log",
		MaxFileSize:     400,
		MaxBackups:      100,
		MaxAge:          28,
		LocalTime:       true,
		Compress:        false,
	}
	clog.SetFileLog(log1)

	log.Info("dir", "test", dir)
	db := dbm.NewDB("store", "leveldb", dir, 100)

	if *queryHashCount {
		mavl.PruningTreePrintDB(db, []byte("_mh_"))
	}

	if *queryLeafCount {
		mavl.PruningTreePrintDB(db, []byte("_mb_"))
	}

	if *queryAllCount {
		mavl.PruningTreePrintDB(db, nil)
	}

	if *queryLeafCountCount {
		mavl.PruningTreePrintDB(db, []byte("..mk.."))
	}

	if *queryOldLeafCountCount {
		mavl.PruningTreePrintDB(db, []byte("..mok.."))
	}

	if *pruneTreeHeight > 0 {
		treeCfg := &mavl.TreeConfig{
			PruneHeight: mavl.DefaultPruneHeight,
		}
		mavl.PruningTree(db, *pruneTreeHeight, treeCfg)
	}

	if len(*querySameLeafCountKeyPri) > 0 {
		mavl.PrintSameLeafKey(db, *querySameLeafCountKeyPri)
	}

	if len(*key) > 0 && len(*hash) > 0 && *height > 0 {
		hashb, err := common.FromHex(*hash)
		if err != nil {
			fmt.Println("common.FromHex err", *hash)
			return
		}
		mavl.PrintLeafNodeParent(db, []byte(*key), hashb, *height)
	}

	if len(*queryHashNode) > 0 {
		key, err := common.FromHex(*queryHashNode)
		if err == nil {
			mavl.PrintNodeDb(db, key)
		}
	}
}
