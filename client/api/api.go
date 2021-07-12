//

package api

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/D-PlatformOperatingSystem/dpos/client"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

/*
            :
1.    client.QueueProtocolAPI
2.    grpc

   client.QueueProtocolAPI   grpc            ，
    ，

    :
1. GetBlockByHash
2. GetLastBlockHash
3. GetRandNum
*/

//ErrAPIEnv api        ，       ，          retry
var errAPIEnv = errors.New("ErrAPIEnv")

//ExecutorAPI
//              ，  ，
type ExecutorAPI interface {
	GetBlockByHashes(param *types.ReqHashes) (*types.BlockDetails, error)
	GetRandNum(param *types.ReqRandHash) ([]byte, error)
	QueryTx(param *types.ReqHash) (*types.TransactionDetail, error)
	IsErr() bool
}

type mainChainAPI struct {
	api     client.QueueProtocolAPI
	errflag int32
}

//New
func New(api client.QueueProtocolAPI, grpcClient types.DplatformOSClient) ExecutorAPI {
	types.AssertConfig(api)
	types := api.GetConfig()
	if types.IsPara() {
		return newParaChainAPI(api, grpcClient)
	}
	return &mainChainAPI{api: api}
}

func (api *mainChainAPI) QueryTx(param *types.ReqHash) (*types.TransactionDetail, error) {
	data, err := api.api.QueryTx(param)
	return data, seterr(err, &api.errflag)
}

func (api *mainChainAPI) IsErr() bool {
	return atomic.LoadInt32(&api.errflag) == 1
}

func (api *mainChainAPI) GetRandNum(param *types.ReqRandHash) ([]byte, error) {
	msg, err := api.api.Query(param.ExecName, "RandNumHash", param)
	if err != nil {
		return nil, seterr(err, &api.errflag)
	}
	reply, ok := msg.(*types.ReplyHash)
	if !ok {
		return nil, types.ErrTypeAsset
	}
	return reply.Hash, nil
}

func (api *mainChainAPI) GetBlockByHashes(param *types.ReqHashes) (*types.BlockDetails, error) {
	data, err := api.api.GetBlockByHashes(param)
	return data, seterr(err, &api.errflag)
}

type paraChainAPI struct {
	api        client.QueueProtocolAPI
	grpcClient types.DplatformOSClient
	errflag    int32
}

func newParaChainAPI(api client.QueueProtocolAPI, grpcClient types.DplatformOSClient) ExecutorAPI {
	return &paraChainAPI{api: api, grpcClient: grpcClient}
}

func (api *paraChainAPI) IsErr() bool {
	return atomic.LoadInt32(&api.errflag) == 1
}

func (api *paraChainAPI) QueryTx(param *types.ReqHash) (*types.TransactionDetail, error) {
	data, err := api.grpcClient.QueryTransaction(context.Background(), param)
	if err != nil {
		err = errAPIEnv
	}
	return data, seterr(err, &api.errflag)
}

func (api *paraChainAPI) GetRandNum(param *types.ReqRandHash) ([]byte, error) {
	reply, err := api.grpcClient.QueryRandNum(context.Background(), param)
	if err != nil {
		err = errAPIEnv
		return nil, seterr(err, &api.errflag)
	}
	return reply.Hash, nil
}

func (api *paraChainAPI) GetBlockByHashes(param *types.ReqHashes) (*types.BlockDetails, error) {
	data, err := api.grpcClient.GetBlockByHashes(context.Background(), param)
	if err != nil {
		err = errAPIEnv
	}
	return data, seterr(err, &api.errflag)
}

func seterr(err error, flag *int32) error {
	if IsGrpcError(err) || IsQueueError(err) {
		atomic.StoreInt32(flag, 1)
	}
	return err
}

//IsGrpcError     api   ，   rpc
func IsGrpcError(err error) bool {
	if err == nil {
		return false
	}
	if err == errAPIEnv {
		return true
	}
	if grpc.Code(err) == codes.Unknown {
		return false
	}
	return true
}

//IsQueueError
func IsQueueError(err error) bool {
	if err == nil {
		return false
	}
	if err == errAPIEnv {
		return true
	}
	if err == queue.ErrQueueTimeout ||
		err == queue.ErrQueueChannelFull ||
		err == queue.ErrIsQueueClosed {
		return true
	}
	return false
}

//IsFatalError
func IsFatalError(err error) bool {
	if err == nil {
		return false
	}
	if err == errAPIEnv {
		return true
	}
	if err == types.ErrConsensusHashErr {
		return true
	}
	return false
}

//IsAPIEnvError    api
func IsAPIEnvError(err error) bool {
	return IsGrpcError(err) || IsQueueError(err) || IsFatalError(err)
}
