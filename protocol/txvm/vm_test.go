package txvm_test

import (
	"bytes"
	"testing"
	"testing/quick"

	"github.com/chain/txvm/errors"
	"github.com/chain/txvm/protocol/txvm"
	"github.com/chain/txvm/protocol/txvm/asm"
	"github.com/chain/txvm/protocol/txvm/op"
	"github.com/chain/txvm/protocol/txvm/txvmtest"
)

func TestVMFuzz(t *testing.T) {
	check := func(prog []byte) (ok bool) {
		defer func() {
			if r := recover(); r != nil {
				t.Log(r)
				ok = false
			}
		}()
		txvm.Validate(prog, 3, 1000000)
		return true
	}
	err := quick.Check(check, &quick.Config{MaxCountScale: 1000.0})
	if err != nil {
		t.Error(err)
	}
}

func TestNoValidate(t *testing.T) {
	// It is not an error to validate a program with no finalize in it
	_, err := txvm.Validate([]byte{0x01, op.Drop}, 3, 100000)
	if err != nil {
		t.Error(err)
	}

	// However, it is an error to execute txid in a program with no
	// finalize in it.
	_, err = txvm.Validate([]byte{op.TxID, op.Drop}, 3, 100000)
	if errors.Root(err) != txvm.ErrUnfinalized {
		if err == nil {
			t.Error("got no error from pre-finalize txid, want ErrUnfinalized")
		} else {
			t.Errorf("got error %s from pre-finalize txid, want ErrUnfinalized", err)
		}
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name string
		src  string
		log  []string
		err  error
	}{
		{
			name: "simple payment",
			src:  txvmtest.SimplePayment,
			log: []string{
				"{'I', x'0000000000000000000000000000000000000000000000000000000000000000', x'7229e653bd7c21efae174d7d3e8087ea8e5e1d074adc59a1dfbd88c484ead9ea'}",
				"{'O', x'0000000000000000000000000000000000000000000000000000000000000000', x'333b102f5eebf7450cced735b1a2518f98f706b52828197d6bd70229e2e669f5'}",
				"{'F', x'0000000000000000000000000000000000000000000000000000000000000000', 3, x'a4eb3b92e93f5889d7dd213530ee968c4f602ca45b3fc34d0936417d6daa59b0'}",
			},
		},
		{
			name: "simple payment 2",
			src:  txvmtest.SimplePayment2,
			log: []string{
				"{'I', x'0000000000000000000000000000000000000000000000000000000000000000', x'7fa08e4c10e99141e90cf0c43602c6f2647ce6397f435d31f8540f0f4f5e5f3c'}",
				"{'O', x'0000000000000000000000000000000000000000000000000000000000000000', x'a7d8e822863645a86cf1b4e2cb677cead511648681ea571853e19f43d4caa1db'}",
				"{'F', x'0000000000000000000000000000000000000000000000000000000000000000', 3, x'a4eb3b92e93f5889d7dd213530ee968c4f602ca45b3fc34d0936417d6daa59b0'}",
			},
		},
		{
			name: "split payment",
			src:  txvmtest.SplitPayment,
			log: []string{
				"{'I', x'0000000000000000000000000000000000000000000000000000000000000000', x'7fa08e4c10e99141e90cf0c43602c6f2647ce6397f435d31f8540f0f4f5e5f3c'}",
				"{'O', x'0000000000000000000000000000000000000000000000000000000000000000', x'8358ba68c21f8f7a1b91a29077a517dc83b7647ca41561c63284d71d621b6037'}",
				"{'O', x'0000000000000000000000000000000000000000000000000000000000000000', x'83fbd4444cf5e461ac8acc162c30ff9ade6576099ba0852a96f8de4b8b6f04eb'}",
				"{'F', x'0000000000000000000000000000000000000000000000000000000000000000', 3, x'a4eb3b92e93f5889d7dd213530ee968c4f602ca45b3fc34d0936417d6daa59b0'}",
			},
		},
		{
			name: "merge payment",
			src:  txvmtest.MergePayment,
			log: []string{
				"{'I', x'0000000000000000000000000000000000000000000000000000000000000000', x'7fa08e4c10e99141e90cf0c43602c6f2647ce6397f435d31f8540f0f4f5e5f3c'}",
				"{'I', x'0000000000000000000000000000000000000000000000000000000000000000', x'6686c6768565b518c6dda32b761368ae119de43f9d238618e13733d0e0cc4dbe'}",
				"{'O', x'0000000000000000000000000000000000000000000000000000000000000000', x'64979f62a3496b23ca7259313e841b39189e35260c0922bfa79bbd7f0e1d6f5c'}",
				"{'F', x'0000000000000000000000000000000000000000000000000000000000000000', 3, x'a7d1c7d8eb2f10af7f647b7d95d4c83c6ef1b7e27fba289aaf4a18d5471a8748'}",
			},
		},
		{
			name: "issuance",
			src:  txvmtest.Issuance,
			log: []string{
				"{'N', x'0000000000000000000000000000000000000000000000000000000000000000', x'0000000000000000000000000000000000000000000000000000000000000000', 'blockchainidblockchainidblockcha', 10}",
				"{'R', x'0000000000000000000000000000000000000000000000000000000000000000', 0, 10}",
				"{'A', x'0000000000000000000000000000000000000000000000000000000000000000', 10, x'd8a92d34192c33551faaa500861e8bd4987847a356d54b7b2c8a6380b0bd0517', x'4f907f68e3a0f9e3094e7908af571f52dd5b6e84cc7602e501c25a2fd17f1fbb'}",
				"{'O', x'0000000000000000000000000000000000000000000000000000000000000000', x'2bc2a72073906c6745123d2a1c46c0623a2e3bf85c955abbdf1f14799985ee7a'}",
				"{'F', x'0000000000000000000000000000000000000000000000000000000000000000', 3, x'b820b13d533796a72bc4df57103cb5f85ed5d54eefd9858b7c919da47b4ba202'}",
			},
		},
		{
			name: "retirement",
			src:  txvmtest.Retirement,
			log: []string{
				"{'I', x'0000000000000000000000000000000000000000000000000000000000000000', x'7fa08e4c10e99141e90cf0c43602c6f2647ce6397f435d31f8540f0f4f5e5f3c'}",
				"{'X', x'0000000000000000000000000000000000000000000000000000000000000000', 10, x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d', x'25703d95c689c0d5619fa7c011fd5df200e0e1ab46623ebeee36c75c1ef16241'}",
				"{'F', x'0000000000000000000000000000000000000000000000000000000000000000', 3, x'a4eb3b92e93f5889d7dd213530ee968c4f602ca45b3fc34d0936417d6daa59b0'}",
			},
		},
		{
			name: "stack limit test",
			src:  txvmtest.StackLimitTest,
			err:  txvm.ErrStackRange,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prog, err := asm.Assemble(c.src)
			if err != nil {
				t.Fatal(err)
			}

			var b bytes.Buffer
			id, err := txvm.Validate(prog, 3, 100000, txvm.Trace(&b), txvm.OnLog(func(v *txvm.VM) {
				if c.log != nil {
					logItem := v.Log[len(v.Log)-1].String()
					if logItem != c.log[len(v.Log)-1] {
						t.Fatalf("Log output does not match expected output. Got: %s, Expected: %s", logItem, c.log[len(v.Log)-1])
					}
				}
			}))

			if c.err != nil {
				if err == nil {
					t.Fatalf("Expected error '%s', no error thrown", c.err)
				} else if errors.Root(err) != c.err {
					t.Fatalf("Thrown error, '%s', not equal to expected error, '%s'", errors.Root(err), c.err)
				}
			} else if err != nil {
				t.Fatalf("error %s\n%s", err, b.String())
			}

			if testing.Verbose() {
				t.Log(id)
				t.Log(b.String())
			}
		})
	}
}

