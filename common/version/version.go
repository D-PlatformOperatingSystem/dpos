// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package version
package version

const version = "1.64.0"

//var
var (
	WalletVerKey     = []byte("WalletVerKey")
	BlockChainVerKey = []byte("BlockChainVerKey")
	LocalDBMeta      = []byte("LocalDBMeta")
	StoreDBMeta      = []byte("StoreDBMeta")
	MavlTreeVerKey   = []byte("MavlTreeVerKey")
	localversion     = "2.0.0"
	storeversion     = "1.0.0"
	appversion       = "1.0.0"
	GitCommit        string
)

//GetLocalDBKeyList     key
func GetLocalDBKeyList() [][]byte {
	return [][]byte{
		WalletVerKey, BlockChainVerKey, LocalDBMeta, StoreDBMeta, MavlTreeVerKey,
	}
}

//GetVersion
func GetVersion() string {
	if GitCommit != "" {
		return version + "-" + GitCommit
	}
	return version
}

//GetLocalDBVersion
/*
  : v1.v2.v3
  : v1    ，      localdb       reindex
*/
func GetLocalDBVersion() string {
	return localversion
}

//SetLocalDBVersion        ，
func SetLocalDBVersion(version string) {
	if version != "" {
		localversion = version
	}
}

//GetStoreDBVersion
/*
  : v1.v2.v3
  : v1    ，      storedb
*/
func GetStoreDBVersion() string {
	return storeversion
}

//SetStoreDBVersion        ，
func SetStoreDBVersion(version string) {
	if version != "" {
		storeversion = version
	}
}

//GetAppVersion      app
func GetAppVersion() string {
	return appversion
}

//SetAppVersion
func SetAppVersion(version string) {
	if version != "" {
		appversion = version
	}
}

//v0.1.2
//    ：
// 1.p2p     nat   ，   peer stream，ping,version

//v0.1.3
//

//v0.1.4
//
// p2p              ，
// p2p    p2p serverStart   ，                 ，         ，      ，

//
// blcokchain db
//	ver=1:            ，
// wallet db:
//	ver=1:  rescan   ，   wallet     tx         blockchain
// state mavltree db

//v5.3.0
//hard fork for bug

//v6.2.0
//    +mempool

//v6.4.0
//      ，
