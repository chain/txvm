package protocol

import (
	"context"
	"math"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/chain/txvm/protocol/bc"
	"github.com/chain/txvm/protocol/bc/bctest"
	"github.com/chain/txvm/protocol/patricia"
	"github.com/chain/txvm/protocol/prottest/memstore"
	"github.com/chain/txvm/protocol/state"
	"github.com/chain/txvm/testutil"
)

func TestGetBlock(t *testing.T) {
	ctx := context.Background()

	b1 := &bc.Block{BlockHeader: &bc.BlockHeader{Height: 1, NextPredicate: &bc.Predicate{}}}
	noBlocks := memstore.New()
	oneBlock := memstore.New()
	oneBlock.SaveBlock(ctx, b1)
	snapshot := state.Empty()
	snapshot.ApplyBlock(b1)
	oneBlock.SaveSnapshot(ctx, snapshot)

	cases := []struct {
		store   Store
		want    *bc.Block
		wantErr bool
	}{
		{noBlocks, nil, true},
		{oneBlock, b1, false},
	}

	for _, test := range cases {
		c, err := NewChain(ctx, b1, test.store, nil)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		got, gotErr := c.GetBlock(ctx, c.Height())
		if !testutil.DeepEqual(got, test.want) {
			t.Errorf("got latest = %+v want %+v", got, test.want)
		}
		if (gotErr != nil) != test.wantErr {
			t.Errorf("got latest err = %q want err?: %t", gotErr, test.wantErr)
		}
	}
}

func TestNoTimeTravel(t *testing.T) {
	b1 := &bc.Block{BlockHeader: &bc.BlockHeader{Height: 1, NextPredicate: &bc.Predicate{}}}
	ctx := context.Background()
	c, err := NewChain(ctx, b1, memstore.New(), nil)
	if err != nil {
		t.Fatal(err)
	}

	c.setHeight(1)
	c.setHeight(2)

	c.setHeight(1) // don't go backward
	if c.state.height != 2 {
		t.Fatalf("c.state.height = %d want 2", c.state.height)
	}
}

