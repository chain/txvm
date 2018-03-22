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
	err = st.ApplyBlock(b)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = c1.CommitAppliedBlock(context.Background(), b, st)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Snapshots are applied asynchronously. This loops waits
	// until the snapshot is created.
	for {
		snap, _ := store.LatestSnapshot(context.Background())
		if snap.Height() > 0 {
			break
		}
	}

	ctx := context.Background()

	c2, err := NewChain(context.Background(), b, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	state, err := c2.Recover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state.Height() != 1 {
		t.Fatalf("state.Height = %d, want %d", state.Height(), 1)
	}

	curState := c2.State()
	if curState.Height() != 1 {
		t.Fatalf("chain.state.Height = %d, want %d", curState.Height(), 1)
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
	err = st.ApplyBlock(b)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = c1.CommitAppliedBlock(context.Background(), b, st)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	for i := 0; i < 5; i++ {
		store.SaveBlock(context.Background(), &bc.Block{
			BlockHeader: &bc.BlockHeader{
				Height:        uint64(i + 2),
				NextPredicate: &bc.Predicate{},
				ContractsRoot: &bc.Hash{},
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
