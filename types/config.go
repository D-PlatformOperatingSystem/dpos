// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"

	"fmt"

	"github.com/D-PlatformOperatingSystem/dpos/types/chaincfg"
	tml "github.com/BurntSushi/toml"
)

//Create ...
type Create func(cfg *DplatformOSConfig)

//          ，
var (
	AllowUserExec = [][]byte{ExecerNone}
	EmptyValue    = []byte("FFFFFFFFemptyBVBiCj5jvE15pEiwro8TQRGnJSNsJF") //
	cliSysParam   = make(map[string]*DplatformOSConfig)                     // map key is title
	regModuleInit = make(map[string]Create)
	regExecInit   = make(map[string]Create)
	runonce       = sync.Once{}
)

// coin conversation
const (
	Coin            int64 = 1e8
	MaxCoin         int64 = 1e17
	MaxTxSize             = 100000 //100K
	MaxTxGroupSize  int32 = 20
	MaxBlockSize          = 20000000 //20M
	MaxTxsPerBlock        = 100000
	TokenPrecision  int64 = 1e8
	MaxTokenBalance int64 = 900 * 1e8 * TokenPrecision //900
	DefaultMinFee   int64 = 1e5
)

//DplatformOSConfig ...
type DplatformOSConfig struct {
	mcfg            *Config
	scfg            *ConfigSubModule
	minerExecs      []string
	title           string
	mu              sync.Mutex
	chainConfig     map[string]interface{}
	mver            *mversion
	coinSymbol      string
	forks           *Forks
	enableCheckFork bool
}

//ChainParam
type ChainParam struct {
	MaxTxNumber  int64
	PowLimitBits uint32
}

//RegFork Reg
func RegFork(name string, create Create) {
	if create == nil {
		panic("config: Register Module Init is nil")
	}
	if _, dup := regModuleInit[name]; dup {
		panic("config: Register Init called twice for driver " + name)
	}
	regModuleInit[name] = create
}

//RegForkInit ...
func RegForkInit(cfg *DplatformOSConfig) {
	for _, item := range regModuleInit {
		item(cfg)
	}
}

//RegExec ...
func RegExec(name string, create Create) {
	if create == nil {
		panic("config: Register Exec Init is nil")
	}
	if _, dup := regExecInit[name]; dup {
		panic("config: Register Exec called twice for driver " + name)
	}
	regExecInit[name] = create
}

//RegExecInit ...
func RegExecInit(cfg *DplatformOSConfig) {
	runonce.Do(func() {
		for _, item := range regExecInit {
			item(cfg)
		}
	})
}

//NewDplatformOSConfig ...
func NewDplatformOSConfig(cfgstring string) *DplatformOSConfig {
	dplatformosCfg := NewDplatformOSConfigNoInit(cfgstring)
	dplatformosCfg.dplatformosCfgInit(dplatformosCfg.mcfg)
	return dplatformosCfg
}

//NewDplatformOSConfigNoInit ...
func NewDplatformOSConfigNoInit(cfgstring string) *DplatformOSConfig {
	cfg, sub := InitCfgString(cfgstring)
	dplatformosCfg := &DplatformOSConfig{
		mcfg:        cfg,
		scfg:        sub,
		minerExecs:  []string{"ticket"}, //       ，     ，  ticket
		title:       cfg.Title,
		chainConfig: make(map[string]interface{}),
		coinSymbol:  "dpos",
		forks:       &Forks{make(map[string]int64)},
	}
	//        fork    DplatformOSConfig ，        toml
	dplatformosCfg.setDefaultConfig()
	dplatformosCfg.setFlatConfig(cfgstring)
	dplatformosCfg.setMver(cfgstring)
	// TODO        NewDplatformOSConfig
	RegForkInit(dplatformosCfg)
	RegExecInit(dplatformosCfg)
	return dplatformosCfg
}

//GetModuleConfig ...
func (c *DplatformOSConfig) GetModuleConfig() *Config {
	return c.mcfg
}

//GetSubConfig ...
func (c *DplatformOSConfig) GetSubConfig() *ConfigSubModule {
	return c.scfg
}

