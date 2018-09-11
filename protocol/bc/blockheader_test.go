package bc

import (
	"bytes"
	"testing"

	"i10r.io/testutil"
)

var filledBlock = &BlockHeader{
	Version:          3,
	Height:           1,
	PreviousBlockId:  hashPtr(NewHash([32]byte{1})),
	TimestampMs:      1000,
	RefsCount:        1,
	Runlimit:         50000,
	TransactionsRoot: hashPtr(NewHash([32]byte{2})),
	ContractsRoot:    hashPtr(NewHash([32]byte{3})),
	NoncesRoot:       hashPtr(NewHash([32]byte{4})),
	NextPredicate: &Predicate{
		Version: 1,
		Quorum:  1,
		Pubkeys: [][]byte{make([]byte, 64)},
	},
}

func TestBlockHeaderHash(t *testing.T) {
	cases := []struct {
		h    *BlockHeader
		want Hash
	}{{
		h: &BlockHeader{
			NextPredicate: &Predicate{
				Version: 1,
			},
		},
		want: mustDecodeHash("d1ae2bb8f50558a859a8578aa8c419257f0e2d2d19b89d8e07b716ceab7df443"),
	}, {
		h:    filledBlock,
		want: mustDecodeHash("5eb5b8cbd3e41b767a6447d5cb4adc2a255eb3fad37b1e2279b5ddb5f64c41af"),
	}}

	for _, c := range cases {
		got := c.h.Hash()
		if got != c.want {
			t.Errorf("Hash(%v) = %x want %x", c.h, got.Bytes(), c.want.Bytes())
		}
	}
}

func TestBlockHeaderScanValue(t *testing.T) {
	filledBytes := mustDecodeHex("080310011a0909000000000000000120e80728d0860330013a0909000000000000000242090900000000000000034a090900000000000000045246080110011a4000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	val, err := filledBlock.Value()
	if err != nil {
		t.Fatal(err)
	}
	vBytes, ok := val.([]byte)
	if !ok {
		t.Fatal("expected value to return byte slice")
	}
	if !bytes.Equal(vBytes, filledBytes) {
		t.Errorf("Value(emptyBlock):\n\tgot:  %x\n\twant: %x", vBytes, filledBytes)
	}

	scanned := new(BlockHeader)
	err = scanned.Scan(vBytes)
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(scanned, filledBlock) {
		t.Errorf("Scan(%x):\n\tgot:  %v\n\twant: %v", vBytes, scanned, filledBlock)
	}

	err = scanned.Scan(int64(5))
	if err == nil {
		t.Fatal("expected error for scanning bad Value type")
	}
}

func hashPtr(hash Hash) *Hash { return &hash }
