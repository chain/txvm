// Package merkle implements merkle binary trees.
package merkle

import (
	"math"

	"github.com/chain/txvm/crypto/sha3"
	"github.com/chain/txvm/crypto/sha3pool"
)

var (
	leafPrefix      = []byte{0x00}
	interiorPrefix  = []byte{0x01}
	emptyStringHash = sha3.Sum256(nil)
)

// Root creates a merkle tree from a slice of byte slices
// and returns the root hash of the tree.
func Root(items [][]byte) [32]byte {
	switch len(items) {
	case 0:
		return emptyStringHash

	case 1:
		h := sha3pool.Get256()
		defer sha3pool.Put256(h)

		h.Write(leafPrefix)
		h.Write(items[0])
		var root [32]byte
		h.Read(root[:])
		return root

	default:
		k := prevPowerOfTwo(len(items))
		left := Root(items[:k])
		right := Root(items[k:])

		h := sha3pool.Get256()
		defer sha3pool.Put256(h)
		h.Write(interiorPrefix)
		h.Write(left[:])
		h.Write(right[:])

		var root [32]byte
		h.Read(root[:])
		return root
	}
}

// prevPowerOfTwo returns the largest power of two that is smaller than a given number.
// In other words, for some input n, the prevPowerOfTwo k is a power of two such that
// k < n <= 2k. This is a helper function used during the calculation of a merkle tree.
func prevPowerOfTwo(n int) int {
	// If the number is a power of two, divide it by 2 and return.
	if n&(n-1) == 0 {
		return n / 2
	}

	// Otherwise, find the previous PoT.
	exponent := uint(math.Log2(float64(n)))
	return 1 << exponent // 2^exponent
}
