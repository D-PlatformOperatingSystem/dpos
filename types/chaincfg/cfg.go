// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package chaincfg   dplatformos
package chaincfg

var configMap = make(map[string]string)

// Register
func Register(name, cfg string) {
	if _, ok := configMap[name]; ok {
		panic("chain default config name " + name + " is exist")
	}
	configMap[name] = cfg
}

// Load
func Load(name string) string {
	return configMap[name]
}

// LoadAll
func LoadAll() map[string]string {
	return configMap
}
