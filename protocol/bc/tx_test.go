package bc

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"i10r.io/protocol/txvm"
	"i10r.io/protocol/txvm/asm"
	"i10r.io/protocol/txvm/op"
	"i10r.io/protocol/txvm/txvmtest"
	"i10r.io/testutil"
)

func TestNewTx(t *testing.T) {
	cases := []struct {
		name string
		asm  string
		want *Tx
	}{
		{
			name: "simple payment",
			asm:  txvmtest.SimplePayment,
			want: &Tx{
				RawTx: RawTx{
					Version:  3,
					Runlimit: 100000,
				},
				Finalized: true,
				ID:        mustDecodeHash("c10da3fb9bed06674f9e247c5fff60c712c7d47881f3843a0c75d91544ed76b8"),
				Contracts: []Contract{
					{InputType, mustDecodeHash("7229e653bd7c21efae174d7d3e8087ea8e5e1d074adc59a1dfbd88c484ead9ea")},
					{OutputType, mustDecodeHash("333b102f5eebf7450cced735b1a2518f98f706b52828197d6bd70229e2e669f5")},
				},
				Anchor: mustDecodeHex("a4eb3b92e93f5889d7dd213530ee968c4f602ca45b3fc34d0936417d6daa59b0"),
				Inputs: []Input{{
					ID:   mustDecodeHash("7229e653bd7c21efae174d7d3e8087ea8e5e1d074adc59a1dfbd88c484ead9ea"),
					Seed: mustDecodeHash("636f6e7472616374736565640000000000000000000000000000000000000000"),
					Stack: []txvm.Data{
						txvm.Tuple{txvm.Bytes{txvm.BytesCode}, txvm.Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41"))},
						txvm.Tuple{
							txvm.Bytes{txvm.ValueCode},
							txvm.Int(10),
							txvm.Bytes(mustDecodeHex("d073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d")),
							txvm.Bytes([]byte("anchor")),
						},
					},
					Program: []byte{op.Put, op.MinPushdata + 7, op.TxID, op.MinSmallInt + 1, op.Roll, op.Get, op.MinSmallInt + 0, op.CheckSig, op.Verify, op.Yield},
					LogPos:  0,
				}},
				Outputs: []Output{{
					ID:   mustDecodeHash("333b102f5eebf7450cced735b1a2518f98f706b52828197d6bd70229e2e669f5"),
					Seed: mustDecodeHash("30b12caddb68c2da018ff46f7d358aa8b8ca9fdfd05c04478ccd5c9599583ad0"),
					Stack: []txvm.Data{
						txvm.Tuple{txvm.Bytes{txvm.BytesCode}, txvm.Bytes(mustDecodeHex("1111111111111111111111111111111111111111111111111111111111111111"))},
						txvm.Tuple{
							txvm.Bytes{txvm.ValueCode},
							txvm.Int(10),
							txvm.Bytes(mustDecodeHex("d073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d")),
							txvm.Bytes(mustDecodeHex("25703d95c689c0d5619fa7c011fd5df200e0e1ab46623ebeee36c75c1ef16241")),
						},
					},
					Program: []byte{op.Put, op.MinPushdata + 7, op.TxID, op.MinSmallInt + 1, op.Roll, op.Get, op.MinSmallInt + 0, op.CheckSig, op.Verify, op.Yield},
					LogPos:  1,
				}},
			},
		},
		{
			name: "issuance",
			asm:  txvmtest.Issuance,
			want: &Tx{
				RawTx: RawTx{
					Version:  3,
					Runlimit: 100000,
				},
				Finalized: true,
				ID:        mustDecodeHash("58bf0fea4ed326388836edfc1db8b24c35c4d804b962faa2890e4a0c5a0fda7a"),
				Contracts: []Contract{{OutputType, mustDecodeHash("2bc2a72073906c6745123d2a1c46c0623a2e3bf85c955abbdf1f14799985ee7a")}},
				Anchor:    mustDecodeHex("b820b13d533796a72bc4df57103cb5f85ed5d54eefd9858b7c919da47b4ba202"),
				Nonces: []Nonce{
					{
						ID:      mustDecodeHash("4f907f68e3a0f9e3094e7908af571f52dd5b6e84cc7602e501c25a2fd17f1fbb"),
						BlockID: mustDecodeHash("626c6f636b636861696e6964626c6f636b636861696e6964626c6f636b636861"),
						ExpMS:   10,
					},
				},
				Timeranges: []Timerange{
					{MinMS: 0, MaxMS: 10},
				},
				Issuances: []Issuance{{
					AssetID: mustDecodeHash("d8a92d34192c33551faaa500861e8bd4987847a356d54b7b2c8a6380b0bd0517"),
					Amount:  10,
					Anchor:  mustDecodeHex("4f907f68e3a0f9e3094e7908af571f52dd5b6e84cc7602e501c25a2fd17f1fbb"),
					LogPos:  2,
				}},
				Outputs: []Output{{
					ID:   mustDecodeHash("2bc2a72073906c6745123d2a1c46c0623a2e3bf85c955abbdf1f14799985ee7a"),
					Seed: mustDecodeHash("30b12caddb68c2da018ff46f7d358aa8b8ca9fdfd05c04478ccd5c9599583ad0"),
					Stack: []txvm.Data{
						txvm.Tuple{txvm.Bytes{txvm.BytesCode}, txvm.Bytes(mustDecodeHex("1111111111111111111111111111111111111111111111111111111111111111"))},
						txvm.Tuple{
							txvm.Bytes{txvm.ValueCode},
							txvm.Int(10),
							txvm.Bytes(mustDecodeHex("d8a92d34192c33551faaa500861e8bd4987847a356d54b7b2c8a6380b0bd0517")),
							txvm.Bytes(mustDecodeHex("4e85b8012d7ef8dbe848b605c7c22e22db5ffc401468d306065c521cea7223b3")),
						},
					},
					Program: []byte{op.Put, op.MinPushdata + 7, op.TxID, op.MinSmallInt + 1, op.Roll, op.Get, op.MinSmallInt + 0, op.CheckSig, op.Verify, op.Yield},
					LogPos:  3,
				}},
			},
		},
		{
			name: "retirement",
			asm:  txvmtest.Retirement,
			want: &Tx{
				RawTx: RawTx{
					Version:  3,
					Runlimit: 100000,
				},
				Finalized: true,
				ID:        mustDecodeHash("a42adebd554ef71ba557837f7318d2854f45993431c481dbefe9fa032dc0a3a5"),
				Contracts: []Contract{{InputType, mustDecodeHash("7fa08e4c10e99141e90cf0c43602c6f2647ce6397f435d31f8540f0f4f5e5f3c")}},
				Anchor:    mustDecodeHex("a4eb3b92e93f5889d7dd213530ee968c4f602ca45b3fc34d0936417d6daa59b0"),
				Inputs: []Input{{
					ID:   mustDecodeHash("7fa08e4c10e99141e90cf0c43602c6f2647ce6397f435d31f8540f0f4f5e5f3c"),
					Seed: mustDecodeHash("636f6e7472616374736565640000000000000000000000000000000000000000"),
					Stack: []txvm.Data{
						txvm.Tuple{txvm.Bytes{txvm.IntCode}, txvm.Int(1)},
						txvm.Tuple{txvm.Bytes{txvm.BytesCode}, txvm.Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41"))},
						txvm.Tuple{txvm.Bytes{txvm.IntCode}, txvm.Int(1)},
						txvm.Tuple{
							txvm.Bytes{txvm.ValueCode},
							txvm.Int(10),
							txvm.Bytes(mustDecodeHex("d073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d")),
							txvm.Bytes([]byte("anchor")),
						},
					},
					Program: []byte{
						op.Put,
						op.MinPushdata + 34, 0x01,
						op.Get, 0, 2, op.Roll, op.Dup, 0, op.Eq, 19, op.JumpIf, 2, op.Peek, 4, op.Roll,
						op.Get, 0, op.CheckSig, 2, op.Roll, op.Add, 1, op.Roll, 1, op.Neg, op.Add, 1, 24, op.Neg, op.JumpIf, op.Drop,
						2, op.Roll, op.Eq, op.Verify, op.Exec,
						op.Yield,
					},

					LogPos: 0,
				}},
				Retirements: []Retirement{{
					Amount:  10,
					AssetID: mustDecodeHash("d073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d"),
					Anchor:  txvm.Bytes(mustDecodeHex("25703d95c689c0d5619fa7c011fd5df200e0e1ab46623ebeee36c75c1ef16241")),
					LogPos:  1,
				}},
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			prog, err := asm.Assemble(c.asm)
			if err != nil {
				t.Fatal(err)
			}

			c.want.Program = prog

			tx, err := NewTx(prog, 3, 100000)
			if err != nil {
				t.Fatal(err)
			}

			c.want.Log = tx.Log

			if !reflect.DeepEqual(tx, c.want) {
				t.Errorf("NewTx\n\tgot:  %s\n\twant: %s\n", spew.Sdump(tx), spew.Sdump(c.want))
			}
		})
	}
}

func TestWitnessHash(t *testing.T) {
	raw, err := asm.Assemble(`"blockchainidblockchainidblockcha" 1000 nonce finalize`)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	tx, err := NewTx(raw, 3, 1000)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	want, _ := hex.DecodeString("039a050d18522e49736038842b0a759f7addf16e3c017d6aec5d976061062b13b0f95a7a8b977557c1f6d012207304f5e4e0349b2683090abe98c0705d8434af")

	var b bytes.Buffer
	_, err = tx.WriteWitnessCommitmentTo(&b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b.Bytes(), want) {
		t.Errorf("Tx.WriteWitnessCommitmentTo yields %x, want %x", b.Bytes(), want)
	}
}
