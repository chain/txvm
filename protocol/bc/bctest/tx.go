package bctest

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"i10r.io/protocol/bc"
	"i10r.io/protocol/txvm/asm"
	"i10r.io/testutil"
)

// EmptyTx produces a minimal valid transaction from "nonce" and
// "finalize". The supplied blockID and exp are for computing the
// nonce. Some random bytes are mixed in as well, so the resulting tx
// is always unique.
func EmptyTx(t testing.TB, blockID bc.Hash, exp time.Time) *bc.Tx {
	var nonce [32]byte
	_, err := rand.Read(nonce[:])
	if err != nil {
		testutil.FatalErr(t, err)
	}

	raw, err := asm.Assemble(fmt.Sprintf(`
		[x'%x' drop
		 x'%x'
		 %d
		 nonce put
		] contract call
		get finalize
		`, nonce[:], blockID.Bytes(), bc.Millis(exp)))
	if err != nil {
		testutil.FatalErr(t, err)
	}

	tx, err := bc.NewTx(raw, 3, 1000)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return tx
}

// WithCommitments takes in a slice of *bc.Tx and wraps each of them with
// their commitments, returning a slice of *bc.CommitmentsTx
func WithCommitments(txs []*bc.Tx) []*bc.CommitmentsTx {
	var commitmentsTxs []*bc.CommitmentsTx
	for _, tx := range txs {
		commitmentsTxs = append(commitmentsTxs, bc.NewCommitmentsTx(tx))
	}
	return commitmentsTxs
}
