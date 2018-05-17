package protocol

import (
	"context"
	"testing"
	"time"

	"github.com/chain/txvm/protocol/bc"
	"github.com/chain/txvm/protocol/prottest/memstore"
	"github.com/chain/txvm/protocol/state"
	"github.com/chain/txvm/testutil"
)

func TestRecoverSnapshotNoAdditionalBlocks(t *testing.T) {
	ctx := context.Background()
	store := memstore.New()
	b1, err := NewInitialBlock(nil, 0, time.Now().Add(-time.Minute))
	if err != nil {
		testutil.FatalErr(t, err)
	}
	c1, err := NewChain(context.Background(), b1, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	c1.blocksPerSnapshot = 0
	st := state.Empty()
	err = st.ApplyBlock(b1.UnsignedBlock)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = c1.CommitAppliedBlock(ctx, b1, st)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	tx := &bc.Tx{ID: bc.NewHash([32]byte{byte(0)})}
	b2, _, err := c1.GenerateBlock(ctx, st, bc.Millis(time.Now()), []*bc.CommitmentsTx{bc.NewCommitmentsTx(tx)})
	if err != nil {
		t.Fatal(err)
	}
	sb2, err := bc.SignBlock(b2, st.Header, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = c1.CommitBlock(ctx, sb2)
	if err != nil {
		t.Fatal(err)
	}

	// Snapshots are applied asynchronously. This loops waits
	// until the snapshot is created.
	for {
		snap, _ := store.LatestSnapshot(context.Background())
		if snap.Height() > 0 {
			break
		}
	}

	c2, err := NewChain(context.Background(), b1, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	state, err := c2.Recover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state.Height() != 2 {
		t.Fatalf("state.Height = %d, want %d", state.Height(), 2)
	}

	curState := c2.State()
	if curState.Height() != 2 {
		t.Fatalf("chain.state.Height = %d, want %d", curState.Height(), 2)
	}
	if curState.Header == nil {
		t.Fatal("chain.state.Header is nil")
	}
}

func TestRecoverSnapshotAdditionalBlocks(t *testing.T) {
	store := memstore.New()
	b, err := NewInitialBlock(nil, 0, time.Now().Add(-time.Minute))
	if err != nil {
		testutil.FatalErr(t, err)
	}
	c1, err := NewChain(context.Background(), b, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	st := state.Empty()
	err = st.ApplyBlock(b.UnsignedBlock)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = c1.CommitAppliedBlock(context.Background(), b, st)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	for i := 0; i < 5; i++ {
		store.SaveBlock(context.Background(), &bc.Block{
			UnsignedBlock: &bc.UnsignedBlock{
				BlockHeader: &bc.BlockHeader{
					Height:        uint64(i + 2),
					NextPredicate: &bc.Predicate{},
					ContractsRoot: &bc.Hash{},
				},
			},
		})
	}

	c2, err := NewChain(context.Background(), b, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	state, err := c2.Recover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.Height() != 6 {
		t.Fatalf("state.Height = %d, want %d", state.Height(), 1)
	}

	curState := c2.State()
	if curState.Height() != 6 {
		t.Fatalf("chain.state.Height = %d, want %d", curState.Height(), 1)
	}
	if curState.Header == nil {
		t.Fatal("chain.state.Header is nil")
	}
}
