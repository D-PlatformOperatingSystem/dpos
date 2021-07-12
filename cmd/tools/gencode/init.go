// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gencode

import (
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/gencode/base"
	_ "github.com/D-PlatformOperatingSystem/dpos/cmd/tools/gencode/dappcode" //init dapp code
)

//GetCodeFilesWithType get code file with type
func GetCodeFilesWithType(typeName string) []base.ICodeFile {

	if fileArr, ok := base.CodeFileManager[typeName]; ok {

		return fileArr
	}

	return nil
}
