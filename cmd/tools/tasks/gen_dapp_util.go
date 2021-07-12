// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tasks

import (
	"fmt"
	"regexp"
	"strings"

	sysutil "github.com/D-PlatformOperatingSystem/dpos/util"
)

type actionInfoItem struct {
	memberName string
	memberType string
}

/**
      Action         ，         ：
1.        proto  
2.     ，     Action     
3.        Action  oneof Value   
4.       oneof Value    ，         
5.             ，       
*/
func readDappActionFromProto(protoContent, actionName string) ([]*actionInfoItem, error) {

	//         ActionName       ，         
	index := strings.Index(protoContent, actionName)
	if index < 0 {
		return nil, fmt.Errorf("action %s Not Existed", actionName)
	}
	expr := fmt.Sprintf(`\s*oneof\s+value\s*{\s+([\w\s=;]*)\}`)
	reg := regexp.MustCompile(expr)
	oneOfValueStrs := reg.FindAllStringSubmatch(protoContent, index)

	expr = fmt.Sprintf(`\s+(\w+)([\s\w]+)=\s+(\d+);`)
	reg = regexp.MustCompile(expr)
	members := reg.FindAllStringSubmatch(oneOfValueStrs[0][0], -1)

	actionInfos := make([]*actionInfoItem, 0)
	for _, member := range members {
		memberType := strings.Replace(member[1], " ", "", -1)
		memberName := strings.Replace(member[2], " ", "", -1)
		//   proto  pb.go   ，           
		memberName, _ = sysutil.MakeStringToUpper(memberName, 0, 1)
		actionInfos = append(actionInfos, &actionInfoItem{
			memberName: memberName,
			memberType: memberType,
		})
	}
	if len(actionInfos) == 0 {
		return nil, fmt.Errorf("can Not Find %s Member Info", actionName)
	}
	return actionInfos, nil
}

func formatExecContent(infos []*actionInfoItem, dappName string) string {

	fnFmtStr := `func (${EXEC_OBJECT} *%s) Exec_%s(payload *%stypes.%s, tx *types.Transaction, index int) (*types.Receipt, error) {
	receipt := &types.Receipt{Ty: types.ExecOk}
	//implement code
	return receipt, nil
}

`
	content := ""
	for _, info := range infos {
		content += fmt.Sprintf(fnFmtStr, dappName, info.memberName, dappName, info.memberType)
	}

	return content
}

func formatExecLocalContent(infos []*actionInfoItem, dappName string) string {

	fnFmtStr := `func (${EXEC_OBJECT} *%s) ExecLocal_%s(payload *%stypes.%s, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	dbSet := &types.LocalDBSet{}
	//implement code, add customize kv to dbSet...
	
	//auto gen for localdb auto rollback
	return ${EXEC_OBJECT}.addAutoRollBack(tx, dbSet.KV), nil  
}

`
	content := ""
	for _, info := range infos {
		content += fmt.Sprintf(fnFmtStr, dappName, info.memberName, dappName, info.memberType)
	}

	return content
}

func formatExecDelLocalContent(infos []*actionInfoItem, dappName string) string {

	fnFmtStr := `func (${EXEC_OBJECT} *%s) ExecDelLocal_%s(payload *%stypes.%s, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var dbSet *types.LocalDBSet
	//implement code
	return dbSet, nil
}

`
	content := ""
	for _, info := range infos {
		content += fmt.Sprintf(fnFmtStr, dappName, info.memberName, dappName, info.memberType)
	}

	return content
}

//       TyLog+ActionName + ActionMemberName
func buildActionLogTypeText(infos []*actionInfoItem, className string) (text string) {
	items := fmt.Sprintf("TyUnknownLog = iota + 100\n")
	for _, info := range infos {
		items += fmt.Sprintf("Ty%sLog\n", info.memberName)
	}
	text = fmt.Sprintf("const (\n%s)\n", items)
	return
}

//       ActionName + ActionMemberName
func buildActionIDText(infos []*actionInfoItem, className string) (text string) {

	items := fmt.Sprintf("TyUnknowAction = iota + 100\n")
	for _, info := range infos {
		items += fmt.Sprintf("Ty%sAction\n", info.memberName)
	}
	items += "\n"
	for _, info := range infos {
		items += fmt.Sprintf("Name%sAction = \"%s\"\n", info.memberName, info.memberName)
	}

	text = fmt.Sprintf("const (\n%s)\n", items)
	return
}

//    map[string]int32
func buildTypeMapText(infos []*actionInfoItem, className string) (text string) {
	var items string
	for _, info := range infos {
		items += fmt.Sprintf("Name%sAction: Ty%sAction,\n", info.memberName, info.memberName)
	}
	text = fmt.Sprintf("map[string]int32{\n%s}", items)
	return
}

//    map[string]*types.LogInfo
func buildLogMapText() (text string) {
	text = fmt.Sprintf("map[int64]*types.LogInfo{\n\t//LogID:	{Ty: reflect.TypeOf(LogStruct), Name: LogName},\n}")
	return
}
