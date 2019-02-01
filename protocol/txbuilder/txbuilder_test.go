package txbuilder

import (
	"bytes"
	"context"
	"math"
	"testing"
	"time"

	"github.com/chain/txvm/crypto/ed25519"
	"github.com/chain/txvm/crypto/sha3pool"
	"github.com/chain/txvm/errors"
	"github.com/chain/txvm/protocol/bc"
	"github.com/chain/txvm/protocol/txbuilder/standard"
	"github.com/chain/txvm/protocol/txbuilder/txresult"
	"github.com/chain/txvm/protocol/txvm"
	"github.com/chain/txvm/testutil"
)

// duplicated from core/key/store.go.
func keyHash(k []byte) []byte {
	var h [32]byte
	sha3pool.Sum256(h[:], k)
	return h[:]
}

func TestTxBuilder(t *testing.T) {
	assetID := bc.HashFromBytes([]byte{1})
	keyIDs := [][]byte{keyHash(testutil.TestXPub[:])}
	pubkeys := []ed25519.PublicKey{testutil.TestPub}

	cases := []struct {
		name    string
		pre     func(*Template)
		post    func(*testing.T, *bc.Tx, error)
		wanterr error // ignored unless post == nil
	}{
		{
			name:    "empty",
			wanterr: ErrNoAnchor,
		},
		{
			name: "insufficient input transfer",
			pre: func(tpl *Template) {
				tpl.AddInput(1, keyIDs, nil, pubkeys, 99, assetID, []byte{1}, nil, 0)
				tpl.AddOutput(1, nil, 100, assetID, nil, nil)
			},
			wanterr: ErrInsufficientValue,
		},
		{
			name: "insufficient input issuance",
			pre: func(tpl *Template) {
				tpl.AddIssuance(2, []byte{1}, nil, 1, keyIDs, nil, pubkeys, 99, nil, nil)
				newAsset := tpl.Issuances[0].AssetID()
				tpl.AddOutput(1, nil, 100, bc.NewHash(newAsset), nil, nil)
			},
			wanterr: ErrInsufficientValue,
		},
		{
			name: "insufficient input retirement",
			pre: func(tpl *Template) {
				tpl.AddInput(1, keyIDs, nil, pubkeys, 99, assetID, []byte{1}, nil, 0)
				tpl.AddRetirement(100, assetID, nil)
			},
			wanterr: ErrInsufficientValue,
		},
		{
			name: "extra input transfer",
			pre: func(tpl *Template) {
				tpl.AddInput(1, keyIDs, nil, pubkeys, 100, assetID, []byte{1}, nil, 0)
				tpl.AddOutput(1, nil, 99, assetID, nil, nil)
			},
			wanterr: ErrNonSigCheck,
		},
		{
			name: "extra input issuance",
			pre: func(tpl *Template) {
				tpl.AddIssuance(2, []byte{1}, nil, 1, keyIDs, nil, pubkeys, 100, nil, nil)
				newAsset := tpl.Issuances[0].AssetID()
				tpl.AddOutput(1, nil, 99, bc.NewHash(newAsset), nil, nil)
			},
			wanterr: ErrNonSigCheck,
		},
		{
			name: "extra input retirement",
			pre: func(tpl *Template) {
				tpl.AddInput(1, keyIDs, nil, pubkeys, 100, assetID, []byte{1}, nil, 0)
				tpl.AddRetirement(99, assetID, nil)
			},
			wanterr: ErrNonSigCheck,
		},
		{
			name: "merge issuances",
			pre: func(tpl *Template) {
				tpl.AddIssuance(2, []byte{1}, nil, 1, keyIDs, nil, pubkeys, 49, nil, nil)
				tpl.AddIssuance(2, []byte{1}, nil, 1, keyIDs, nil, pubkeys, 47, nil, nil)
				tpl.AddIssuance(2, []byte{1}, nil, 1, keyIDs, nil, pubkeys, 4, nil, nil)
				newAsset := tpl.Issuances[0].AssetID()
				tpl.AddOutput(1, nil, 100, bc.NewHash(newAsset), nil, nil)
			},
			post: func(t *testing.T, tx *bc.Tx, err error) {
				if tx == nil {
					t.Fatal("nil tx")
				}
				// A tx with multiple issuances should only produce one nonce.  (The
				// anchor for the second comes from the value produced by the first.)
				if len(tx.Nonces) != 1 {
					t.Errorf("got %d nonces, want 1", len(tx.Nonces))
				}
				checkLogTypes(t, tx.Log, 0, 1, 0, 3, 1)
			},
		},
		{
			name: "split issuance",
			pre: func(tpl *Template) {
				tpl.AddIssuance(2, []byte{1}, nil, 1, keyIDs, nil, pubkeys, 100, nil, nil)
				newAsset := tpl.Issuances[0].AssetID()
				tpl.AddOutput(1, nil, 49, bc.NewHash(newAsset), nil, nil)
				tpl.AddOutput(1, nil, 47, bc.NewHash(newAsset), nil, nil)
				tpl.AddOutput(1, nil, 4, bc.NewHash(newAsset), nil, nil)
			},
			post: func(t *testing.T, tx *bc.Tx, err error) {
				if tx == nil {
					t.Fatal("nil tx")
				}
				checkLogTypes(t, tx.Log, 0, 3, 0, 1, 1)
			},
		},
		{
			name: "transfer many to many",
			pre: func(tpl *Template) {
				tpl.AddInput(1, keyIDs, nil, pubkeys, 52, assetID, []byte{1}, nil, 0)
				tpl.AddInput(1, keyIDs, nil, pubkeys, 48, assetID, []byte{1}, nil, 0)
				tpl.AddOutput(1, nil, 49, assetID, nil, nil)
				tpl.AddOutput(1, nil, 47, assetID, nil, nil)
				tpl.AddOutput(1, nil, 4, assetID, nil, nil)
			},
			post: func(t *testing.T, tx *bc.Tx, err error) {
				if tx == nil {
					t.Fatal("nil tx")
				}
				checkLogTypes(t, tx.Log, 2, 3, 0, 0, 1)
			},
		},
		{
			name: "merge to retirement",
			pre: func(tpl *Template) {
				tpl.AddInput(1, keyIDs, nil, pubkeys, 49, assetID, []byte{1}, nil, 0)
				tpl.AddInput(1, keyIDs, nil, pubkeys, 47, assetID, []byte{1}, nil, 0)
				tpl.AddInput(1, keyIDs, nil, pubkeys, 4, assetID, []byte{1}, nil, 0)
				tpl.AddRetirement(100, assetID, nil)
			},
			post: func(t *testing.T, tx *bc.Tx, err error) {
				if tx == nil {
					t.Fatal("nil tx")
				}
				checkLogTypes(t, tx.Log, 3, 0, 1, 0, 1)
			},
		},
		{
			name: "issue and spend",
			pre: func(tpl *Template) {
				tpl.AddIssuance(2, []byte{1}, nil, 1, keyIDs, nil, pubkeys, 49, nil, nil)
				newAsset := tpl.Issuances[0].AssetID()
				tpl.AddInput(1, keyIDs, nil, pubkeys, 47, bc.NewHash(newAsset), []byte{1}, nil, 0)
				tpl.AddInput(1, keyIDs, nil, pubkeys, 4, bc.NewHash(newAsset), []byte{1}, nil, 0)
				tpl.AddIssuance(2, []byte{1}, nil, 1, keyIDs, nil, pubkeys, 1, nil, nil)
				tpl.AddOutput(1, nil, 100, bc.NewHash(newAsset), nil, nil)
				tpl.AddOutput(1, nil, 1, bc.NewHash(newAsset), nil, nil)
			},
			post: func(t *testing.T, tx *bc.Tx, err error) {
				if tx == nil {
					t.Fatal("nil tx")
				}
				checkLogTypes(t, tx.Log, 2, 2, 0, 2, 1)
			},
		},
		{
			name: "output and retire",
			pre: func(tpl *Template) {
				tpl.AddInput(1, keyIDs, nil, pubkeys, 99, assetID, []byte{1}, nil, 0)
				tpl.AddInput(1, keyIDs, nil, pubkeys, 1, assetID, []byte{1}, nil, 0)
				tpl.AddRetirement(49, assetID, nil)
				tpl.AddOutput(1, nil, 47, assetID, nil, nil)
				tpl.AddRetirement(4, assetID, nil)
			},
			post: func(t *testing.T, tx *bc.Tx, err error) {
				if tx == nil {
					t.Fatal("nil tx")
				}
				checkLogTypes(t, tx.Log, 2, 1, 2, 0, 1)
			},
		},
		{
			name: "complex transaction",
			pre: func(tpl *Template) {
				tpl.AddIssuance(2, []byte{1}, nil, 1, keyIDs, nil, pubkeys, 49, nil, nil)
				newAsset := tpl.Issuances[0].AssetID()
				tpl.AddInput(1, keyIDs, nil, pubkeys, 47, bc.NewHash(newAsset), []byte{1}, nil, 0)
				tpl.AddInput(1, keyIDs, nil, pubkeys, 4, bc.NewHash(newAsset), []byte{1}, nil, 0)
				tpl.AddIssuance(2, []byte{1}, nil, 1, keyIDs, nil, pubkeys, 1, nil, nil)
				tpl.AddOutput(1, nil, 100, bc.NewHash(newAsset), nil, nil)
				tpl.AddOutput(1, nil, 1, bc.NewHash(newAsset), nil, nil)
				tpl.AddInput(1, keyIDs, nil, pubkeys, 99, assetID, []byte{1}, nil, 0)
				tpl.AddInput(1, keyIDs, nil, pubkeys, 1, assetID, []byte{1}, nil, 0)
				tpl.AddRetirement(49, assetID, nil)
				tpl.AddOutput(1, nil, 47, assetID, nil, nil)
				tpl.AddRetirement(4, assetID, nil)
			},
			post: func(t *testing.T, tx *bc.Tx, err error) {
				if tx == nil {
					t.Fatal("nil tx")
				}
				checkLogTypes(t, tx.Log, 4, 3, 2, 2, 1)
			},
		},
		{
			name: "mintime & maxtime",
			pre: func(tpl *Template) {
				tpl.AddIssuance(2, []byte{1}, nil, 1, keyIDs, nil, pubkeys, 10, nil, nil)
				newAsset := tpl.Issuances[0].AssetID()
				tpl.AddOutput(1, nil, 10, bc.NewHash(newAsset), nil, nil)
				tpl.RestrictMaxTime(time.Unix(0, int64(999*time.Millisecond)))
				tpl.RestrictMinTime(time.Unix(0, int64(100*time.Millisecond)))
			},
			post: func(t *testing.T, tx *bc.Tx, err error) {
				if tx == nil {
					t.Fatal("nil tx")
				}
				tr := tx.Log[len(tx.Log)-3]
				if tr[2].(txvm.Int) != 100 {
					t.Errorf("got mintime = %d want 100", tr[2].(txvm.Int))
				}
				if tr[3].(txvm.Int) != 999 {
					t.Errorf("got maxtime = %d want 999", tr[3].(txvm.Int))
				}
			},
		},
		{
			name: "merge big issuances",
			pre: func(tpl *Template) {
				tpl.AddIssuance(2, []byte{1}, nil, 1, keyIDs, nil, pubkeys, math.MaxInt64-10, nil, nil)
				tpl.AddIssuance(2, []byte{1}, nil, 1, keyIDs, nil, pubkeys, 20, nil, nil)
				newAsset := tpl.Issuances[0].AssetID()
				tpl.AddOutput(1, nil, math.MaxInt64-5, bc.NewHash(newAsset), nil, nil)
				tpl.AddOutput(1, nil, 15, bc.NewHash(newAsset), nil, nil)
			},
			post: func(t *testing.T, tx *bc.Tx, err error) {
				if tx == nil {
					t.Fatal("nil tx")
				}
				checkLogTypes(t, tx.Log, 0, 2, 0, 2, 1)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tpl := &Template{MaxTimeMS: bc.Millis(time.Now().Add(time.Minute))}
			if c.pre != nil {
				c.pre(tpl)
			}
			tx, err := tpl.Tx()
			if c.post != nil {
				c.post(t, tx, err)
			} else if errors.Root(err) != c.wanterr {
				if err == nil {
					t.Errorf("got nil error, want %s", c.wanterr)
				} else if c.wanterr == nil {
					t.Errorf("got error %s, want nil", err)
				} else {
					t.Errorf("got error %s, want %s", err, c.wanterr)
				}
			}
		})
	}
}

