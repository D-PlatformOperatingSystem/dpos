// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/gencode/base"
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/types"
)

func init() {

	base.RegisterCodeFile(rpcCodeFile{})
}

type rpcCodeFile struct {
	base.DappCodeFile
}

func (c rpcCodeFile) GetDirName() string {

	return "rpc"
}

func (c rpcCodeFile) GetFiles() map[string]string {

	return map[string]string{
		rpcName:   rpcContent,
		typesName: typesContent,
	}
}

func (c rpcCodeFile) GetFileReplaceTags() []string {

	return []string{types.TagExecName, types.TagImportPath, types.TagClassName}
}

var (
	rpcName    = "rpc.go"
	rpcContent = `package rpc


/* 
 *   json rpc grpc service  
 * json rpc Jrpc        
 * grpc  channelClient        
*/

`

	typesName    = "types.go"
	typesContent = `package rpc

import (
	${EXECNAME}types "${IMPORTPATH}/${EXECNAME}/types"
	rpctypes "github.com/D-PlatformOperatingSystem/dpos/rpc/types"
)

/* 
 * rpc          
*/

//   grpc service  
type channelClient struct {
	rpctypes.ChannelClient
}

// Jrpc   json rpc    
type Jrpc struct {
	cli *channelClient
}

// Grpc grpc
type Grpc struct {
	*channelClient
}

// Init init rpc
func Init(name string, s rpctypes.RPCServer) {
	cli := &channelClient{}
	grpc := &Grpc{channelClient: cli}
	cli.Init(name, s, &Jrpc{cli: cli}, grpc)
	//  grpc service   grpc server，       pb.go  
	${EXECNAME}types.Register${CLASSNAME}Server(s.GRPC(), grpc)
}`
)
