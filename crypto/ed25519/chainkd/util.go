package chainkd

import (
	"io"

	"i10r.io/crypto/ed25519"
)

// Utility functions

// NewXKeys produces a new keypair from a source of random bytes.
func NewXKeys(r io.Reader) (xprv XPrv, xpub XPub, err error) {
	xprv, err = NewXPrv(r)
	if err != nil {
		return
	}
	return xprv, xprv.XPub(), nil
}

// XPubKeys extracts the ed25519.PublicKey from each XPub.
func XPubKeys(xpubs []XPub) []ed25519.PublicKey {
	res := make([]ed25519.PublicKey, 0, len(xpubs))
	for _, xpub := range xpubs {
		res = append(res, xpub.PublicKey())
	}
	return res
}

// DeriveXPubs calls Derive on each XPub, all with the same path.
func DeriveXPubs(xpubs []XPub, path [][]byte) []XPub {
	res := make([]XPub, 0, len(xpubs))
	for _, xpub := range xpubs {
		d := xpub.Derive(path)
		res = append(res, d)
	}
	return res
}