func TestResults(t *testing.T) {
	var (
		keyIDs  = [][]byte{keyHash(testutil.TestXPub[:])}
		pubkey  = testutil.TestPub
		assetID = bc.HashFromBytes([]byte{2})
	)

	cases := []struct {
		name string
		pre  func(*Template)
		post func(*txresult.Result)
	}{
		{
			name: "transfer",
			pre: func(tpl *Template) {
				tpl.AddInput(1, keyIDs, nil, []ed25519.PublicKey{pubkey}, 10, assetID, []byte("anchor"), []byte("inrefdata"), 0)
				tpl.AddOutput(1, []ed25519.PublicKey{pubkey}, 10, assetID, []byte("refdata"), nil)
				sign(t, tpl)
			},
			post: func(res *txresult.Result) {
				if len(res.Inputs) != 1 {
					t.Fatalf("got %d inputs, want 1", len(res.Inputs))
				}
				inp := res.Inputs[0]
				if inp.Value == nil {
					t.Error("inp.Value is nil")
				} else {
					if inp.Value.Amount != 10 {
						t.Errorf("got input amount %d, want 10", inp.Value.Amount)
					}
					if inp.Value.AssetID != assetID {
						t.Errorf("got input asset ID %x, want 0100...", inp.Value.AssetID.Bytes())
					}
					if !bytes.Equal(inp.Value.Anchor, []byte("anchor")) {
						t.Errorf("got input anchor %x, want 'anchor'", inp.Value.Anchor)
					}
				}
				if !bytes.Equal(inp.RefData, []byte("inrefdata")) {
					t.Errorf("got input inrefdata %x, want 'inrefdata'", inp.RefData)
				}

				if len(res.Outputs) != 1 {
					t.Fatalf("got %d outputs, want 1", len(res.Outputs))
				}
				out := res.Outputs[0]
				if out.Value == nil {
					t.Error("out.Value is nil")
				} else {
					if out.Value.Amount != 10 {
						t.Errorf("got output amount %d, want 10", out.Value.Amount)
					}
					if out.Value.AssetID != assetID {
						t.Errorf("got output asset ID %x, want 0100...", out.Value.AssetID.Bytes())
					}
				}
				if !bytes.Equal(out.RefData, []byte("refdata")) {
					t.Errorf("got output refdata %x, want 'refdata'", out.RefData)
				}
				if len(out.Pubkeys) != 1 {
					t.Errorf("got %d output pubkeys, want 1", len(out.Pubkeys))
				} else if !bytes.Equal(out.Pubkeys[0], pubkey) {
					t.Errorf("got output pubkey %x, want %x", out.Pubkeys[0], pubkey)
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tpl := &Template{MaxTimeMS: bc.Millis(time.Now().Add(time.Minute))}
			c.pre(tpl)
			tx, err := tpl.Tx()
			if err != nil {
				t.Fatal(err)
			}
			res := txresult.New(tx)
			c.post(res)
		})
	}
}

func TestSigBits(t *testing.T) {
	ctx := context.Background()

	tpl := &Template{MaxTimeMS: bc.Millis(time.Now().Add(time.Minute))}

	pub, prv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	pubkeys := []ed25519.PublicKey{pub}
	assetID := bc.NewHash(standard.AssetID(2, 1, pubkeys, nil))

	tpl.AddIssuance(2, nil, nil, 1, [][]byte{{0}}, nil, pubkeys, 1, nil, nil)
	tpl.AddOutput(0, nil, 1, assetID, nil, nil)
	err = tpl.Sign(ctx, func(_ context.Context, msg []byte, keyID []byte, _ [][]byte) ([]byte, error) {
		return ed25519.Sign(prv, msg), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = tpl.Tx()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(tpl.Issuances[0].Sigs[0]); i++ {
		for j := uint(0); j < 8; j++ {
			tpl.Issuances[0].Sigs[0][i] ^= 1 << j
			tpl.Dematerialize()
			_, err = tpl.Tx()
			if err == nil {
				t.Errorf("got no error with twiddled bit %d", uint(8*i)+j)
			}
			tpl.Issuances[0].Sigs[0][i] ^= 1 << j // put it back the way it was
		}
	}
}

func TestMultisig(t *testing.T) {
	ctx := context.Background()

	tpl := &Template{MaxTimeMS: bc.Millis(time.Now().Add(time.Minute))}

	var (
		prvkeys []ed25519.PrivateKey
		pubkeys []ed25519.PublicKey
	)
	for i := 0; i < 3; i++ {
		pub, prv, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		prvkeys = append(prvkeys, prv)
		pubkeys = append(pubkeys, pub)
	}

	assetID := bc.NewHash(standard.AssetID(2, 2, pubkeys, nil))

	tpl.AddIssuance(2, nil, nil, 2, [][]byte{{0}, {1}, {2}}, nil, pubkeys, 1, nil, nil)
	tpl.AddOutput(0, nil, 1, assetID, nil, nil)
	err := tpl.Sign(ctx, func(_ context.Context, msg []byte, keyID []byte, _ [][]byte) ([]byte, error) {
		if bytes.Equal(keyID, []byte{0}) {
			return ed25519.Sign(prvkeys[0], msg), nil
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = tpl.Tx()
	if err == nil {
		t.Errorf("got no error with 1 signature for 2-of-3 issuance")
	}

	tpl.Dematerialize()
	err = tpl.Sign(ctx, func(_ context.Context, msg []byte, keyID []byte, _ [][]byte) ([]byte, error) {
		if bytes.Equal(keyID, []byte{1}) {
			return ed25519.Sign(prvkeys[1], msg), nil
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = tpl.Tx()
	if err != nil {
		t.Errorf("got error %s with 2 signatures for 2-of-3 issuance, want no error", err)
	}
}

func sign(t *testing.T, tpl *Template) {
	ctx := context.Background()
	err := tpl.Sign(ctx, func(_ context.Context, data []byte, _ []byte, path [][]byte) ([]byte, error) {
		derived := testutil.TestXPrv.Derive(path)
		return derived.Sign(data), nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func checkLogTypes(t *testing.T, log []txvm.Tuple,
	inputCount, outputCount, retireCount, issueCount, finalizeCount int) {
	counts := make(map[byte]int)
	for _, l := range log {
		typeCode := l[0].(txvm.Bytes)[0]
		counts[typeCode]++
	}
	ops := []struct {
		code  byte
		count int
	}{
		{code: txvm.InputCode, count: inputCount},
		{code: txvm.OutputCode, count: outputCount},
		{code: txvm.RetireCode, count: retireCount},
		{code: txvm.IssueCode, count: issueCount},
		{code: txvm.FinalizeCode, count: finalizeCount},
	}

	for _, o := range ops {
		if counts[o.code] != o.count {
			t.Errorf("got %d instance(s) of '%c' in tx log, want %d", counts[o.code], o.code, o.count)
		}
	}
}
