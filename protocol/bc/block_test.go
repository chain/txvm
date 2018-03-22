package bc

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/chain/txvm/protocol/txvm/asm"
	"github.com/chain/txvm/protocol/txvm/txvmtest"
	"github.com/chain/txvm/testutil"
)

var (
	testBlock      Block
	testBlockBytes []byte
)

func init() {
	prog, err := asm.Assemble(txvmtest.SimplePayment)
	if err != nil {
		panic(err)
	}
	tx, err := NewTx(prog, 3, 100000)
	if err != nil {
		panic(err)
	}

	testBlock = Block{
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
		Arguments:    []interface{}{[]byte("a"), int64(5), []*DataItem{{Type: DataType_BYTES, Bytes: []byte("b")}}},
	}

	testBlockBytes = mustDecodeHex(
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
}

func TestBlockBytes(t *testing.T) {
	block := new(Block)
	*block = testBlock

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

	badTx := &Tx{Version: 3, Runlimit: 10000, WitnessProg: []byte("badprog")}
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
	block := new(Block)
	*block = testBlock

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
	block := new(Block)
	*block = testBlock

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
