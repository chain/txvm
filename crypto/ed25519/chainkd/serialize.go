package chainkd

import (
	"encoding/hex"
	"errors"
)

const (
	extendedPublicKeySize  = 64
	extendedPrivateKeySize = 64
)

// ErrBadKeyStr is produced when an error is encountered parsing an
// XPub or XPrv.
var ErrBadKeyStr = errors.New("bad key string")

// MarshalText satisfies the encoding.TextMarshaler interface.
func (xpub XPub) MarshalText() ([]byte, error) {
	hexBytes := make([]byte, hex.EncodedLen(len(xpub.Bytes())))
	hex.Encode(hexBytes, xpub.Bytes())
	return hexBytes, nil
}

// Bytes produces xpub as a byte slice.
func (xpub XPub) Bytes() []byte {
	return xpub[:]
}

// MarshalText satisfies the encoding.TextMarshaler interface.
func (xprv XPrv) MarshalText() ([]byte, error) {
	hexBytes := make([]byte, hex.EncodedLen(len(xprv.Bytes())))
	hex.Encode(hexBytes, xprv.Bytes())
	return hexBytes, nil
}

// Bytes produces xprv as a byte slice.
func (xprv XPrv) Bytes() []byte {
	return xprv[:]
}

// UnmarshalText satisfies the encoding.TextUnmarshaler interface.
func (xpub *XPub) UnmarshalText(inp []byte) error {
	if len(inp) != 2*extendedPublicKeySize {
		return ErrBadKeyStr
	}
	_, err := hex.Decode(xpub[:], inp)
	return err
}

// String produces a hex-encoded copy of xpub.
func (xpub XPub) String() string {
	return hex.EncodeToString(xpub.Bytes())
}

// UnmarshalText satisfies the encoding.TextUnmarshaler interface.
func (xprv *XPrv) UnmarshalText(inp []byte) error {
	if len(inp) != 2*extendedPrivateKeySize {
		return ErrBadKeyStr
	}
	_, err := hex.Decode(xprv[:], inp)
	return err
}

func (xprv XPrv) String() string {
	return hex.EncodeToString(xprv.Bytes())
}
