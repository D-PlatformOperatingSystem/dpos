// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"sync/atomic"
	"time"
)

var deltaTime int64

//NtpHosts ntp hosts
var NtpHosts = []string{
	"ntp.aliyun.com:123",
	"time1.cloud.tencent.com:123",
	"time.ustc.edu.cn:123",
	"cn.ntp.org.cn:123",
	"time.apple.com:123",
}

//SetTimeDelta realtime - localtime
//  60s
//       ，
func SetTimeDelta(dt int64) {
	if dt > 300*int64(time.Second) || dt < -300*int64(time.Second) {
		dt = 0
	}
	atomic.StoreInt64(&deltaTime, dt)
}

//Now
func Now() time.Time {
	dt := time.Duration(atomic.LoadInt64(&deltaTime))
	return time.Now().Add(dt)
}

//Since Since
func Since(t time.Time) time.Duration {
	return Now().Sub(t)
}