//EnableCheckFork ...
func (c *DplatformOSConfig) EnableCheckFork(enable bool) {
	c.enableCheckFork = false
}

//GetForks ...
func (c *DplatformOSConfig) GetForks() (map[string]int64, error) {
	if c.forks == nil {
		return nil, ErrNotFound
	}
	return c.forks.forks, nil
}

func (c *DplatformOSConfig) setDefaultConfig() {
	c.S("TestNet", false)
	c.SetMinFee(DefaultMinFee)
	for key, cfg := range chaincfg.LoadAll() {
		c.S("cfg."+key, cfg)
	}
	//   error   ，
	if !c.HasConf("cfg.dplatformos") {
		c.S("cfg.dplatformos", "")
	}
	if !c.HasConf("cfg.local") {
		c.S("cfg.local", "")
	}
}

func (c *DplatformOSConfig) setFlatConfig(cfgstring string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cfg := make(map[string]interface{})
	if _, err := tml.Decode(cfgstring, &cfg); err != nil {
		panic(err)
	}
	flat := FlatConfig(cfg)
	for k, v := range flat {
		c.setChainConfig("config."+k, v)
	}
}

func (c *DplatformOSConfig) setChainConfig(key string, value interface{}) {
	c.chainConfig[key] = value
}

func (c *DplatformOSConfig) getChainConfig(key string) (value interface{}, err error) {
	if data, ok := c.chainConfig[key]; ok {
		return data, nil
	}
	//
	tlog.Error("chain config " + key + " not found")
	return nil, ErrNotFound
}

// Init
func (c *DplatformOSConfig) dplatformosCfgInit(cfg *Config) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.forks == nil {
		c.forks = &Forks{}
	}
	c.forks.SetTestNetFork()

	if cfg != nil {
		if c.isLocal() {
			c.setTestNet(true)
		} else {
			c.setTestNet(cfg.TestNet)
		}

		if cfg.Wallet.MinFee < cfg.Mempool.MinTxFeeRate {
			panic("config must meet: wallet.minFee >= mempool.minTxFeeRate")
		}
		if cfg.Mempool.MaxTxFeeRate == 0 {
			cfg.Mempool.MaxTxFeeRate = 1e7 //0.1 coins
		}
		if cfg.Mempool.MaxTxFee == 0 {
			cfg.Mempool.MaxTxFee = 1e9 // 10 coins
		}
		c.setTxFeeConfig(cfg.Mempool.MinTxFeeRate, cfg.Mempool.MaxTxFeeRate, cfg.Mempool.MaxTxFee)
		if cfg.Consensus != nil {
			c.setMinerExecs(cfg.Consensus.MinerExecs)
		}
		c.setChainConfig("FixTime", cfg.FixTime)
		if cfg.CoinSymbol != "" {
			if strings.Contains(cfg.CoinSymbol, "-") {
				panic("config CoinSymbol must without '-'")
			}
			c.coinSymbol = cfg.CoinSymbol
		} else {
			if c.isPara() {
				panic("must config CoinSymbol in para chain")
			} else {
				c.coinSymbol = DefaultCoinsSymbol
			}
		}
		//TxHeight
		c.setChainConfig("TxHeight", cfg.TxHeight)
	}
	if c.needSetForkZero() { //local
		if c.isLocal() {
			c.forks.setLocalFork()
			c.setChainConfig("Debug", true)
		} else {
			c.forks.setForkForParaZero()
		}
	} else {
		if cfg != nil && cfg.Fork != nil {
			c.initForkConfig(cfg.Fork)
		}
	}
	//   fork
	if c.mver != nil {
		c.mver.UpdateFork(c.forks)
	}
}

func (c *DplatformOSConfig) needSetForkZero() bool {
	if c.isLocal() {
		return true
	} else if c.isPara() &&
		(c.mcfg == nil || c.mcfg.Fork == nil || c.mcfg.Fork.System == nil) &&
		!c.mcfg.EnableParaFork {
		//  para     fork，       fork   0（       ）
		return true
	}
	return false
}

