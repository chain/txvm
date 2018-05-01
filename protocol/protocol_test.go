package protocol

import (
	"context"
	"testing"

	"github.com/chain/txvm/protocol/bc"
	"github.com/chain/txvm/protocol/prottest/memstore"
)

func TestNewChainHeight(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	block := &bc.Block{BlockHeader: &bc.BlockHeader{NextPredicate: &bc.Predicate{}}}
	heights := make(chan uint64, 4)
	c, err := NewChain(ctx, block, memstore.New(), heights)
	if err != nil {
		t.Fatal(err)
	}

	heights <- 1
	heights <- 2
	heights <- 0
	heights <- 3

	<-c.BlockWaiter(3)
	if got := c.Height(); got != 3 {
		t.Errorf("c.Height() = %d, want %d", got, 3)
	}
	cancel()
}
