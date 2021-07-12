// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package log
package log

import (
	"os"

	"github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var (
	//           ，          ，
	fileHandler    *log15.Handler
	consoleHandler *log15.Handler
)

func init() {
	//resetWithLogLevel("error")
}

//SetLogLevel
func SetLogLevel(logLevel string) {
	handler := getConsoleLogHandler(logLevel)
	(*handler).SetMaxLevel(int(getLevel(logLevel)))
	log15.Root().SetHandler(*handler)
}

//SetFileLog
func SetFileLog(log *types.Log) {
	if log == nil {
		log = &types.Log{LogFile: "logs/dplatformos.log"}
	}
	if log.LogFile == "" {
		SetLogLevel(log.LogConsoleLevel)
	} else {
		resetLog(log)
	}
}

//          Handler，
func resetLog(log *types.Log) {
	fillDefaultValue(log)
	log15.Root().SetHandler(log15.MultiHandler(*getConsoleLogHandler(log.LogConsoleLevel), *getFileLogHandler(log)))
}

//         error  ，
func fillDefaultValue(log *types.Log) {
	if log.Loglevel == "" {
		log.Loglevel = log15.LvlError.String()
	}
	if log.LogConsoleLevel == "" {
		log.LogConsoleLevel = log15.LvlError.String()
	}
}

func isWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}

func getConsoleLogHandler(logLevel string) *log15.Handler {
	if consoleHandler != nil {
		return consoleHandler
	}
	format := log15.TerminalFormat()
	if isWindows() {
		format = log15.LogfmtFormat()
	}
	stdouth := log15.LvlFilterHandler(
		getLevel(logLevel),
		log15.StreamHandler(os.Stdout, format),
	)

	consoleHandler = &stdouth

	return &stdouth
}

func getFileLogHandler(log *types.Log) *log15.Handler {
	if fileHandler != nil {
		return fileHandler
	}

	rotateLogger := &lumberjack.Logger{
		Filename:   log.LogFile,
		MaxSize:    int(log.MaxFileSize),
		MaxBackups: int(log.MaxBackups),
		MaxAge:     int(log.MaxAge),
		LocalTime:  log.LocalTime,
		Compress:   log.Compress,
	}

	fileh := log15.LvlFilterHandler(
		getLevel(log.Loglevel),
		log15.StreamHandler(rotateLogger, log15.LogfmtFormat()),
	)

	//          、
	if log.CallerFile {
		fileh = log15.CallerFileHandler(fileh)
	}
	if log.CallerFunction {
		fileh = log15.CallerFuncHandler(fileh)
	}

	fileHandler = &fileh

	return &fileh
}

func getLevel(lvlString string) log15.Lvl {
	lvl, err := log15.LvlFromString(lvlString)
	if err != nil {
		//              error
		return log15.LvlError
	}
	return lvl
}

//New new
func New(ctx ...interface{}) log15.Logger {
	return NewMain(ctx...)
}

//NewMain new
func NewMain(ctx ...interface{}) log15.Logger {
	return log15.Root().New(ctx...)
}