func TestWaitForBlockSoonAlreadyExists(t *testing.T) {
	c, _ := newTestChain(t, time.Now())
	makeEmptyBlock(t, c) // height=2
	makeEmptyBlock(t, c) // height=3

	err := <-c.BlockSoonWaiter(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWaitForBlockSoonDistantFuture(t *testing.T) {
	c, _ := newTestChain(t, time.Now())

	got := <-c.BlockSoonWaiter(context.Background(), 100) // distant future
	want := ErrTheDistantFuture
	if got != want {
		t.Errorf("BlockSoonWaiter(100) = %+v want %+v", got, want)
	}
}

func TestWaitForBlockSoonWaits(t *testing.T) {
	// This test is inherently racy. It's possible
	// that the block creation might run before
	// the wait's internal test loop finds no block.
	// In that case, the test will pass, but it will
	// not have tested anything.
	//
	// It's the best we can do.

	c, _ := newTestChain(t, time.Now())
	makeEmptyBlock(t, c) // height=2

	go func() {
		time.Sleep(10 * time.Millisecond) // sorry for the slow test 
		makeEmptyBlock(t, c)              // height=3
	}()

	err := <-c.BlockSoonWaiter(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if g := c.Height(); g != 3 {
		t.Errorf("height after waiting = %d want 3", g)
	}
}

func TestWaitForBlockSoonTimesout(t *testing.T) {
	c, _ := newTestChain(t, time.Now())
	go func() {
		makeEmptyBlock(t, c) // height=2
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := <-c.BlockSoonWaiter(ctx, 3)
	if err != ctx.Err() {
		t.Fatalf("expected timeout err, got %v", err)
	}
}

func TestGenerateBlock(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(233400000, 0)
	c, b1 := newTestChain(t, now)

	txs := []*bc.Tx{
		{ID: bc.NewHash([32]byte{1}), Contracts: []bc.Contract{{Type: bc.OutputType, ID: bc.NewHash([32]byte{2})}}},
		{ID: bc.NewHash([32]byte{3}), Contracts: []bc.Contract{{Type: bc.OutputType, ID: bc.NewHash([32]byte{4})}}},
	}

	st := state.Empty()
	err := st.ApplyBlock(b1)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := c.GenerateBlock(ctx, st, bc.Millis(now)+1, txs)
	if err != nil {
		t.Fatalf("err got = %v want nil", err)
	}

	// TODO(bobg): verify these hashes are correct
	wantTxRoot := mustDecodeHash("e437b69d1dd70254e165163415e69830b8cbf2eded94b79aa5de911e0691a89f")
	wantContractsRoot := mustDecodeHash("5ff56d780f78809e63fb7be7fdbd7bf825704914311b4d0819c50d411f3b662d")
	wantNoncesRoot := new(patricia.Tree).RootHash()

	b1ID := b1.Hash()
	want := &bc.Block{
		BlockHeader: &bc.BlockHeader{
			Version:          3,
			Height:           2,
			RefsCount:        1,
			PreviousBlockId:  &b1ID,
			TimestampMs:      bc.Millis(now) + 1,
			TransactionsRoot: &wantTxRoot,
			ContractsRoot:    &wantContractsRoot,
			NoncesRoot:       &wantNoncesRoot,
			NextPredicate:    b1.NextPredicate,
		},
		Transactions: txs,
	}

	if !testutil.DeepEqual(got, want) {
		t.Errorf("generated block:\ngot:  %+v\nwant: %+v", got, want)
	}

	_, _, err = c.GenerateBlock(ctx, st, bc.Millis(now), nil)
	if err == nil {
		t.Error("expected error for bad generate timestamp")
	}

	for i := 0; i < maxBlockTxs+1; i++ {
		txs = append(txs, bctest.EmptyTx(t, b1.Hash(), now.Add(time.Minute)))
	}

	got, _, err = c.GenerateBlock(ctx, st, bc.Millis(now)+1, txs)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Transactions) != maxBlockTxs {
		t.Errorf("expected block to have maximum number of txs, got %d", len(got.Transactions))
	}

	txs = []*bc.Tx{txs[0], txs[1]}
	txs[0].Runlimit = 500
	txs[1].Runlimit = math.MaxInt64

	got, _, err = c.GenerateBlock(ctx, st, bc.Millis(now)+1, txs)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Transactions) != 1 {
		t.Errorf("expected block to have 1 tx due to runlimit, got %d", len(got.Transactions))
	}

	txs = []*bc.Tx{
		{ID: bc.NewHash([32]byte{1}), Contracts: []bc.Contract{{Type: bc.InputType, ID: bc.NewHash([32]byte{2})}}},
	}

	got, _, err = c.GenerateBlock(ctx, st, bc.Millis(now)+1, txs)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Transactions) != 0 {
		t.Errorf("expected block to have no txs due to tx error, got %d", len(got.Transactions))
	}
}

func TestCommitBlockIdempotence(t *testing.T) {
	const numOfBlocks = 10
	const concurrency = 5
	ctx := context.Background()

	now := time.Now()
	c, b1 := newTestChain(t, now)

	var blocks []*bc.Block
	s := state.Empty()
	s.ApplyBlock(b1)
	for i := 0; i < numOfBlocks; i++ {
		tx := &bc.Tx{ID: bc.NewHash([32]byte{byte(i)})}
		newBlock, newSnapshot, err := c.GenerateBlock(ctx, s, bc.Millis(now)+uint64(i+1), []*bc.Tx{tx})
		if err != nil {
			testutil.FatalErr(t, err)
		}
		err = c.CommitAppliedBlock(ctx, newBlock, newSnapshot)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		blocks = append(blocks, newBlock)
		s = newSnapshot
	}
	wantSnapshot := s

	// Create a fresh Chain for the same blockchain / initial hash.
	c, err := NewChain(ctx, b1, memstore.New(), nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	c.MaxNonceWindow = 48 * time.Hour
	snapshot := state.Empty()
	snapshot.ApplyBlock(b1)
	c.setState(snapshot)

	// Apply all of the blocks concurrently in separate goroutines
	// using CommitBlock. They should all succeed.
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < len(blocks); j++ {
				err := c.CommitBlock(ctx, blocks[j])
				if err != nil {
					testutil.FatalErr(t, err)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	gotSnapshot := c.State()
	if !reflect.DeepEqual(gotSnapshot, wantSnapshot) {
		t.Errorf("got snapshot:\n%swant snapshot:\n%s", spew.Sdump(gotSnapshot), spew.Sdump(wantSnapshot))
	}
}

// newTestChain returns a new Chain using memstore for storage,
// along with an initial block b1 (with a 0/0 multisig program).
// It commits b1 before returning.
func newTestChain(tb testing.TB, ts time.Time) (c *Chain, b1 *bc.Block) {
	ctx := context.Background()

	var err error

	b1, err = NewInitialBlock(nil, 0, ts)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	c, err = NewChain(ctx, b1, memstore.New(), nil)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	// TODO(tessr): consider adding MaxNonceWindow to NewChain
	c.MaxNonceWindow = 48 * time.Hour
	c.MaxBlockWindow = 100
	st := state.Empty()
	err = st.ApplyBlock(b1)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	err = c.CommitAppliedBlock(ctx, b1, st)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	return c, b1
}

func makeEmptyBlock(tb testing.TB, c *Chain) {
	ctx := context.Background()

	curState := c.State()
	nextBlock, nextState, err := c.GenerateBlock(ctx, curState, curState.TimestampMS()+1, nil)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	err = c.CommitAppliedBlock(ctx, nextBlock, nextState)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
}

func mustDecodeHash(s string) (h bc.Hash) {
	err := h.UnmarshalText([]byte(s))
	if err != nil {
		panic(err)
	}
	return h
}
