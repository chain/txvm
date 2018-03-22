package protocol

import (
	"github.com/chain/txvm/errors"
	"github.com/chain/txvm/protocol/bc"
)

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")

// CheckNonceWindow ensures that all nonces in tx expire within the
// MaxNonceWindow deadline.
func (c *Chain) CheckNonceWindow(tx *bc.Tx, blockTimeMS uint64) error {
	if c.MaxNonceWindow == 0 {
		return nil
	}
	for _, nonce := range tx.Nonces {
		if nonce.ExpMS > bc.DurationMillis(c.MaxNonceWindow)+blockTimeMS {
			return errors.WithDetailf(ErrBadTx, "nonce's time window is larger than the network maximum (%s)", c.MaxNonceWindow)
		}
	}
	return nil
}