func (c *DplatformOSConfig) setTestNet(isTestNet bool) {
	if !isTestNet {
		c.setChainConfig("TestNet", false)
		return
	}
	c.setChainConfig("TestNet", true)
	//const    TestNet
}

// GetP   ChainParam
func (c *DplatformOSConfig) GetP(height int64) *ChainParam {
	conf := Conf(c, "mver.consensus")
	chain := &ChainParam{}
	chain.MaxTxNumber = conf.MGInt("maxTxNumber", height)
	chain.PowLimitBits = uint32(conf.MGInt("powLimitBits", height))
	return chain
}

// GetMinerExecs
func (c *DplatformOSConfig) GetMinerExecs() []string {
	return c.minerExecs
}

func (c *DplatformOSConfig) setMinerExecs(execs []string) {
	if len(execs) > 0 {
		c.minerExecs = execs
	}
}

// GetFundAddr
func (c *DplatformOSConfig) GetFundAddr() string {
	return c.MGStr("mver.consensus.fundKeyAddr", 0)
}

// G   ChainConfig
func (c *DplatformOSConfig) G(key string) (value interface{}, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value, err = c.getChainConfig(key)
	return
}

// MG   mver config
func (c *DplatformOSConfig) MG(key string, height int64) (value interface{}, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.mver == nil {
		panic("mver is nil")
	}
	return c.mver.Get(key, height)
}

// GStr   ChainConfig
func (c *DplatformOSConfig) GStr(name string) string {
	value, err := c.G(name)
	if err != nil {
		return ""
	}
	if i, ok := value.(string); ok {
		return i
	}
	return ""
}

// MGStr   mver config
func (c *DplatformOSConfig) MGStr(name string, height int64) string {
	value, err := c.MG(name, height)
	if err != nil {
		return ""
	}
	if i, ok := value.(string); ok {
		return i
	}
	return ""
}

func parseInt(value interface{}) int64 {
	if i, ok := value.(int64); ok {
		return i
	}
	if s, ok := value.(string); ok {
		if strings.HasPrefix(s, "0x") {
			i, err := strconv.ParseUint(s, 0, 64)
			if err == nil {
				return int64(i)
			}
		}
	}
	return 0
}

// GInt   ChainConfig
func (c *DplatformOSConfig) GInt(name string) int64 {
	value, err := c.G(name)
	if err != nil {
		return 0
	}
	return parseInt(value)
}

// MGInt   mver config
func (c *DplatformOSConfig) MGInt(name string, height int64) int64 {
	value, err := c.MG(name, height)
	if err != nil {
		return 0
	}
	return parseInt(value)
}

// IsEnable   ChainConfig
func (c *DplatformOSConfig) IsEnable(name string) bool {
	isenable, err := c.G(name)
	if err == nil && isenable.(bool) {
		return true
	}
	return false
}

// MIsEnable   mver config
func (c *DplatformOSConfig) MIsEnable(name string, height int64) bool {
	isenable, err := c.MG(name, height)
	if err == nil && isenable.(bool) {
		return true
	}
	return false
}

// HasConf   chainConfig
func (c *DplatformOSConfig) HasConf(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.chainConfig[key]
	return ok
}

// S   chainConfig
func (c *DplatformOSConfig) S(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if strings.HasPrefix(key, "config.") {
		if !c.isLocal() && !c.isTestPara() { //only local and test para can modify for test
			panic("prefix config. is readonly")
		} else {
			tlog.Error("modify " + key + " is only for test")
		}
	}
	c.setChainConfig(key, value)
}

//SetTitleOnlyForTest set title only for test use
func (c *DplatformOSConfig) SetTitleOnlyForTest(ti string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.title = ti

}

// GetTitle   title
func (c *DplatformOSConfig) GetTitle() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.title
}

// GetCoinSymbol    coin symbol
func (c *DplatformOSConfig) GetCoinSymbol() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.coinSymbol
}

func (c *DplatformOSConfig) isLocal() bool {
	return c.title == "local"
}

// IsLocal   locak title
func (c *DplatformOSConfig) IsLocal() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isLocal()
}

