/*
Package validation implements the block-validation algorithms from the
Chain Protocol spec.
*/
package validation

import (
	"i10r.io/crypto/ed25519"
	"i10r.io/errors"
	"i10r.io/protocol/bc"
)

var (
	errMismatchedBlock       = errors.New("mismatched block")
	errMismatchedMerkleRoot  = errors.New("mismatched merkle root")
	errMisorderedBlockHeight = errors.New("misordered block height")
	errMisorderedBlockTime   = errors.New("misordered block time")
	errNoPrevBlock           = errors.New("no previous block")
	errTxVersion             = errors.New("invalid transaction version")
	errVersionRegression     = errors.New("version regression")
	errBadPredicate          = errors.New("invalid block predicate")
	errBadArguments          = errors.New("invalid block arguments for predicate")
	errRunlimit              = errors.New("block runlimit not sufficient for transactions")
	errRefsCount             = errors.New("refscount greater than allowed by previous block")
	errExtraFields           = errors.New("unknown field(s) in blockheader")
)

// BlockSig checks the predicate against b.
func BlockSig(b *bc.Block, predicate *bc.Predicate) error {
	if predicate.Version != 1 {
		return errors.WithDetailf(errBadPredicate, "predicate version %d", predicate.Version)
	}
	if predicate.Quorum < 0 {
		return errors.WithDetailf(errBadPredicate, "predicate quorum %d", predicate.Quorum)
	}
	if int(predicate.Quorum) > len(predicate.Pubkeys) {
		return errors.WithDetailf(errBadPredicate, "predicate quorum %d, pubkeys %d", predicate.Quorum, len(predicate.Pubkeys))
	}
	if len(b.Arguments) != len(predicate.Pubkeys) {
		return errors.WithDetailf(errBadArguments, "pubkeys %d, signatures %d", len(predicate.Pubkeys), len(b.Arguments))
	}

	var (
		sigCount int32
		hash     = b.Hash()
	)

	for i := 0; i < len(b.Arguments); i++ {
		pk := predicate.Pubkeys[i]
		if len(pk) != ed25519.PublicKeySize {
			return errors.WithDetailf(errBadPredicate, "public key length %d", len(pk))
		}

		sig, ok := b.Arguments[i].([]byte)
		if !ok {
			return errors.WithDetailf(errBadArguments, "invalid signature type %T", b.Arguments[i])
		}

		if len(sig) == 0 {
			continue
		}

		if len(sig) != ed25519.SignatureSize {
			return errors.WithDetailf(errBadArguments, "invalid signature length %d", len(sig))
		}

		if !ed25519.Verify(pk, hash.Bytes(), sig) {
			return errors.WithDetailf(errBadArguments, "message %x, public key %x, signature %x", hash.Bytes(), pk, sig)
		}

		sigCount++
	}

	if sigCount != predicate.Quorum {
		return errors.WithDetail(errBadArguments, "insufficient signatures for quorum")
	}

	return nil
}

// Block validates a block and the transactions within.
// It does not check the predicate; for that, see ValidateBlockSig.
func Block(b *bc.UnsignedBlock, prev *bc.BlockHeader) error {
	if b.Height > 1 {
		if prev == nil {
			return errors.WithDetailf(errNoPrevBlock, "height %d", b.Height)
		}
		err := BlockPrev(b, prev)
		if err != nil {
			return err
		}
	}

	return BlockOnly(b)
}

// BlockOnly performs those parts of block validation that depend only
// on the block and not on the previous block header.
// TODO(eric): consider another name
func BlockOnly(b *bc.UnsignedBlock) error {
	// TODO(bobg): check version >= 3?

	runlimit := b.Runlimit
	for _, tx := range b.Transactions {
		if b.Version == 3 && tx.Version != 3 {
			return errors.WithDetailf(errTxVersion, "block version %d, transaction version %d", b.Version, tx.Version)
		}

		runlimit -= tx.Runlimit
		if runlimit < 0 {
			return errors.Wrap(errRunlimit)
		}
	}

	txRoot := bc.TxMerkleRoot(b.Transactions)
	if txRoot != *b.TransactionsRoot {
		return errors.WithDetailf(errMismatchedMerkleRoot, "computed %x, current block wants %x", txRoot.Bytes(), b.TransactionsRoot.Bytes())
	}

	if b.Version == 3 && len(b.ExtraFields) > 0 {
		return errExtraFields
	}

	return nil
}

// BlockPrev performs those parts of block validation that require the
// previous block's header.
func BlockPrev(b *bc.UnsignedBlock, prev *bc.BlockHeader) error {
	if b.Version < prev.Version {
		return errors.WithDetailf(errVersionRegression, "previous block verson %d, current block version %d", prev.Version, b.Version)
	}
	if b.Height != prev.Height+1 {
		return errors.WithDetailf(errMisorderedBlockHeight, "previous block height %d, current block height %d", prev.Height, b.Height)
	}
	if prev.Hash() != *b.PreviousBlockId {
		return errors.WithDetailf(errMismatchedBlock, "previous block ID %x, current block wants %x", prev.Hash().Bytes(), b.PreviousBlockId.Bytes())
	}
	if b.TimestampMs <= prev.TimestampMs {
		return errors.WithDetailf(errMisorderedBlockTime, "previous block time %d, current block time %d", prev.TimestampMs, b.TimestampMs)
	}
	if b.RefsCount > prev.RefsCount+1 {
		return errors.WithDetailf(errRefsCount, "previous block prevblocks %d, current block %d", prev.RefsCount, b.RefsCount)
	}
	return nil
}
