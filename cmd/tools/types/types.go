// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package types
package types

//const
const (
	//      go vendor   ，
	KeyImportPackage            = "import_package"
	KeyCreateSimpleExecProject  = "create_simple_project"
	KeyCreateAdvanceExecProject = "create_advance_project"
	KeyConfigFolder             = "config_folder"
	KeyProjectName              = "project_name"
	KeyClassName                = "class_name"
	KeyExecutorName             = "executor_name"
	KeyActionName               = "action_name"
	KeyProtobufFile             = "protobuf_file"
	KeyTemplateFilePath         = "template_file_path"
	KeyUpdateInit               = "update_init"
	KeyCreatePlugin             = "create_plugin"
	KeyGenDapp                  = "generate_dapp"
	KeyDappOutDir               = "generate_dapp_out_dir"

	DefCpmConfigfile = "dplatformos.cpm.toml"

	TagGoPath          = "${GOPATH}"
	TagProjectName     = "${PROJECTNAME}"   //
	TagClassName       = "${CLASSNAME}"     //
	TagClassTypeName   = "${CLASSTYPENAME}" // ClassName+Type
	TagActionName      = "${ACTIONNAME}"    //        Action
	TagExecName        = "${EXECNAME}"      //
	TagProjectPath     = "${PROJECTPATH}"
	TagExecNameFB      = "${EXECNAME_FB}" //
	TagTyLogActionType = "${TYLOGACTIONTYPE}"
	TagActionIDText    = "${ACTIONIDTEXT}"
	TagLogMapText      = "${LOGMAPTEXT}"
	TagTypeMapText     = "${TYPEMAPTEXT}"
	TagTypeName        = "${TYPENAME}"

	//TagImport
	TagImportPath = "${IMPORTPATH}"

	//Tag proto file
	TagProtoFileContent = "${PROTOFILECONTENT}"
	TagProtoFileAppend  = "${PROTOFILEAPPEND}"

	//Tag exec.go file
	TagExecFileContent         = "${EXECFILECONTENT}"
	TagExecLocalFileContent    = "${EXECLOCALFILECONTENT}"
	TagExecDelLocalFileContent = "${EXECDELLOCALFILECONTENT}"

	TagExecObject = "${EXEC_OBJECT}" //          ,
)
