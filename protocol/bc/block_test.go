package bc

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/chain/txvm/crypto/ed25519"
	"github.com/chain/txvm/errors"
	"github.com/chain/txvm/protocol/txvm/asm"
	"github.com/chain/txvm/protocol/txvm/txvmtest"
	"github.com/chain/txvm/testutil"
)

var testBlockBytes = mustDecodeHex(
	"0a8101080310011a0909000000000000000120e80728d0860330013a0909000000000000000242090900000000000000034a" +
		"090900000000000000045246080110011a400000000000000000000000000000000000000000000000000000000000000000" +
		"000000000000000000000000000000000000000000000000000000000000000012f801080310a08d061aef0160436b636f6e" +
		"747261637473656564692e663e012a2d003b404460537f4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7" +
		"010b588d41025460560a7fd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d65616e63686f72" +
		"0454055446432d2d0032012a2e7f11111111111111111111111111111111111111111111111111111111111111112e6d2d2d" +
		"692e663e012a2d003b40444748433f9f01b6c0a6c50580fce2fac7d432bd9403fefef880df52a0e4407d2240b00c53cdb4a2" +
		"01c25c12faedd0bcdff9f2fa0598eb577bdb7808e75f99c7c0526cf995c7052e431a031201611a04080118051a0708022203" +
		"120162",
)

func testBlock() *Block {
	prog, err := asm.Assemble(txvmtest.SimplePayment)
	if err != nil {
		panic(err)
	}
	tx, err := NewTx(prog, 3, 100000)
	if err != nil {
		panic(err)
	}

	return &Block{
		UnsignedBlock: &UnsignedBlock{
			BlockHeader: &BlockHeader{
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
			},
			Transactions: []*Tx{tx},
		},
		Arguments: []interface{}{[]byte("a"), int64(5), []*DataItem{{Type: DataType_BYTES, Bytes: []byte("b")}}},
	}
}

func TestBlockBytes(t *testing.T) {
	block := testBlock()

	gotBytes, err := block.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(gotBytes, testBlockBytes) {
		t.Errorf("Bytes(%v):\ngot:  %x\n\twant: %x", block, gotBytes, testBlockBytes)
	}

	gotBlock := new(Block)
	err = gotBlock.FromBytes(testBlockBytes)
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(gotBlock, block) {
		t.Errorf("FromBytes(%x):\ngot:  %v\n\twant: %v", testBlockBytes, gotBlock, block)
	}

	badBlock := []byte("badblock")
	err = gotBlock.FromBytes(badBlock)
	if err == nil {
		t.Error("expected error for bad block bytes")
	}

	badTx := &Tx{RawTx: RawTx{Version: 3, Runlimit: 10000, Program: []byte("badprog")}}
	block.Transactions = append(block.Transactions, badTx)
	badTxBlock, err := block.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	err = gotBlock.FromBytes(badTxBlock)
	if err == nil {
		t.Error("expected error for bad tx bytes")
	}
}

func TestBlockMarshal(t *testing.T) {
	block := testBlock()

	blockBytes, err := block.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	wantBytes := []byte(hex.EncodeToString(blockBytes))
	gotBytes, err := block.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(gotBytes, wantBytes) {
		t.Errorf("MarshalText(%v):\ngot:  %x\n\twant: %x", block, gotBytes, wantBytes)
	}

	gotBlock := new(Block)
	err = gotBlock.UnmarshalText(wantBytes)
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(gotBlock, block) {
		t.Errorf("UnmarshalText(%x):\ngot:  %v\n\twant: %v", wantBytes, gotBlock, block)
	}
}

func TestBlockScan(t *testing.T) {
	block := testBlock()

	wantBytes, err := block.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	gotVal, err := block.Value()
	if err != nil {
		t.Fatal(err)
	}
	gotBytes, ok := gotVal.([]byte)
	if !ok {
		t.Fatal("expected bytes from Value")
	}

	if !bytes.Equal(wantBytes, gotBytes) {
		t.Errorf("Value(%v):\ngot:  %x\n\twant: %x", block, gotBytes, wantBytes)
	}

	gotBlock := new(Block)
	err = gotBlock.Scan(wantBytes)
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(gotBlock, block) {
		t.Errorf("Scan(%x):\ngot:  %v\n\twant: %v", wantBytes, gotBlock, block)
	}
}