// GetMinTxFeeRate get min transaction fee rate
func (c *DplatformOSConfig) GetMinTxFeeRate() int64 {
	return c.GInt("MinTxFeeRate")
}

// GetMaxTxFeeRate get max transaction fee rate
func (c *DplatformOSConfig) GetMaxTxFeeRate() int64 {
	return c.GInt("MaxTxFeeRate")
}

// GetMaxTxFee get max transaction fee
func (c *DplatformOSConfig) GetMaxTxFee() int64 {
	return c.GInt("MaxTxFee")
}

// SetTxFeeConfig
func (c *DplatformOSConfig) SetTxFeeConfig(minTxFeeRate, maxTxFeeRate, maxTxFee int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setTxFeeConfig(minTxFeeRate, maxTxFeeRate, maxTxFee)
}

func (c *DplatformOSConfig) setTxFeeConfig(minTxFeeRate, maxTxFeeRate, maxTxFee int64) {
	if minTxFeeRate < 0 {
		panic("minTxFeeRate less than zero")
	}

	if minTxFeeRate > maxTxFeeRate || maxTxFeeRate > maxTxFee {
		panic("SetTxFee, tx fee must meet, minTxFeeRate <= maxTxFeeRate <= maxTxFee")
	}
	c.setChainConfig("MinTxFeeRate", minTxFeeRate)
	c.setChainConfig("MaxTxFeeRate", maxTxFeeRate)
	c.setChainConfig("MaxTxFee", maxTxFee)
	c.setChainConfig("MinBalanceTransfer", minTxFeeRate*10)
}

// SetMinFee
func (c *DplatformOSConfig) SetMinFee(fee int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setTxFeeConfig(fee, fee*100, fee*10000)
}

func (c *DplatformOSConfig) isPara() bool {
	return strings.Count(c.title, ".") == 3 && strings.HasPrefix(c.title, ParaKeyX)
}

func (c *DplatformOSConfig) isTestPara() bool {
	return strings.Count(c.title, ".") == 3 && strings.HasPrefix(c.title, ParaKeyX) && strings.HasSuffix(c.title, "test.")
}

// IsPara
func (c *DplatformOSConfig) IsPara() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isPara()
}

// IsParaExecName
func IsParaExecName(exec string) bool {
	return strings.HasPrefix(exec, ParaKeyX)
}

//IsMyParaExecName      para
func (c *DplatformOSConfig) IsMyParaExecName(exec string) bool {
	return IsParaExecName(exec) && strings.HasPrefix(exec, c.GetTitle())
}

//IsSpecificParaExecName
func IsSpecificParaExecName(title, exec string) bool {
	return IsParaExecName(exec) && strings.HasPrefix(exec, title)
}

//GetParaExecTitleName          ，    title
func GetParaExecTitleName(exec string) (string, bool) {
	if IsParaExecName(exec) {
		for i := len(ParaKey); i < len(exec); i++ {
			if exec[i] == '.' {
				return exec[:i+1], true
			}
		}
	}
	return "", false
}

// IsTestNet
func (c *DplatformOSConfig) IsTestNet() bool {
	return c.IsEnable("TestNet")
}

// GetParaName      name
func (c *DplatformOSConfig) GetParaName() string {
	if c.IsPara() {
		return c.GetTitle()
	}
	return ""
}

// FlagKV   kv
func FlagKV(key []byte, value int64) *KeyValue {
	return &KeyValue{Key: key, Value: Encode(&Int64{Data: value})}
}

// MergeConfig Merge
func MergeConfig(conf map[string]interface{}, def map[string]interface{}) string {
	errstr := checkConfig("", conf, def)
	if errstr != "" {
		return errstr
	}
	mergeConfig(conf, def)
	return ""
}

//
func checkConfig(key string, conf map[string]interface{}, def map[string]interface{}) string {
	errstr := ""
	for key1, value1 := range conf {
		if vdef, ok := def[key1]; ok {
			conf1, ok1 := value1.(map[string]interface{})
			def1, ok2 := vdef.(map[string]interface{})
			if ok1 && ok2 {
				errstr += checkConfig(getkey(key, key1), conf1, def1)
			} else {
				errstr += "rewrite defalut key " + getkey(key, key1) + "\n"
			}
		}
	}
	return errstr
}

