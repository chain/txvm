package protocol

import (
	"github.com/chain/txvm/errors"
	"github.com/chain/txvm/protocol/bc"
)

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")

// checkTransactionTime ensures that a transaction is fit
// to be included in a block generated at blockTimeMS.
func (c *Chain) checkTransactionTime(tx *bc.Tx, blockTimeMS uint64) error {
	for _, tr := range tx.Timeranges {
		if tr.MaxMS > 0 && blockTimeMS > uint64(tr.MaxMS) {
			return errors.WithDetailf(ErrBadTx, "transaction time range %d-%d too old", tr.MinMS, tr.MaxMS)
		}
		if tr.MinMS > 0 && blockTimeMS > 0 && blockTimeMS < uint64(tr.MinMS) {
			return errors.WithDetailf(ErrBadTx, "transaction time range %d-%d too far in the future", tr.MinMS, tr.MaxMS)
		}
	}

	if c.MaxNonceWindow > 0 {
		for _, nonce := range tx.Nonces {
			if nonce.ExpMS > bc.DurationMillis(c.MaxNonceWindow)+blockTimeMS {
				return errors.WithDetailf(ErrBadTx, "nonce's time window is larger than the network maximum (%s)", c.MaxNonceWindow)
			}
		}
	}
	return nil
}
