package txvm

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestVMHash(t *testing.T) {
	cases := []struct {
		fn      string
		inp     []byte
		wantHex string
	}{
		{"f", []byte("x"), "17d00cf13f5cb7024201fadb919b1778804923fc01818cf2f1b904f7bf563d1f"},
		{"function_name", []byte("x_input"), "e7830b396576c7c0c33aab33ca084756f1c3a0cf0d413298a716d0378eb6f114"},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			got := VMHash(c.fn, c.inp)
			want, _ := hex.DecodeString(c.wantHex)
			if !bytes.Equal(got[:], want) {
				t.Errorf("got %x, want %x", got[:], want)
			}
		})
	}
}
