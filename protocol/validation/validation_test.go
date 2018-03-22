package validation

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/chain/txvm/crypto/ed25519"
	"github.com/chain/txvm/errors"
	"github.com/chain/txvm/protocol/bc"
)

func TestBlock(t *testing.T) {
	cases := []struct {
		f       func(*bc.Block, *bc.Block) (*bc.Block, *bc.BlockHeader)
		wantErr bool
	}{{
		f: func(b1, b2 *bc.Block) (*bc.Block, *bc.BlockHeader) {
			return b1, nil
		},
		wantErr: false,
	}, {
		f: func(b1, b2 *bc.Block) (*bc.Block, *bc.BlockHeader) {
			transactionsRoot := bc.NewHash([32]byte{1})
			b1.TransactionsRoot = &transactionsRoot // make b1 be invalid
			return b1, nil
		},
		wantErr: true,
	}, {
		f: func(b1, b2 *bc.Block) (*bc.Block, *bc.BlockHeader) {
			return b2, b1.BlockHeader
		},
		wantErr: false,
	}, {
		f: func(b1, b2 *bc.Block) (*bc.Block, *bc.BlockHeader) {
			transactionsRoot := bc.NewHash([32]byte{1})
			b2.TransactionsRoot = &transactionsRoot // make b2 be invalid
			return b2, b1.BlockHeader
		},
		wantErr: true,
	}, {
		f: func(b1, b2 *bc.Block) (*bc.Block, *bc.BlockHeader) {
			b2.Version = 2
			return b2, b1.BlockHeader
		},
		wantErr: true,
	}, {
		f: func(b1, b2 *bc.Block) (*bc.Block, *bc.BlockHeader) {
			return b2, nil
		},
		wantErr: true,
	}, {
		f: func(b1, b2 *bc.Block) (*bc.Block, *bc.BlockHeader) {
			b2.ExtraFields = append(b2.ExtraFields, &bc.DataItem{Type: bc.DataType_INT})
			return b2, b1.BlockHeader
		},
		wantErr: true,
	}}

	for _, c := range cases {
		b1 := newInitialBlock(t)
		b2 := generate(t, b1)
		b, bh := c.f(b1, b2)
		err := Block(b, bh)
		if (err == nil) && c.wantErr {
			t.Errorf("Block(%v, %v) = nil, want error", b, bh)
		} else if (err != nil) && !c.wantErr {
			t.Errorf("Block(%v, %v) = %v, want nil", b, bh, err)
		}
	}
}

func TestBlockSig(t *testing.T) {
	// A pubkey and a sig that will fail signature validation. Can't use
	// all-zeroes, which succeeds on about 25% of messages.
	badPubkey := mustDecodeHex("1111111111111111111111111111111111111111111111111111111111111111")
	badSig := mustDecodeHex("22222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222222")

	cases := []struct {
		pred    *bc.Predicate
		args    []interface{}
		wantErr error
	}{{
		pred:    &bc.Predicate{Version: 2}, // bad version
		wantErr: errBadPredicate,
	}, {
		pred:    &bc.Predicate{Version: 1, Quorum: -1}, // bad quorum
		wantErr: errBadPredicate,
	}, {
		pred:    &bc.Predicate{Version: 1, Quorum: 1}, // quorum > len(pubkeys)
		args:    []interface{}{make([]byte, ed25519.SignatureSize)},
		wantErr: errBadPredicate,
	}, {
		pred:    &bc.Predicate{Version: 1, Quorum: 1, Pubkeys: [][]byte{make([]byte, ed25519.PublicKeySize)}},
		wantErr: errBadArguments, // len(args) < len(pubkeys)
	}, {
		pred:    &bc.Predicate{Version: 1, Quorum: 1, Pubkeys: [][]byte{[]byte("badkeylen")}}, // bad public key size
		args:    []interface{}{make([]byte, ed25519.SignatureSize)},
		wantErr: errBadPredicate,
	}, {
		pred:    &bc.Predicate{Version: 1, Quorum: 1, Pubkeys: [][]byte{make([]byte, ed25519.PublicKeySize)}},
		args:    []interface{}{int64(1)}, // bad signature type
		wantErr: errBadArguments,
	}, {
		pred:    &bc.Predicate{Version: 1, Quorum: 0, Pubkeys: [][]byte{make([]byte, ed25519.PublicKeySize)}},
		args:    []interface{}{[]byte{}}, // empty sig is skipped
		wantErr: nil,
	}, {
		pred:    &bc.Predicate{Version: 1, Quorum: 1, Pubkeys: [][]byte{make([]byte, ed25519.PublicKeySize)}},
		args:    []interface{}{[]byte("badsiglen")}, // bad signature length
		wantErr: errBadArguments,
	}, {
		pred:    &bc.Predicate{Version: 1, Quorum: 1, Pubkeys: [][]byte{badPubkey}},
		args:    []interface{}{badSig}, // failing ed25519 check
		wantErr: errBadArguments,
	}, {
		pred:    &bc.Predicate{Version: 1, Quorum: 1, Pubkeys: [][]byte{make([]byte, ed25519.PublicKeySize)}},
		args:    []interface{}{[]byte{}},
		wantErr: errBadArguments, // insufficient signatures
	}, {
		pred:    &bc.Predicate{Version: 1, Quorum: 1, Pubkeys: [][]byte{mustDecodeHex("af8c8a878c85b47903bd90e7cdbb239847b7712748064fb16ecbf480c89d3606")}},
		args:    []interface{}{mustDecodeHex("59317b3f59a91613d801fbfada0abaf635900af8c84d66f7ad9932baa2a14bd5cad4649b993fced3d162a66ad4c688d8d5f4c943efa3da25383d168210c8980e")},
		wantErr: nil,
	}}

	block := &bc.Block{
		BlockHeader: &bc.BlockHeader{
			NextPredicate: &bc.Predicate{
				Version: 1,
			},
		},
	}

	for i, c := range cases {
		block.Arguments = c.args
		gotErr := BlockSig(block, c.pred)
		if errors.Root(gotErr) != c.wantErr {
			t.Errorf("BlockSig(%d) = %v want %v", i, gotErr, c.wantErr)
		}
	}
}