func TestSignBlock(t *testing.T) {
	var (
		prv1 = mustDecodePrvHex("bd0fdc7670a69fda496b64277c173e17aaaedf1931f87fb11ee55c15020e8d48189d132ea856159387ea2a1beaa1749f3c60712ddff42e33996c571e276e5c12")
		prv2 = mustDecodePrvHex("67d8117e6ae63592d329b865a5c1c714f8eec4c50c1cafb4e0e6c073513ff41a2bbc4f1d61f1764a6345495e728a268d2b17bd99cc6637f4bc9ebc8b606b178a")
		prv3 = mustDecodePrvHex("ad08752268a613f2f96303fff27d515eba4c55d002f1804c8e743ed14b2b7335d33e67c2eed3f571935345b17d6a01de3175b9afd5b140d5a7a05270aa38cd3c")

		// This is an initial block with a 2-of-3 predicate using the keys
		// above. The ID of the initial block is
		// c03154168e3e08aff37359bb20c1a70ac4bdf885b349e2394a8a6c87a77bf4a2.
		initBlock = mustDecodeBlockHex("0aa1010803100120f5bea289b62c3a240966d71ebff8c6ffa71162d661a05647c15119fa493be44dff80f5214a43f8804b0ad88242004a00526a080110021a20189d132ea856159387ea2a1beaa1749f3c60712ddff42e33996c571e276e5c121a202bbc4f1d61f1764a6345495e728a268d2b17bd99cc6637f4bc9ebc8b606b178a1a20d33e67c2eed3f571935345b17d6a01de3175b9afd5b140d5a7a05270aa38cd3c")

		block2 = UnsignedBlock{
			BlockHeader: &BlockHeader{
				Version:          3,
				Height:           2,
				PreviousBlockId:  mustDecodeHashPtr("c03154168e3e08aff37359bb20c1a70ac4bdf885b349e2394a8a6c87a77bf4a2"),
				TimestampMs:      1526343442294, // 1+the timestamp in the initial block
				TransactionsRoot: new(Hash),
				ContractsRoot:    new(Hash),
				NoncesRoot:       new(Hash),
				NextPredicate:    new(Predicate),
			},
		}
		blockID = block2.Hash().Bytes()

		sig1 = ed25519.Sign(prv1, blockID)
		sig2 = ed25519.Sign(prv2, blockID)
		sig3 = ed25519.Sign(prv3, blockID)
	)

	errTooFewKeys := errors.New("too few keys")
	cases := []struct {
		name     string
		keys     []ed25519.PrivateKey
		wantsigs []interface{}
		wanterr  error
	}{
		{
			name:     "1and2",
			keys:     []ed25519.PrivateKey{prv1, prv2, nil},
			wantsigs: []interface{}{sig1, sig2, nil},
		},
		{
			name:     "1and3",
			keys:     []ed25519.PrivateKey{prv1, nil, prv3},
			wantsigs: []interface{}{sig1, nil, sig3},
		},
		{
			name:     "2and3",
			keys:     []ed25519.PrivateKey{nil, prv2, prv3},
			wantsigs: []interface{}{nil, sig2, sig3},
		},
		{
			name: "all",
			keys: []ed25519.PrivateKey{prv1, prv2, prv3},
		},
		{
			name:    "just1",
			keys:    []ed25519.PrivateKey{prv1, nil, nil},
			wanterr: ErrTooFewSignatures,
		},
		{
			name:    "just2",
			keys:    []ed25519.PrivateKey{nil, prv2, nil},
			wanterr: ErrTooFewSignatures,
		},
		{
			name:    "just3",
			keys:    []ed25519.PrivateKey{nil, nil, prv3},
			wanterr: ErrTooFewSignatures,
		},
		{
			name:    "none",
			keys:    []ed25519.PrivateKey{nil, nil, nil},
			wanterr: ErrTooFewSignatures,
		},
		{
			name:    "nokeys",
			wanterr: errTooFewKeys,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			b, err := SignBlock(&block2, initBlock.BlockHeader, func(idx int) (interface{}, error) {
				if idx >= len(tc.keys) {
					return nil, errTooFewKeys
				}
				if len(tc.keys[idx]) == 0 {
					return nil, nil
				}
				return ed25519.Sign(tc.keys[idx], blockID), nil
			})
			if tc.wanterr != nil {
				if err == nil {
					t.Errorf("got no error, want %v", tc.wanterr)
				} else if errors.Root(err) != tc.wanterr {
					t.Errorf("got %v, want %v", err, tc.wanterr)
				}
			} else if tc.name != "all" {
				if !testutil.DeepEqual(b.Arguments, tc.wantsigs) {
					t.Errorf("got %s, want %s", showSigs(b.Arguments), showSigs(tc.wantsigs))
				}
			}
		})
	}
}

func showSigs(sigs []interface{}) string {
	var strs []string
	for _, sig := range sigs {
		if sig == nil {
			strs = append(strs, "nil")
		} else {
			strs = append(strs, hex.EncodeToString(sig.([]byte)))
		}
	}
	return "[" + strings.Join(strs, " ") + "]"
}

func mustDecodePrvHex(h string) ed25519.PrivateKey {
	return ed25519.PrivateKey(mustDecodeHex(h))
}

func mustDecodeBlockHex(h string) *Block {
	var b Block
	err := b.UnmarshalText([]byte(h))
	if err != nil {
		panic(err)
	}
	return &b
}

func mustDecodeHashPtr(h string) *Hash {
	var hash Hash
	err := hash.UnmarshalText([]byte(h))
	if err != nil {
		panic(err)
	}
	return &hash
}
