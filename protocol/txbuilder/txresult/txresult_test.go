package txresult

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/chain/txvm/crypto/ed25519"
	"github.com/chain/txvm/crypto/sha3pool"
	"github.com/chain/txvm/protocol/bc"
	"github.com/chain/txvm/protocol/txbuilder"
	"github.com/chain/txvm/testutil"
)

// duplicated from core/key/store.go.
func keyHash(k []byte) []byte {
	var h [32]byte
	sha3pool.Sum256(h[:], k)
	return h[:]
}

func TestResult(t *testing.T) {
	var (
		keyIDs  = [][]byte{keyHash(testutil.TestXPub[:])}
		pubkeys = []ed25519.PublicKey{testutil.TestPub}
		tpl     = &txbuilder.Template{MaxTimeMS: bc.Millis(time.Now().Add(time.Minute))}
	)
	tpl.AddIssuance(2, []byte{1}, nil, 1, keyIDs, nil, pubkeys, 100, nil, nil)
	newAsset := tpl.Issuances[0].AssetID()
	tpl.AddOutput(1, nil, 100, bc.NewHash(newAsset), nil, nil)
	err := tpl.Sign(context.Background(), func(_ context.Context, data, _ []byte, path [][]byte) ([]byte, error) {
		derived := testutil.TestXPrv.Derive(path)
		return derived.Sign(data), nil
	})
	tx, err := tpl.Tx()
	if err != nil {
		t.Fatal(err)
	}
	txr := New(tx)
	if len(txr.Issuances) != 1 {
		t.Fatalf("got %d issuances, want 1", len(txr.Issuances))
	}
	if txr.Issuances[0].Quorum != 1 {
		t.Errorf("got issuance quorum %d, want 1", txr.Issuances[0].Quorum)
	}
	if len(txr.Issuances[0].Pubkeys) != 1 {
		t.Fatalf("got %d issuance pubkeys, want 1", len(txr.Issuances[0].Pubkeys))
	}
	if !bytes.Equal(txr.Issuances[0].Pubkeys[0], testutil.TestPub) {
		t.Errorf("got issuance pubkey %x, want %x", txr.Issuances[0].Pubkeys[0], testutil.TestPub)
	}
}

// TODO(bobg): more tests needed.