func TestBlockOnly(t *testing.T) {
	cases := []struct {
		tx      *bc.Tx
		wantErr error
	}{{
		tx: &bc.Tx{
			ID:       bc.NewHash([32]byte{1}),
			Version:  3,
			Runlimit: 2000,
		},
		wantErr: nil,
	}, {
		tx: &bc.Tx{
			ID:       bc.NewHash([32]byte{1}),
			Version:  2,
			Runlimit: 2000,
		},
		wantErr: errTxVersion,
	}, {
		tx: &bc.Tx{
			ID:       bc.NewHash([32]byte{1}),
			Version:  3,
			Runlimit: 5001,
		},
		wantErr: errRunlimit,
	}, {
		tx: &bc.Tx{
			ID:       bc.NewHash([32]byte{2}),
			Version:  3,
			Runlimit: 2000,
		},
		wantErr: errMismatchedMerkleRoot,
	}}

	txRoot := bc.TxMerkleRoot([]*bc.Tx{cases[0].tx})
	block := &bc.Block{
		BlockHeader: &bc.BlockHeader{
			Version:          3,
			Runlimit:         5000,
			TransactionsRoot: &txRoot,
		},
	}

	for i, c := range cases {
		block.Transactions = []*bc.Tx{c.tx}
		gotErr := BlockOnly(block)
		if errors.Root(gotErr) != c.wantErr {
			t.Errorf("BlockOnly(%d) = %v want %v", i, gotErr, c.wantErr)
		}
	}
}

func TestBlockPrev(t *testing.T) {
	prev := &bc.BlockHeader{
		Version:       3,
		Height:        10,
		TimestampMs:   1000,
		RefsCount:     5,
		NextPredicate: &bc.Predicate{},
	}
	prevHash := prev.Hash()

	cases := []struct {
		current *bc.BlockHeader
		wantErr error
	}{{
		current: &bc.BlockHeader{
			Version:         3,
			Height:          11,
			TimestampMs:     2000,
			RefsCount:       6,
			PreviousBlockId: &prevHash,
		},
		wantErr: nil,
	}, {
		current: &bc.BlockHeader{
			Version:         2, // bad version
			Height:          11,
			TimestampMs:     2000,
			RefsCount:       6,
			PreviousBlockId: &prevHash,
		},
		wantErr: errVersionRegression,
	}, {
		current: &bc.BlockHeader{
			Version:         3,
			Height:          12,
			TimestampMs:     2000,
			RefsCount:       6,
			PreviousBlockId: &prevHash,
		},
		wantErr: errMisorderedBlockHeight,
	}, {
		current: &bc.BlockHeader{
			Version:         3,
			Height:          11,
			TimestampMs:     2000,
			RefsCount:       6,
			PreviousBlockId: &bc.Hash{},
		},
		wantErr: errMismatchedBlock,
	}, {
		current: &bc.BlockHeader{
			Version:         3,
			Height:          11,
			TimestampMs:     1000,
			RefsCount:       6,
			PreviousBlockId: &prevHash,
		},
		wantErr: errMisorderedBlockTime,
	}, {
		current: &bc.BlockHeader{
			Version:         3,
			Height:          11,
			TimestampMs:     500,
			RefsCount:       6,
			PreviousBlockId: &prevHash,
		},
		wantErr: errMisorderedBlockTime,
	}, {
		current: &bc.BlockHeader{
			Version:         3,
			Height:          11,
			TimestampMs:     2000,
			RefsCount:       7,
			PreviousBlockId: &prevHash,
		},
		wantErr: errRefsCount,
	}}

	for i, c := range cases {
		gotErr := BlockPrev(&bc.Block{BlockHeader: c.current}, prev)
		if errors.Root(gotErr) != c.wantErr {
			t.Errorf("BlockPrev(%d) = %v want %v", i, gotErr, c.wantErr)
		}
	}
}

func newInitialBlock(tb testing.TB) *bc.Block {
	root := bc.TxMerkleRoot(nil) // calculate the zero value of the tx merkle root

	return &bc.Block{
		BlockHeader: &bc.BlockHeader{
			Version:          3,
			Height:           1,
			TimestampMs:      bc.Millis(time.Now()),
			TransactionsRoot: &root,
			NextPredicate:    &bc.Predicate{Version: 1, Quorum: 0},
		},
	}
}

func generate(tb testing.TB, prev *bc.Block) *bc.Block {
	prevID := prev.Hash()
	b := &bc.Block{
		BlockHeader: &bc.BlockHeader{
			Version:         3,
			Height:          prev.Height + 1,
			PreviousBlockId: &prevID,
			TimestampMs:     prev.TimestampMs + 1,
			NextPredicate:   prev.NextPredicate,
		},
	}

	txRoot := bc.TxMerkleRoot(nil)
	b.TransactionsRoot = &txRoot

	return b
}

func mustDecodeHex(str string) []byte {
	decoded, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return decoded
}
