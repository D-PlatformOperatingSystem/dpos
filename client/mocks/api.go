// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	queue "github.com/D-PlatformOperatingSystem/dpos/queue"
	mock "github.com/stretchr/testify/mock"

	types "github.com/D-PlatformOperatingSystem/dpos/types"
)

// QueueProtocolAPI is an autogenerated mock type for the QueueProtocolAPI type
type QueueProtocolAPI struct {
	mock.Mock
}

// AddPushSubscribe provides a mock function with given fields: param
func (_m *QueueProtocolAPI) AddPushSubscribe(param *types.PushSubscribeReq) (*types.ReplySubscribePush, error) {
	ret := _m.Called(param)

	var r0 *types.ReplySubscribePush
	if rf, ok := ret.Get(0).(func(*types.PushSubscribeReq) *types.ReplySubscribePush); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplySubscribePush)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.PushSubscribeReq) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Close provides a mock function with given fields:
func (_m *QueueProtocolAPI) Close() {
	_m.Called()
}

// CloseQueue provides a mock function with given fields:
func (_m *QueueProtocolAPI) CloseQueue() (*types.Reply, error) {
	ret := _m.Called()

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func() *types.Reply); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExecWallet provides a mock function with given fields: param
func (_m *QueueProtocolAPI) ExecWallet(param *types.ChainExecutor) (types.Message, error) {
	ret := _m.Called(param)

	var r0 types.Message
	if rf, ok := ret.Get(0).(func(*types.ChainExecutor) types.Message); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Message)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ChainExecutor) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExecWalletFunc provides a mock function with given fields: driver, funcname, param
func (_m *QueueProtocolAPI) ExecWalletFunc(driver string, funcname string, param types.Message) (types.Message, error) {
	ret := _m.Called(driver, funcname, param)

	var r0 types.Message
	if rf, ok := ret.Get(0).(func(string, string, types.Message) types.Message); ok {
		r0 = rf(driver, funcname, param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Message)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, types.Message) error); ok {
		r1 = rf(driver, funcname, param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAddrOverview provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetAddrOverview(param *types.ReqAddr) (*types.AddrOverview, error) {
	ret := _m.Called(param)

	var r0 *types.AddrOverview
	if rf, ok := ret.Get(0).(func(*types.ReqAddr) *types.AddrOverview); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.AddrOverview)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqAddr) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlockByHashes provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetBlockByHashes(param *types.ReqHashes) (*types.BlockDetails, error) {
	ret := _m.Called(param)

	var r0 *types.BlockDetails
	if rf, ok := ret.Get(0).(func(*types.ReqHashes) *types.BlockDetails); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.BlockDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqHashes) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlockBySeq provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetBlockBySeq(param *types.Int64) (*types.BlockSeq, error) {
	ret := _m.Called(param)

	var r0 *types.BlockSeq
	if rf, ok := ret.Get(0).(func(*types.Int64) *types.BlockSeq); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.BlockSeq)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.Int64) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlockHash provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetBlockHash(param *types.ReqInt) (*types.ReplyHash, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyHash
	if rf, ok := ret.Get(0).(func(*types.ReqInt) *types.ReplyHash); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyHash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqInt) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlockOverview provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetBlockOverview(param *types.ReqHash) (*types.BlockOverview, error) {
	ret := _m.Called(param)

	var r0 *types.BlockOverview
	if rf, ok := ret.Get(0).(func(*types.ReqHash) *types.BlockOverview); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.BlockOverview)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqHash) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlockSequences provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetBlockSequences(param *types.ReqBlocks) (*types.BlockSequences, error) {
	ret := _m.Called(param)

	var r0 *types.BlockSequences
	if rf, ok := ret.Get(0).(func(*types.ReqBlocks) *types.BlockSequences); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.BlockSequences)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqBlocks) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlocks provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetBlocks(param *types.ReqBlocks) (*types.BlockDetails, error) {
	ret := _m.Called(param)

	var r0 *types.BlockDetails
	if rf, ok := ret.Get(0).(func(*types.ReqBlocks) *types.BlockDetails); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.BlockDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqBlocks) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetConfig provides a mock function with given fields:
