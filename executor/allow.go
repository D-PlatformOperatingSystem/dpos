// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"bytes"

	"github.com/pkg/errors"

	drivers "github.com/D-PlatformOperatingSystem/dpos/system/dapp"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

func isAllowKeyWrite(e *executor, key, realExecer []byte, tx *types.Transaction, index int) bool {
	keyExecer, err := types.FindExecer(key)
	if err != nil {
		elog.Error("find execer ", "err", err, "key", string(key), "keyexecer", string(keyExecer))
		return false
	}
	//     user.p.guodun.xxxx ->      xxxx
	//  : user.p.guodun.user.evm.hash -> user.evm.hash     evm
	cfg := e.api.GetConfig()
	exec := cfg.GetParaExec(tx.Execer)
	//    1: (                 )
	if bytes.Equal(keyExecer, exec) {
		return true
	}
	//          dom fork
	// manage  key   config
	// token    key   mavl-create-token-
	if !cfg.IsFork(e.height, "ForkExecKey") {
		if bytes.Equal(exec, []byte("manage")) && bytes.Equal(keyExecer, []byte("config")) {
			return true
		}
		if bytes.Equal(exec, []byte("token")) {
			if bytes.HasPrefix(key, []byte("mavl-create-token-")) {
				return true
			}
		}
	}
	//     ，        ，
	//            ，
	//  execaddr           ，     realExecer
	keyExecAddr, ok := types.GetExecKey(key)
	if ok && keyExecAddr == drivers.ExecAddress(string(tx.Execer)) {
		return true
	}
	//         ，       ，    :
	//
	execdriver := keyExecer
	if ok && keyExecAddr == drivers.ExecAddress(string(realExecer)) {
		//  user.p.xxx.token       token
		execdriver = realExecer
	}
	//  loadDriver    ，            index
	//      driver.Allow     index      ，
	c := e.loadDriver(&types.Transaction{Execer: execdriver}, index)
	//   -> friend
	return c.IsFriend(execdriver, key, tx)
}

func isAllowLocalKey(cfg *types.DplatformOSConfig, execer []byte, key []byte) error {
	err := isAllowLocalKey2(cfg, execer, key)
	if err != nil {
		realexec := types.GetRealExecName(execer)
		if !bytes.Equal(realexec, execer) {
			err2 := isAllowLocalKey2(cfg, realexec, key)
			err = errors.Wrapf(err2, "1st check err: %s. 2nd check err", err.Error())
		}
		if err != nil {
			elog.Error("isAllowLocalKey failed", "err", err.Error())
			return errors.Cause(err)
		}

	}
	return nil
}

func isAllowLocalKey2(cfg *types.DplatformOSConfig, execer []byte, key []byte) error {
	if len(execer) < 1 {
		return errors.Wrap(types.ErrLocalPrefix, "execer empty")
	}
	minkeylen := len(types.LocalPrefix) + len(execer) + 2
	if len(key) <= minkeylen {
		err := errors.Wrapf(types.ErrLocalKeyLen, "isAllowLocalKey too short. key=%s exec=%s", string(key), string(execer))
		return err
	}
	if key[minkeylen-1] != '-' || key[len(types.LocalPrefix)] != '-' {
		err := errors.Wrapf(types.ErrLocalPrefix,
			"isAllowLocalKey prefix last char or separator is not '-'. key=%s exec=%s minkeylen=%d title=%s",
			string(key), string(execer), minkeylen, cfg.GetTitle())
		return err
	}
	if !bytes.HasPrefix(key, types.LocalPrefix) {
		err := errors.Wrapf(types.ErrLocalPrefix, "isAllowLocalKey common prefix not match. key=%s exec=%s",
			string(key), string(execer))
		return err
	}
	if !bytes.HasPrefix(key[len(types.LocalPrefix)+1:], execer) {
		err := errors.Wrapf(types.ErrLocalPrefix, "isAllowLocalKey key prefix not match. key=%s exec=%s",
			string(key), string(execer))
		return err
	}
	return nil
}
