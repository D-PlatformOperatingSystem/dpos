// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

//GetMostCommit   commits      ，   string
func GetMostCommit(commits [][]byte) (int, string) {
	stats := make(map[string]int)
	n := len(commits)
	for i := 0; i < n; i++ {
		if _, ok := stats[string(commits[i])]; ok {
			stats[string(commits[i])]++
		} else {
			stats[string(commits[i])] = 1
		}
	}
	most := -1
	var key string
	for k, v := range stats {
		if v > most {
			most = v
			key = k
		}
	}
	return most, key
}

//IsCommitDone     most   total*2/3
func IsCommitDone(total, most int) bool {
	return 3*most > 2*total
}
