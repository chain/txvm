package json

import (
	"encoding/hex"
	"errors"
)

// ErrNotMap is returned when Map.UnmarshalJSON is called
// with bytes not representing null or a json object.
var ErrNotMap = errors.New("cannot unmarshal into Map, not a json object")

// HexBytes is a byte slice with hex encoding and decoding when
// marshaled and unmarshaled.
type HexBytes []byte

// MarshalText satisfies the encoding.TextMarshaler interface.
func (h HexBytes) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h)), nil
}

// UnmarshalText satisfies the encoding.TextUnmarshaler interface
func (h *HexBytes) UnmarshalText(text []byte) error {
	n := hex.DecodedLen(len(text))
	*h = make([]byte, n)
	_, err := hex.Decode(*h, text)
	return err
}

// Map is a byte slice that should parse as a string encoding a JSON
// object. This is checked in UnmarshalJSON.
type Map []byte

// MarshalJSON satisfies the json.Marshaler interface.
func (m Map) MarshalJSON() ([]byte, error) {
	if len(m) == 0 {
		return []byte("{}"), nil
	}
	return m, nil
}

// UnmarshalJSON satsifies the json.Unmarshaler interface.
func (m *Map) UnmarshalJSON(text []byte) error {
	// UnmarshalJSON takes only valid json, we can take advantage of this
	// to see if the first character is either '{' for an object or 'n'
	// for null.
	var first byte
	for _, c := range text {
		if !isSpace(c) {
			first = c
			break
		}
	}
	if first == 'n' {
		return nil
	} else if first != '{' {
		return ErrNotMap
	}

	*m = text
	return nil
}

// taken from std encoding/json
func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}
