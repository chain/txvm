package state

import (
	"reflect"
	"testing"

	"github.com/chain/txvm/protocol/bc"
)

func empty(t *testing.T) *Snapshot {
	s := Empty()
	b1 := &bc.UnsignedBlock{
		BlockHeader: &bc.BlockHeader{
			Version:       3,
			Height:        1,
			TimestampMs:   1,
			NextPredicate: &bc.Predicate{},
		},
	}
	err := s.ApplyBlock(b1)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestApplyTxSpend(t *testing.T) {
	snap := empty(t)
	spentOutputID := bc.NewHash([32]byte{1})
	snap.ContractsTree.Insert(spentOutputID.Bytes())

	tx := &bc.Tx{Contracts: []bc.Contract{{Type: bc.InputType, ID: spentOutputID}}}

	// Apply the spend transaction.
	err := snap.ApplyTx(bc.NewCommitmentsTx(tx))
	if err != nil {
		t.Fatal(err)
	}
	if snap.ContractsTree.Contains(spentOutputID.Bytes()) {
		t.Error("snapshot contains spent prevout")
	}
	err = snap.ApplyTx(bc.NewCommitmentsTx(tx))
	if err == nil {
		t.Error("expected error applying spend twice, got nil")
	}
}

func TestApplyIssuanceTwice(t *testing.T) {
	snap := empty(t)
	issuance := &bc.Tx{
		Nonces: []bc.Nonce{{ID: bc.NewHash([32]byte{2}), ExpMS: 5}},
	}
	err := snap.ApplyTx(bc.NewCommitmentsTx(issuance))
	if err != nil {
		t.Fatal(err)
	}
	err = snap.ApplyTx(bc.NewCommitmentsTx(issuance))
	if err == nil {
		t.Errorf("expected error for duplicate nonce, got %s", err)
	}
}

func TestCopySnapshot(t *testing.T) {
	snap := empty(t)
	tx := &bc.Tx{
		Contracts: []bc.Contract{{Type: bc.OutputType, ID: bc.NewHash([32]byte{1})}},
		Nonces:    []bc.Nonce{{ID: bc.NewHash([32]byte{2}), ExpMS: 5}},
	}
	snap.ApplyTx(bc.NewCommitmentsTx(tx))
	dupe := Copy(snap)
	if !reflect.DeepEqual(dupe, snap) {
		t.Errorf("got %#v, want %#v", dupe, snap)
	}
}

func TestApplyBlock(t *testing.T) {
	maxTime := uint64(10)
	// Setup a snapshot with a nonce with a known expiry.
	snap := empty(t)
	snap.NonceTree.Insert(bc.NonceCommitment(bc.Hash{}, maxTime))

	// Land a block later than the issuance's max time.
	block := &bc.UnsignedBlock{
		BlockHeader: &bc.BlockHeader{
			Height:        2,
			TimestampMs:   maxTime + 1,
			NextPredicate: &bc.Predicate{},
		},
	}
	err := snap.ApplyBlock(block)
	if err != nil {
		t.Fatal(err)
	}
	if snap.NonceTree.RootHash() != ([32]byte{}) {
		t.Error("got non-empty nonce tree")
	}

	snap = Empty()
	err = snap.ApplyBlock(block)
	if err == nil {
		t.Error("expected error for uninitialized state")
	}

	snap = empty(t)
	block = &bc.UnsignedBlock{
		BlockHeader: &bc.BlockHeader{
			Height:        1,
			NextPredicate: &bc.Predicate{},
		},
	}
	err = snap.ApplyBlock(block)
	if err == nil {
		t.Error("expected error for initialized state")
	}

	snap = empty(t)
	block = &bc.UnsignedBlock{
		BlockHeader: &bc.BlockHeader{
			Height:        2,
			NextPredicate: &bc.Predicate{},
		},
		Transactions: []*bc.Tx{{
			Contracts: []bc.Contract{{
				Type: bc.InputType,
				ID:   bc.NewHash([32]byte{1}),
			}},
		}},
	}
	err = snap.ApplyBlock(block)
	if err == nil {
		t.Error("expected error for transaction")
	}
}

func TestApplyTx(t *testing.T) {
	tx := &bc.Tx{}
	snap := Empty()

	err := snap.ApplyTx(bc.NewCommitmentsTx(tx))
	if err == nil {
		t.Error("expected uninitialized error")
	}

	snap = empty(t)
	err = snap.ApplyTx(bc.NewCommitmentsTx(tx))
	if err != nil {
		t.Fatal(err)
	}
}

func TestRefIDNonce(t *testing.T) {
	snap := empty(t)
	b1 := &bc.UnsignedBlock{
		BlockHeader: &bc.BlockHeader{
			Height:        2,
			NextPredicate: &bc.Predicate{},
		},
	}
	err := snap.ApplyBlock(b1)
	if err != nil {
		t.Fatal(err)
	}
	tx := &bc.Tx{
		Nonces: []bc.Nonce{{
			ID:      bc.NewHash([32]byte{1}),
			BlockID: b1.Hash(),
			ExpMS:   10000,
		}},
	}
	err = snap.ApplyTx(bc.NewCommitmentsTx(tx))
	if err != nil {
		t.Fatal(err)
	}

	tx = &bc.Tx{
		Nonces: []bc.Nonce{{
			ID:      bc.NewHash([32]byte{2}),
			BlockID: bc.NewHash([32]byte{255}),
			ExpMS:   10000,
		}},
	}
	err = snap.ApplyTx(bc.NewCommitmentsTx(tx))
	if err == nil {
		t.Error("expected error for applying tx with invalid block id")
	}
}

func TestAtomicApplyTx(t *testing.T) {
	snap := empty(t)
	missingSpend := &bc.Tx{
		Nonces: []bc.Nonce{{
			ID:      bc.NewHash([32]byte{1}),
			BlockID: bc.Hash{},
			ExpMS:   1000,
		}},
		Contracts: []bc.Contract{{
			Type: bc.OutputType,
			ID:   bc.NewHash([32]byte{2}),
		}, {
			Type: bc.InputType,
			ID:   bc.NewHash([32]byte{3}),
		}},
	}

	wantCSRoot := snap.ContractsTree.RootHash()
	wantNonceRoot := snap.NonceTree.RootHash()

	err := snap.ApplyTx(bc.NewCommitmentsTx(missingSpend))
	if err == nil {
		t.Fatal("expected err")
	}

	gotCSRoot := snap.ContractsTree.RootHash()
	gotNonceRoot := snap.NonceTree.RootHash()

	if wantCSRoot != gotCSRoot {
		t.Fatal("invalid tx affected contracts state tree")
	}

	if wantNonceRoot != gotNonceRoot {
		t.Fatal("invalid tx affected nonces state tree")
	}
}

func TestHeaderAccessors(t *testing.T) {
	cases := []struct {
		snap          *Snapshot
		wantHeight    uint64
		wantTimestamp uint64
	}{{
		snap: nil,
	}, {
		snap: Empty(),
	}, {
		snap: &Snapshot{
			Header: &bc.BlockHeader{
				Height:      5,
				TimestampMs: 1000,
			},
		},
		wantHeight:    5,
		wantTimestamp: 1000,
	}}

	for i, c := range cases {
		gotHeight := c.snap.Height()
		if gotHeight != c.wantHeight {
			t.Errorf("height(%d) = %d want %d", i, gotHeight, c.wantHeight)
		}

		gotTimestamp := c.snap.TimestampMS()
		if gotTimestamp != c.wantTimestamp {
			t.Errorf("timestamp(%d) = %d want %d", i, gotTimestamp, c.wantTimestamp)
		}
	}
}
