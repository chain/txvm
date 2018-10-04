package ed25519

import (
	"bytes"
	"encoding/json"

	chainjson "github.com/chain/txvm/encoding/json"
)

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (pub *PublicKey) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		return nil
	}
	return json.Unmarshal(b, (*chainjson.HexBytes)(pub))
}

// MarshalJSON satisfies the json.Marshaler interface.
func (pub PublicKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(chainjson.HexBytes(pub))
}
