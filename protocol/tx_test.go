package protocol

import (
	"context"
	"testing"
	"time"

	"github.com/chain/txvm/protocol/bc"
)

func TestBadMaxNonceWindow(t *testing.T) {
	ctx := context.Background()
	c, b1 := newTestChain(t, time.Now())
	c.MaxNonceWindow = time.Second

	tx := &bc.Tx{
		Nonces: []bc.Nonce{{ExpMS: bc.Millis(time.Now().Add(5 * time.Second))}},
	}

	st := c.State()
	got, _, err := c.GenerateBlock(ctx, st, b1.TimestampMs+1, []*bc.CommitmentsTx{bc.NewCommitmentsTx(tx)})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Transactions) != 0 {
		t.Error("expected issuance past max issuance window to be rejected")
	}

	c.MaxNonceWindow = 0
	got, _, err = c.GenerateBlock(ctx, st, b1.TimestampMs+1, []*bc.CommitmentsTx{bc.NewCommitmentsTx(tx)})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Transactions) != 1 {
		t.Error("expected 0 max issuance to be ignored")
	}
}