func (_m *QueueProtocolAPI) GetConfig() *types.DplatformOSConfig {
	ret := _m.Called()

	var r0 *types.DplatformOSConfig
	if rf, ok := ret.Get(0).(func() *types.DplatformOSConfig); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.DplatformOSConfig)
		}
	}

	return r0
}

// GetHeaders provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetHeaders(param *types.ReqBlocks) (*types.Headers, error) {
	ret := _m.Called(param)

	var r0 *types.Headers
	if rf, ok := ret.Get(0).(func(*types.ReqBlocks) *types.Headers); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Headers)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqBlocks) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLastBlockMainSequence provides a mock function with given fields:
func (_m *QueueProtocolAPI) GetLastBlockMainSequence() (*types.Int64, error) {
	ret := _m.Called()

	var r0 *types.Int64
	if rf, ok := ret.Get(0).(func() *types.Int64); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Int64)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLastBlockSequence provides a mock function with given fields:
func (_m *QueueProtocolAPI) GetLastBlockSequence() (*types.Int64, error) {
	ret := _m.Called()

	var r0 *types.Int64
	if rf, ok := ret.Get(0).(func() *types.Int64); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Int64)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLastHeader provides a mock function with given fields:
func (_m *QueueProtocolAPI) GetLastHeader() (*types.Header, error) {
	ret := _m.Called()

	var r0 *types.Header
	if rf, ok := ret.Get(0).(func() *types.Header); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Header)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLastMempool provides a mock function with given fields:
func (_m *QueueProtocolAPI) GetLastMempool() (*types.ReplyTxList, error) {
	ret := _m.Called()

	var r0 *types.ReplyTxList
	if rf, ok := ret.Get(0).(func() *types.ReplyTxList); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyTxList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetMainSequenceByHash provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetMainSequenceByHash(param *types.ReqHash) (*types.Int64, error) {
	ret := _m.Called(param)

	var r0 *types.Int64
	if rf, ok := ret.Get(0).(func(*types.ReqHash) *types.Int64); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Int64)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqHash) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetMempool provides a mock function with given fields: req
