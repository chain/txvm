package txvm

import (
	"testing"

	"i10r.io/protocol/txvm/op"
	"i10r.io/testutil"
)

func TestIntrospection(t *testing.T) {
	cases := []struct {
		name     string
		version  int64
		runlimit int64
		seed     []byte
		opcode   byte
		stack    stack
	}{
		{
			name:     "basic",
			version:  3,
			runlimit: 1000,
			seed:     make([]byte, 32),
			opcode:   op.Add,
			stack:    stack{Int(2), Int(10)},
		},
		{
			name:     "old",
			version:  2,
			runlimit: 50,
			seed:     []byte{0, 13, 0, 1, 0, 1, 15},
			opcode:   op.Put,
			stack:    stack{Int(2), Int(10)},
		},
		{
			name:     "new",
			version:  100,
			runlimit: 500000000,
			seed:     Bytes("some really long seed string"),
			opcode:   op.Mod,
			stack:    stack{Int(2), Int(10)},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := &VM{
				txVersion: c.version,
				runlimit:  c.runlimit,
				contract: &contract{
					seed:     c.seed,
					program:  []byte{c.opcode},
					stack:    c.stack,
					typecode: ContractCode,
				},
				argstack: stack{},
				caller:   emptySeed,
				opcode:   c.opcode,
			}

			if v.Runlimit() != c.runlimit {
				t.Fatalf("Runlimit does not match expected. Got %v, wanted %v", v.Runlimit(), c.runlimit)
			}

			if !testutil.DeepEqual(v.Seed(), c.seed) {
				t.Fatalf("Seed does not match expected value. Got %v, wanted %v", v.Seed(), c.seed)
			}

			if v.Version() != c.version {
				t.Fatalf("Version does not match expected version. Got %v, wanted %v", v.Version(), c.version)
			}

			if v.StackLen() != len(c.stack) {
				t.Fatalf("StackLen does not match expected. Got %v, wanted %v", v.StackLen(), len(c.stack))
			} else {
				for i := range c.stack {
					item, err := uninspect(v.StackItem(i).(Tuple))
					if err != nil {
						t.Fatalf("Stack item at location %v is not a Data type and can't be uninspected. Item: %v", i, v.StackItem(i))
					}
					if item != c.stack[i] {
						t.Fatalf("Stack item at location %v does not match expected. Got %v, wanted %v", i, v.StackItem(i), c.stack[i].inspect())
					}
				}
			}

			if v.OpCode() != c.opcode {
				t.Fatalf("OpCode does not match expected. Got %v, wanted %v", v.OpCode(), c.opcode)
			}
		})
	}
}