func mergeConfig(conf map[string]interface{}, def map[string]interface{}) {
	for key1, value1 := range def {
		if vdef, ok := conf[key1]; ok {
			conf1, ok1 := value1.(map[string]interface{})
			def1, ok2 := vdef.(map[string]interface{})
			if ok1 && ok2 {
				mergeConfig(conf1, def1)
				conf[key1] = conf1
			}
		} else {
			conf[key1] = value1
		}
	}
}

func getkey(key, key1 string) string {
	if key == "" {
		return key1
	}
	return key + "." + key1
}

//MergeCfg ...
func MergeCfg(cfgstring, cfgdefault string) string {
	if cfgdefault != "" {
		return mergeCfgString(cfgstring, cfgdefault)
	}
	return cfgstring
}

func mergeCfgString(cfgstring, cfgdefault string) string {
	//1. defconfig
	def := make(map[string]interface{})
	_, err := tml.Decode(cfgdefault, &def)
	if err != nil {
		panic(err)
	}
	//2. userconfig
	conf := make(map[string]interface{})
	_, err = tml.Decode(cfgstring, &conf)
	if err != nil {
		panic(err)
	}
	errstr := MergeConfig(conf, def)
	if errstr != "" {
		panic(errstr)
	}
	buf := new(bytes.Buffer)
	tml.NewEncoder(buf).Encode(conf)
	return buf.String()
}