func (_m *QueueProtocolAPI) GetMempool(req *types.ReqGetMempool) (*types.ReplyTxList, error) {
	ret := _m.Called(req)

	var r0 *types.ReplyTxList
	if rf, ok := ret.Get(0).(func(*types.ReqGetMempool) *types.ReplyTxList); ok {
		r0 = rf(req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyTxList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqGetMempool) error); ok {
		r1 = rf(req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNetInfo provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetNetInfo(param *types.P2PGetNetInfoReq) (*types.NodeNetInfo, error) {
	ret := _m.Called(param)

	var r0 *types.NodeNetInfo
	if rf, ok := ret.Get(0).(func(*types.P2PGetNetInfoReq) *types.NodeNetInfo); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.NodeNetInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.P2PGetNetInfoReq) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetParaTxByHeight provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetParaTxByHeight(param *types.ReqParaTxByHeight) (*types.ParaTxDetails, error) {
	ret := _m.Called(param)

	var r0 *types.ParaTxDetails
	if rf, ok := ret.Get(0).(func(*types.ReqParaTxByHeight) *types.ParaTxDetails); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ParaTxDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqParaTxByHeight) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetParaTxByTitle provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetParaTxByTitle(param *types.ReqParaTxByTitle) (*types.ParaTxDetails, error) {
	ret := _m.Called(param)

	var r0 *types.ParaTxDetails
	if rf, ok := ret.Get(0).(func(*types.ReqParaTxByTitle) *types.ParaTxDetails); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ParaTxDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqParaTxByTitle) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetProperFee provides a mock function with given fields: req
func (_m *QueueProtocolAPI) GetProperFee(req *types.ReqProperFee) (*types.ReplyProperFee, error) {
	ret := _m.Called(req)

	var r0 *types.ReplyProperFee
	if rf, ok := ret.Get(0).(func(*types.ReqProperFee) *types.ReplyProperFee); ok {
		r0 = rf(req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyProperFee)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqProperFee) error); ok {
		r1 = rf(req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPushSeqLastNum provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetPushSeqLastNum(param *types.ReqString) (*types.Int64, error) {
	ret := _m.Called(param)

	var r0 *types.Int64
	if rf, ok := ret.Get(0).(func(*types.ReqString) *types.Int64); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Int64)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqString) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSequenceByHash provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetSequenceByHash(param *types.ReqHash) (*types.Int64, error) {
	ret := _m.Called(param)

	var r0 *types.Int64
	if rf, ok := ret.Get(0).(func(*types.ReqHash) *types.Int64); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Int64)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqHash) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTransactionByAddr provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetTransactionByAddr(param *types.ReqAddr) (*types.ReplyTxInfos, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyTxInfos
	if rf, ok := ret.Get(0).(func(*types.ReqAddr) *types.ReplyTxInfos); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyTxInfos)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqAddr) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTransactionByHash provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetTransactionByHash(param *types.ReqHashes) (*types.TransactionDetails, error) {
	ret := _m.Called(param)

	var r0 *types.TransactionDetails
	if rf, ok := ret.Get(0).(func(*types.ReqHashes) *types.TransactionDetails); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.TransactionDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqHashes) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTxList provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetTxList(param *types.TxHashList) (*types.ReplyTxList, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyTxList
	if rf, ok := ret.Get(0).(func(*types.TxHashList) *types.ReplyTxList); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyTxList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.TxHashList) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsNtpClockSync provides a mock function with given fields:
func (_m *QueueProtocolAPI) IsNtpClockSync() (*types.Reply, error) {
	ret := _m.Called()

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func() *types.Reply); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsSync provides a mock function with given fields:
func (_m *QueueProtocolAPI) IsSync() (*types.Reply, error) {
	ret := _m.Called()

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func() *types.Reply); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListPushes provides a mock function with given fields:
func (_m *QueueProtocolAPI) ListPushes() (*types.PushSubscribes, error) {
	ret := _m.Called()

	var r0 *types.PushSubscribes
	if rf, ok := ret.Get(0).(func() *types.PushSubscribes); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.PushSubscribes)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LoadParaTxByTitle provides a mock function with given fields: param
func (_m *QueueProtocolAPI) LoadParaTxByTitle(param *types.ReqHeightByTitle) (*types.ReplyHeightByTitle, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyHeightByTitle
	if rf, ok := ret.Get(0).(func(*types.ReqHeightByTitle) *types.ReplyHeightByTitle); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyHeightByTitle)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqHeightByTitle) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LocalBegin provides a mock function with given fields: param
func (_m *QueueProtocolAPI) LocalBegin(param *types.Int64) error {
	ret := _m.Called(param)

	var r0 error
	if rf, ok := ret.Get(0).(func(*types.Int64) error); ok {
		r0 = rf(param)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// LocalClose provides a mock function with given fields: param
func (_m *QueueProtocolAPI) LocalClose(param *types.Int64) error {
	ret := _m.Called(param)

	var r0 error
	if rf, ok := ret.Get(0).(func(*types.Int64) error); ok {
		r0 = rf(param)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// LocalCommit provides a mock function with given fields: param
func (_m *QueueProtocolAPI) LocalCommit(param *types.Int64) error {
	ret := _m.Called(param)

	var r0 error
	if rf, ok := ret.Get(0).(func(*types.Int64) error); ok {
		r0 = rf(param)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// LocalGet provides a mock function with given fields: param
func (_m *QueueProtocolAPI) LocalGet(param *types.LocalDBGet) (*types.LocalReplyValue, error) {
	ret := _m.Called(param)

	var r0 *types.LocalReplyValue
	if rf, ok := ret.Get(0).(func(*types.LocalDBGet) *types.LocalReplyValue); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.LocalReplyValue)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.LocalDBGet) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LocalList provides a mock function with given fields: param
func (_m *QueueProtocolAPI) LocalList(param *types.LocalDBList) (*types.LocalReplyValue, error) {
	ret := _m.Called(param)

	var r0 *types.LocalReplyValue
	if rf, ok := ret.Get(0).(func(*types.LocalDBList) *types.LocalReplyValue); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.LocalReplyValue)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.LocalDBList) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LocalNew provides a mock function with given fields: readOnly
func (_m *QueueProtocolAPI) LocalNew(readOnly bool) (*types.Int64, error) {
	ret := _m.Called(readOnly)

	var r0 *types.Int64
	if rf, ok := ret.Get(0).(func(bool) *types.Int64); ok {
		r0 = rf(readOnly)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Int64)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(bool) error); ok {
		r1 = rf(readOnly)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LocalRollback provides a mock function with given fields: param
func (_m *QueueProtocolAPI) LocalRollback(param *types.Int64) error {
	ret := _m.Called(param)

	var r0 error
	if rf, ok := ret.Get(0).(func(*types.Int64) error); ok {
		r0 = rf(param)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// LocalSet provides a mock function with given fields: param
func (_m *QueueProtocolAPI) LocalSet(param *types.LocalDBSet) error {
	ret := _m.Called(param)

	var r0 error
	if rf, ok := ret.Get(0).(func(*types.LocalDBSet) error); ok {
		r0 = rf(param)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NetProtocols provides a mock function with given fields: _a0
func (_m *QueueProtocolAPI) NetProtocols(_a0 *types.ReqNil) (*types.NetProtocolInfos, error) {
	ret := _m.Called(_a0)

	var r0 *types.NetProtocolInfos
	if rf, ok := ret.Get(0).(func(*types.ReqNil) *types.NetProtocolInfos); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.NetProtocolInfos)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqNil) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMessage provides a mock function with given fields: topic, msgid, data
func (_m *QueueProtocolAPI) NewMessage(topic string, msgid int64, data interface{}) *queue.Message {
	ret := _m.Called(topic, msgid, data)

	var r0 *queue.Message
	if rf, ok := ret.Get(0).(func(string, int64, interface{}) *queue.Message); ok {
		r0 = rf(topic, msgid, data)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*queue.Message)
		}
	}

	return r0
}

// Notify provides a mock function with given fields: topic, ty, data
func (_m *QueueProtocolAPI) Notify(topic string, ty int64, data interface{}) (*queue.Message, error) {
	ret := _m.Called(topic, ty, data)

	var r0 *queue.Message
	if rf, ok := ret.Get(0).(func(string, int64, interface{}) *queue.Message); ok {
		r0 = rf(topic, ty, data)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*queue.Message)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int64, interface{}) error); ok {
		r1 = rf(topic, ty, data)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PeerInfo provides a mock function with given fields: param
func (_m *QueueProtocolAPI) PeerInfo(param *types.P2PGetPeerReq) (*types.PeerList, error) {
	ret := _m.Called(param)

	var r0 *types.PeerList
	if rf, ok := ret.Get(0).(func(*types.P2PGetPeerReq) *types.PeerList); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.PeerList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.P2PGetPeerReq) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Query provides a mock function with given fields: driver, funcname, param
func (_m *QueueProtocolAPI) Query(driver string, funcname string, param types.Message) (types.Message, error) {
	ret := _m.Called(driver, funcname, param)

	var r0 types.Message
	if rf, ok := ret.Get(0).(func(string, string, types.Message) types.Message); ok {
		r0 = rf(driver, funcname, param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Message)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, types.Message) error); ok {
		r1 = rf(driver, funcname, param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// QueryChain provides a mock function with given fields: param
func (_m *QueueProtocolAPI) QueryChain(param *types.ChainExecutor) (types.Message, error) {
	ret := _m.Called(param)

	var r0 types.Message
	if rf, ok := ret.Get(0).(func(*types.ChainExecutor) types.Message); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Message)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ChainExecutor) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// QueryConsensus provides a mock function with given fields: param
func (_m *QueueProtocolAPI) QueryConsensus(param *types.ChainExecutor) (types.Message, error) {
	ret := _m.Called(param)

	var r0 types.Message
	if rf, ok := ret.Get(0).(func(*types.ChainExecutor) types.Message); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Message)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ChainExecutor) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// QueryConsensusFunc provides a mock function with given fields: driver, funcname, param
func (_m *QueueProtocolAPI) QueryConsensusFunc(driver string, funcname string, param types.Message) (types.Message, error) {
	ret := _m.Called(driver, funcname, param)

	var r0 types.Message
	if rf, ok := ret.Get(0).(func(string, string, types.Message) types.Message); ok {
		r0 = rf(driver, funcname, param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.Message)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, types.Message) error); ok {
		r1 = rf(driver, funcname, param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// QueryTx provides a mock function with given fields: param
func (_m *QueueProtocolAPI) QueryTx(param *types.ReqHash) (*types.TransactionDetail, error) {
	ret := _m.Called(param)

	var r0 *types.TransactionDetail
	if rf, ok := ret.Get(0).(func(*types.ReqHash) *types.TransactionDetail); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.TransactionDetail)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqHash) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendTx provides a mock function with given fields: param
func (_m *QueueProtocolAPI) SendTx(param *types.Transaction) (*types.Reply, error) {
	ret := _m.Called(param)

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func(*types.Transaction) *types.Reply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.Transaction) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StoreCommit provides a mock function with given fields: param
func (_m *QueueProtocolAPI) StoreCommit(param *types.ReqHash) (*types.ReplyHash, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyHash
	if rf, ok := ret.Get(0).(func(*types.ReqHash) *types.ReplyHash); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyHash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqHash) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StoreDel provides a mock function with given fields: param
func (_m *QueueProtocolAPI) StoreDel(param *types.StoreDel) (*types.ReplyHash, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyHash
	if rf, ok := ret.Get(0).(func(*types.StoreDel) *types.ReplyHash); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyHash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.StoreDel) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StoreGet provides a mock function with given fields: _a0
func (_m *QueueProtocolAPI) StoreGet(_a0 *types.StoreGet) (*types.StoreReplyValue, error) {
	ret := _m.Called(_a0)

	var r0 *types.StoreReplyValue
	if rf, ok := ret.Get(0).(func(*types.StoreGet) *types.StoreReplyValue); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.StoreReplyValue)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.StoreGet) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StoreGetTotalCoins provides a mock function with given fields: _a0
func (_m *QueueProtocolAPI) StoreGetTotalCoins(_a0 *types.IterateRangeByStateHash) (*types.ReplyGetTotalCoins, error) {
	ret := _m.Called(_a0)

	var r0 *types.ReplyGetTotalCoins
	if rf, ok := ret.Get(0).(func(*types.IterateRangeByStateHash) *types.ReplyGetTotalCoins); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyGetTotalCoins)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.IterateRangeByStateHash) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StoreList provides a mock function with given fields: param
func (_m *QueueProtocolAPI) StoreList(param *types.StoreList) (*types.StoreListReply, error) {
	ret := _m.Called(param)

	var r0 *types.StoreListReply
	if rf, ok := ret.Get(0).(func(*types.StoreList) *types.StoreListReply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.StoreListReply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.StoreList) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StoreMemSet provides a mock function with given fields: param
func (_m *QueueProtocolAPI) StoreMemSet(param *types.StoreSetWithSync) (*types.ReplyHash, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyHash
	if rf, ok := ret.Get(0).(func(*types.StoreSetWithSync) *types.ReplyHash); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyHash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.StoreSetWithSync) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StoreRollback provides a mock function with given fields: param
func (_m *QueueProtocolAPI) StoreRollback(param *types.ReqHash) (*types.ReplyHash, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyHash
	if rf, ok := ret.Get(0).(func(*types.ReqHash) *types.ReplyHash); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyHash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqHash) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StoreSet provides a mock function with given fields: param
func (_m *QueueProtocolAPI) StoreSet(param *types.StoreSetWithSync) (*types.ReplyHash, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyHash
	if rf, ok := ret.Get(0).(func(*types.StoreSetWithSync) *types.ReplyHash); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyHash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.StoreSetWithSync) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Version provides a mock function with given fields:
func (_m *QueueProtocolAPI) Version() (*types.VersionInfo, error) {
	ret := _m.Called()

	var r0 *types.VersionInfo
	if rf, ok := ret.Get(0).(func() *types.VersionInfo); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.VersionInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