func TestOptions(t *testing.T) {
	const startLimit int64 = 1000

	cases := []struct {
		name  string
		src   string
		items []string // represents the objects on top of the stack as the program progresses
		cost  int64
	}{
		{
			name: "log ops",
			src:  "10 log 0 peeklog log 'blockchainid' 8 nonce finalize",
			items: []string{
				"{'Z', 10}",
				"{'Z', 0}",
				"{'T', {'L', x'0000000000000000000000000000000000000000000000000000000000000000', 10}}",
				"{'S', 'blockchainid'}",
				"{'Z', 8}",
				"{'V', 0, x'0000000000000000000000000000000000000000000000000000000000000000', x'51a2a5efebb78d8907c97c07f807ef8b72a0482a959224cfd1555103c6195cd6'}",
			},
			cost: 178,
		},
		{
			name: "basic math",
			src:  "10 2 add 3 div 3 mod 2 gt not verify 'id' 10 nonce finalize",
			items: []string{
				"{'Z', 10}",
				"{'Z', 2}",
				"{'Z', 12}",
				"{'Z', 3}",
				"{'Z', 4}",
				"{'Z', 3}",
				"{'Z', 1}",
				"{'Z', 2}",
				"{'Z', 0}",
				"{'Z', 1}",
				"{'S', 'id'}",
				"{'Z', 10}",
				"{'V', 0, x'0000000000000000000000000000000000000000000000000000000000000000', x'87f629d8630208e6756a515c60335d75da8dce70eadcccd58a937a70b1e7eaa3'}",
			},
			cost: 162,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prog, err := asm.Assemble(c.src)
			if err != nil {
				t.Fatal(err)
			}

			var runlimit int64
			i := 0

			var b bytes.Buffer
			id, err := txvm.Validate(prog, 3, startLimit,
				txvm.BeforeStep(func(v *txvm.VM) {
					compareItems(t, v.StackItem(v.StackLen()-1).String(), c.items[i])
					i++
				}),
				txvm.AfterStep(func(v *txvm.VM) {
					compareItems(t, v.StackItem(v.StackLen()-1).String(), c.items[i])
				}),
				txvm.OnFinalize(func(v *txvm.VM) {
					compareItems(t, v.StackItem(v.StackLen()-1).String(), c.items[i])
				}),
				txvm.GetRunlimit(&runlimit),
				txvm.StopAfterFinalize,
				txvm.EnableExtension,
				txvm.Trace(&b),
			)
			if testing.Verbose() {
				t.Log(id)
				t.Log(b.String())
			}
			if err != nil {
				t.Fatal(err)
			}

			if runlimit != startLimit-c.cost {
				t.Fatalf("Runlimit from GetRunLimit does not match expected cost of program."+
					"Wanted %v - %v = %v, got %v", startLimit, c.cost, startLimit-c.cost, runlimit)
			}
		})
	}
}

func compareItems(t *testing.T, stackItem, testItem string) {
	if stackItem != testItem {
		t.Fatalf("Item on top of stack does not match expected item. Got %v, wanted %v", stackItem, testItem)
	}
}
