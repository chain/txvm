package bc

import (
	"encoding/hex"
	"testing"
)

func TestTxMerkleRoot(t *testing.T) {
	cases := []struct {
		txs  []*Tx
		want Hash
	}{{
		txs: []*Tx{
			{ID: NewHash([32]byte{1})},
		},
		want: mustDecodeHash("aef1dc9c35132b522eb83c5862a92d376a6795b4b80217a33a18fceaeb396f6c"),
	}, {
		txs: []*Tx{
			{ID: NewHash([32]byte{1})},
			{ID: NewHash([32]byte{2})},
		},
		want: mustDecodeHash("630de80b9c14b06caaa23b916ea0f5b8840c9b5e7aaffa71be66646ea0e82530"),
	}, {
		txs: []*Tx{
			{ID: NewHash([32]byte{1})},
			{ID: NewHash([32]byte{2})},
			{ID: NewHash([32]byte{3})},
		},
		want: mustDecodeHash("591060c4860613ed97f6bed1c804b16b636108e99d9d535cc14a4698f09f564a"),
	}}

	for _, c := range cases {
		got := TxMerkleRoot(c.txs)
		if got != c.want {
			t.Log("txs", c.txs)
			t.Errorf("got merkle root = %x want %x", got.Bytes(), c.want.Bytes())
		}
	}
}

func mustDecodeHex(s string) []byte {
	b, err := hex.DecodeString(s)
	must(err)
	return b
}

func mustDecodeHash(s string) (h Hash) {
	err := h.UnmarshalText([]byte(s))
	must(err)
	return h
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
