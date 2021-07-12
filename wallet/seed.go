// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wallet

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/wallet/bipwallet"
)

var (
	// SeedLong
	SeedLong = 15
	// SaveSeedLong
	SaveSeedLong = 12

	// WalletSeed
	WalletSeed = []byte("walletseed")
	seedlog    = log.New("module", "wallet")

	// ChineseSeedCache
	ChineseSeedCache = make(map[string]string)
	// EnglishSeedCache
	EnglishSeedCache = make(map[string]string)
)

// BACKUPKEYINDEX     Key
const BACKUPKEYINDEX = "backupkeyindex"

// CreateSeed           seed  ，
//lang = 0
//lang = 1
//bitsize=128   12       ，bitsize+32=160    15       ，bitszie=256   24
func CreateSeed(folderpath string, lang int32) (string, error) {
	mnem, err := bipwallet.NewMnemonicString(int(lang), 160)
	if err != nil {
		seedlog.Error("CreateSeed", "NewMnemonicString err", err)
		return "", err
	}
	return mnem, nil
}

// InitSeedLibrary    seed       map ，  seed
func InitSeedLibrary() {
	//     seed
	englieshstrs := strings.Split(englishText, " ")
	chinesestrs := strings.Split(chineseText, " ")

	//     seed    map
	for _, wordstr := range chinesestrs {
		ChineseSeedCache[wordstr] = wordstr
	}

	for _, wordstr := range englieshstrs {
		EnglishSeedCache[wordstr] = wordstr
	}
}

// VerifySeed      seed        ，
func VerifySeed(seed string, signType int, coinType uint32) (bool, error) {

	_, err := bipwallet.NewWalletFromMnemonic(coinType, uint32(signType), seed)
	if err != nil {
		seedlog.Error("VerifySeed NewWalletFromMnemonic", "err", err)
		return false, err
	}
	return true, nil
}

// SaveSeedInBatch
func SaveSeedInBatch(db dbm.DB, seed string, password string, batch dbm.Batch) (bool, error) {
	if len(seed) == 0 || len(password) == 0 {
		return false, types.ErrInvalidParam
	}

	Encrypted, err := AesgcmEncrypter([]byte(password), []byte(seed))
	if err != nil {
		seedlog.Error("SaveSeed", "AesgcmEncrypter err", err)
		return false, err
	}
	batch.Set(WalletSeed, Encrypted)
	//seedlog.Info("SaveSeed ok", "Encryptedseed", Encryptedseed)
	return true, nil
}

//GetSeed   password  seed
func GetSeed(db dbm.DB, password string) (string, error) {
	if len(password) == 0 {
		return "", types.ErrInvalidParam
	}
	Encryptedseed, err := db.Get(WalletSeed)
	if err != nil {
		return "", err
	}
	if len(Encryptedseed) == 0 {
		return "", types.ErrSeedNotExist
	}
	seed, err := AesgcmDecrypter([]byte(password), Encryptedseed)
	if err != nil {
		seedlog.Error("GetSeed", "AesgcmDecrypter err", err)
		return "", types.ErrInputPassword
	}
	return string(seed), nil
}

//GetPrivkeyBySeed   seed
func GetPrivkeyBySeed(db dbm.DB, seed string, specificIndex uint32, SignType int, coinType uint32) (string, error) {
	var backupindex uint32
	var Hexsubprivkey string
	var err error
	var index uint32
	signType := uint32(SignType)
	//         child
	if specificIndex == 0 {
		backuppubkeyindex, err := db.Get([]byte(BACKUPKEYINDEX))
		if backuppubkeyindex == nil || err != nil {
			index = 0
		} else {
			if err = json.Unmarshal(backuppubkeyindex, &backupindex); err != nil {
				return "", err
			}
			index = backupindex + 1
		}
	} else {
		index = specificIndex
	}
	cryptoName := crypto.GetName(SignType)
	if cryptoName == "unknown" {
		return "", types.ErrNotSupport
	}

	wallet, err := bipwallet.NewWalletFromMnemonic(coinType, signType, seed)
	if err != nil {
		seedlog.Error("GetPrivkeyBySeed NewWalletFromMnemonic", "err", err)
		wallet, err = bipwallet.NewWalletFromSeed(coinType, signType, []byte(seed))
		if err != nil {
			seedlog.Error("GetPrivkeyBySeed NewWalletFromSeed", "err", err)
			return "", types.ErrNewWalletFromSeed
		}
	}

	//      Key pair
	priv, pub, err := wallet.NewKeyPair(index)
	if err != nil {
		seedlog.Error("GetPrivkeyBySeed NewKeyPair", "err", err)
		return "", types.ErrNewKeyPair
	}

	Hexsubprivkey = hex.EncodeToString(priv)

	public, err := bipwallet.PrivkeyToPub(coinType, signType, priv)
	if err != nil {
		seedlog.Error("GetPrivkeyBySeed PrivkeyToPub", "err", err)
		return "", types.ErrPrivkeyToPub
	}
	if !bytes.Equal(pub, public) {
		seedlog.Error("GetPrivkeyBySeed NewKeyPair pub  != PrivkeyToPub", "err", err)
		return "", types.ErrSubPubKeyVerifyFail
	}

	// back up index in db
	if specificIndex == 0 {
		var pubkeyindex []byte
		pubkeyindex, err = json.Marshal(index)
		if err != nil {
			seedlog.Error("GetPrivkeyBySeed", "Marshal err ", err)
			return "", types.ErrMarshal
		}

		err = db.SetSync([]byte(BACKUPKEYINDEX), pubkeyindex)
		if err != nil {
			seedlog.Error("GetPrivkeyBySeed", "SetSync err ", err)
			return "", err
		}
	}
	return Hexsubprivkey, nil
}

//AesgcmEncrypter      password seed  aesgcm  ,      seed
func AesgcmEncrypter(password []byte, seed []byte) ([]byte, error) {
	key := make([]byte, 32)
	if len(password) > 32 {
		key = password[0:32]
	} else {
		copy(key, password)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		seedlog.Error("AesgcmEncrypter NewCipher err", "err", err)
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		seedlog.Error("AesgcmEncrypter NewGCM err", "err", err)
		return nil, err
	}

	Encrypted := aesgcm.Seal(nil, key[:12], seed, nil)
	//seedlog.Info("AesgcmEncrypter Seal", "seed", seed, "key", key, "Encrypted", Encrypted)
	return Encrypted, nil
}

//AesgcmDecrypter      password seed  aesgcm  ,      seed
func AesgcmDecrypter(password []byte, seed []byte) ([]byte, error) {
	key := make([]byte, 32)
	if len(password) > 32 {
		key = password[0:32]
	} else {
		copy(key, password)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		seedlog.Error("AesgcmDecrypter", "NewCipher err", err)
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		seedlog.Error("AesgcmDecrypter", "NewGCM err", err)
		return nil, err
	}
	decryptered, err := aesgcm.Open(nil, key[:12], seed, nil)
	if err != nil {
		seedlog.Error("AesgcmDecrypter", "aesgcm Open err", err)
		return nil, err
	}
	//seedlog.Info("AesgcmDecrypter", "password", string(password), "seed", seed, "decryptered", string(decryptered))
	return decryptered, nil
}
