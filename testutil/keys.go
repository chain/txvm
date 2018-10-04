package testutil

import (
	"github.com/chain/txvm/crypto/ed25519"
	"github.com/chain/txvm/crypto/ed25519/chainkd"

	miscreant "github.com/miscreant/miscreant/go"
)

var (
	// TestXPub is an xpub. Its corresponding xprv is TestXPrv.
	TestXPub chainkd.XPub

	// TestXPrv is an xprv. Its corresponding xpub is TestXPub.
	TestXPrv chainkd.XPrv

	// TestPub is the pubkey extracted from TestXPub.
	TestPub ed25519.PublicKey

	// TestPubs is a list of pubkeys containing TestPub.
	TestPubs []ed25519.PublicKey

	// TestCipher is an AES-PMAC-SIV cipher.
	TestCipher *miscreant.Cipher
)

type zeroReader struct{}

func (z zeroReader) Read(buf []byte) (int, error) {
	for i := range buf {
		buf[i] = 0
	}
	return len(buf), nil
}

func init() {
	var err error
	TestXPrv, TestXPub, err = chainkd.NewXKeys(zeroReader{})
	if err != nil {
		panic(err)
	}
	TestPub = TestXPub.PublicKey()
	TestPubs = []ed25519.PublicKey{TestPub}
	TestCipher, err = miscreant.NewAESPMACSIV(miscreant.GenerateKey(32))
	if err != nil {
		panic(err)
	}
}