func initCfgString(cfgstring string) (*Config, error) {
	var cfg Config
	if _, err := tml.Decode(cfgstring, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// InitCfg
func InitCfg(path string) (*Config, *ConfigSubModule) {
	return InitCfgString(readFile(path))
}

func flatConfig(key string, conf map[string]interface{}, flat map[string]interface{}) {
	for key1, value1 := range conf {
		conf1, ok := value1.(map[string]interface{})
		if ok {
			flatConfig(getkey(key, key1), conf1, flat)
		} else {
			flat[getkey(key, key1)] = value1
		}
	}
}

// FlatConfig Flat
func FlatConfig(conf map[string]interface{}) map[string]interface{} {
	flat := make(map[string]interface{})
	flatConfig("", conf, flat)
	return flat
}

func (c *DplatformOSConfig) setMver(cfgstring string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mver = newMversion(cfgstring)
}

// InitCfgString
func InitCfgString(cfgstring string) (*Config, *ConfigSubModule) {
	//cfgstring = c.mergeCfg(cfgstring) // TODO
	//setFlatConfig(cfgstring)         //  set
	cfg, err := initCfgString(cfgstring)
	if err != nil {
		panic(err)
	}
	//setMver(cfg.Title, cfgstring)    //  set
	sub, err := initSubModuleString(cfgstring)
	if err != nil {
		panic(err)
	}
	return cfg, sub
}

// subModule
type subModule struct {
	Store     map[string]interface{}
	Exec      map[string]interface{}
	Consensus map[string]interface{}
	Wallet    map[string]interface{}
	Mempool   map[string]interface{}
	Metrics   map[string]interface{}
	P2P       map[string]interface{}
}

//ReadFile ...
func ReadFile(path string) string {
	return readFile(path)
}

func readFile(path string) string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func initSubModuleString(cfgstring string) (*ConfigSubModule, error) {
	var cfg subModule
	if _, err := tml.Decode(cfgstring, &cfg); err != nil {
		return nil, err
	}
	return parseSubModule(&cfg)
}

func parseSubModule(cfg *subModule) (*ConfigSubModule, error) {
	var subcfg ConfigSubModule
	subcfg.Store = parseItem(cfg.Store)
	subcfg.Exec = parseItem(cfg.Exec)
	subcfg.Consensus = parseItem(cfg.Consensus)
	subcfg.Wallet = parseItem(cfg.Wallet)
	subcfg.Mempool = parseItem(cfg.Mempool)
	subcfg.Metrics = parseItem(cfg.Metrics)
	subcfg.P2P = parseItem(cfg.P2P)
	return &subcfg, nil
}

//ModifySubConfig json data modify
func ModifySubConfig(sub []byte, key string, value interface{}) ([]byte, error) {
	var data map[string]interface{}
	err := json.Unmarshal(sub, &data)
	if err != nil {
		return nil, err
	}
	data[key] = value
	return json.Marshal(data)
}

func parseItem(data map[string]interface{}) map[string][]byte {
	subconfig := make(map[string][]byte)
	if len(data) == 0 {
		return subconfig
	}
	for key := range data {
		if key == "sub" {
			subcfg := data[key].(map[string]interface{})
			for k := range subcfg {
				subconfig[k], _ = json.Marshal(subcfg[k])
			}
		}
	}
	return subconfig
}

// ConfQuery
type ConfQuery struct {
	cfg    *DplatformOSConfig
	prefix string
}

// Conf
func Conf(cfg *DplatformOSConfig, prefix string) *ConfQuery {
	if prefix == "" || (!strings.HasPrefix(prefix, "config.") && !strings.HasPrefix(prefix, "mver.")) {
		panic("ConfQuery must init buy prefix config. or mver.")
	}
	return &ConfQuery{cfg: cfg, prefix: prefix}
}

// ConfSub
func ConfSub(cfg *DplatformOSConfig, name string) *ConfQuery {
	return Conf(cfg, "config.exec.sub."+name)
}

// G     key
func (query *ConfQuery) G(key string) (interface{}, error) {
	return query.cfg.G(getkey(query.prefix, key))
}

func parseStrList(data interface{}) []string {
	var list []string
	if item, ok := data.([]interface{}); ok {
		for i := 0; i < len(item); i++ {
			one, ok := item[i].(string)
			if ok {
				list = append(list, one)
			}
		}
	}
	return list
}

// GStrList
func (query *ConfQuery) GStrList(key string) []string {
	data, err := query.G(key)
	if err == nil {
		return parseStrList(data)
	}
	return []string{}
}

// GInt   int
func (query *ConfQuery) GInt(key string) int64 {
	return query.cfg.GInt(getkey(query.prefix, key))
}

// GStr   string
func (query *ConfQuery) GStr(key string) string {
	return query.cfg.GStr(getkey(query.prefix, key))
}

// IsEnable   bool
func (query *ConfQuery) IsEnable(key string) bool {
	return query.cfg.IsEnable(getkey(query.prefix, key))
}

// MG   mversion
func (query *ConfQuery) MG(key string, height int64) (interface{}, error) {
	return query.cfg.MG(getkey(query.prefix, key), height)
}

// MGInt   mversion int
func (query *ConfQuery) MGInt(key string, height int64) int64 {
	return query.cfg.MGInt(getkey(query.prefix, key), height)
}

// MGStr   mversion string
func (query *ConfQuery) MGStr(key string, height int64) string {
	return query.cfg.MGStr(getkey(query.prefix, key), height)
}

// MGStrList   mversion string list
func (query *ConfQuery) MGStrList(key string, height int64) []string {
	data, err := query.MG(key, height)
	if err == nil {
		return parseStrList(data)
	}
	return []string{}
}

// MIsEnable   mversion bool
func (query *ConfQuery) MIsEnable(key string, height int64) bool {
	return query.cfg.MIsEnable(getkey(query.prefix, key), height)
}

//SetCliSysParam ...
func SetCliSysParam(title string, cfg *DplatformOSConfig) {
	if cfg == nil {
		panic("set cli system DplatformOSConfig param is nil")
	}
	cliSysParam[title] = cfg
}

//GetCliSysParam ...
func GetCliSysParam(title string) *DplatformOSConfig {
	if v, ok := cliSysParam[title]; ok {
		return v
	}
	panic(fmt.Sprintln("can not find CliSysParam title", title))
}

//AssertConfig ...
func AssertConfig(check interface{}) {
	if check == nil {
		panic("check object is nil (DplatformOSConfig)")
	}
}
