package asm

import (
	"math"
	"runtime"
	"testing"
	"time"

	"i10r.io/protocol/txvm"
)

func TestRunlimits(t *testing.T) {
	cases := []struct {
		name string
		pre  string
		src  string
		want int64
	}{
		{
			name: "int",
			pre:  "x'01'",
			src:  "int",
			want: 1,
		},
		{
			name: "add",
			pre:  "3 4",
			src:  "add",
			want: 1,
		},
		{
			name: "neg",
			pre:  "1",
			src:  "neg",
			want: 1,
		},
		{
			name: "mul",
			pre:  "3 4",
			src:  "mul",
			want: 1,
		},
		{
			name: "div",
			pre:  "6 2",
			src:  "div",
			want: 1,
		},
		{
			name: "mod",
			pre:  "6 4",
			src:  "mod",
			want: 1,
		},
		{
			name: "gt",
			pre:  "5 6",
			src:  "gt",
			want: 1,
		},
		{
			name: "not 0",
			pre:  "0",
			src:  "not",
			want: 1,
		},
		{
			name: "not 1",
			pre:  "1",
			src:  "not",
			want: 1,
		},
		{
			name: "and",
			pre:  "1 0",
			src:  "and",
			want: 1,
		},
		{
			name: "or",
			pre:  "1 0",
			src:  "or",
			want: 1,
		},
		{
			name: "roll",
			pre:  "4 5 6 7 2",
			src:  "roll",
			want: 3,
		},
		{
			name: "bury",
			pre:  "4 5 6 7 2",
			src:  "bury",
			want: 3,
		},
		{
			name: "reverse",
			pre:  "4 5 6 7 2",
			src:  "reverse",
			want: 3,
		},
		{
			name: "get",
			pre:  "5 put",
			src:  "get",
			want: 1,
		},
		{
			name: "put",
			pre:  "5",
			src:  "put",
			want: 1,
		},
		{
			name: "depth",
			pre:  "4 5 6 7",
			src:  "depth",
			want: 1,
		},
		{
			name: "nonce",
			pre:  "x'0000000000000000000000000000000000000000000000000000000000000000' 1",
			src:  "nonce",
			want: 140,
		},
		{
			name: "merge",
			pre:  "x'0000000000000000000000000000000000000000000000000000000000000000' 1 nonce 1 '' issue x'0000000000000000000000000000000000000000000000000000000000000000' 2 nonce 2 '' issue",
			src:  "merge",
			want: 129,
		},
		{
			name: "split",
			pre:  "x'0000000000000000000000000000000000000000000000000000000000000000' 10 nonce 1 '' issue 2",
			src:  "split",
			want: 1,
		},
		{
			name: "issue",
			pre:  "x'0000000000000000000000000000000000000000000000000000000000000000' 10 nonce 1 ''",
			src:  "issue",
			want: 135,
		},
		{
			name: "retire",
			pre:  "x'0000000000000000000000000000000000000000000000000000000000000000' 1 nonce 1 '' issue",
			src:  "retire",
			want: 7,
		},
		{
			name: "amount",
			pre:  "x'0000000000000000000000000000000000000000000000000000000000000000' 1 nonce 1 '' issue",
			src:  "amount",
			want: 1,
		},
		{
			name: "assetid",
			pre:  "x'0000000000000000000000000000000000000000000000000000000000000000' 1 nonce 1 '' issue",
			src:  "assetid",
			want: 34,
		},
		{
			name: "anchor",
			pre:  "x'0000000000000000000000000000000000000000000000000000000000000000' 1 nonce 1 '' issue",
			src:  "anchor",
			want: 34,
		},
		{
			name: "vmhash",
			pre:  "'foo' 'bar'",
			src:  "vmhash",
			want: 34,
		},
		{
			name: "sha256",
			pre:  "'foo'",
			src:  "sha256",
			want: 34,
		},
		{
			name: "sha3",
			pre:  "'foo'",
			src:  "sha3",
			want: 34,
		},
		{
			name: "checksig nonempty",
			pre:  "x'0000000000000000000000000000000000000000000000000000000000000000' x'51ac841492645979812f0ce08125259de7d7ec50a019e26e0f600b1f96748e69' x'580647ec8809101c2fb0a19b2fc1f7102c8197664bde0bc8487ed1a7814f0ac9e4196e2c37872f9223302138954d37eef3a171fcee16959098f075d0e40e2d06' 0",
			src:  "checksig",
			want: 2049,
		},
		{
			name: "checksig empty",
			pre:  "x'0000000000000000000000000000000000000000000000000000000000000000' x'51ac841492645979812f0ce08125259de7d7ec50a019e26e0f600b1f96748e69' '' 0",
			src:  "checksig",
			want: 1,
		},
		{
			name: "log",
			pre:  "'foo'",
			src:  "log",
			want: 5,
		},
		{
			name: "peeklog",
			pre:  "0",
			src:  "peeklog",
			want: 1,
		},
		{
			name: "txid",
			pre:  "x'0000000000000000000000000000000000000000000000000000000000000000' 1 nonce finalize",
			src:  "txid",
			want: 34,
		},
		{
			name: "finalize",
			pre:  "x'0000000000000000000000000000000000000000000000000000000000000000' 1 nonce",
			src:  "finalize",
			want: 6,
		},
		{
			name: "verify",
			pre:  "1",
			src:  "verify",
			want: 1,
		},
		{
			name: "jumpif",
			pre:  "1 0",
			src:  "jumpif",
			want: 1,
		},
		{
			name: "exec",
			pre:  "''",
			src:  "exec",
			want: 1,
		},
		{
			name: "call",
			pre:  "'' contract",
			src:  "call",
			want: 1,
		},
		// yield
		// wrap
		// output
		{
			name: "input",
			pre:  "{'C', '', x'2a80d7b52d4abec170d260e79083bc5c97cf17a04dcc0209ca48d02d22b28f6d'}",
			src:  "input",
			want: 133,
		},
		{
			name: "contract",
			pre:  "''",
			src:  "contract",
			want: 129,
		},
		{
			name: "seed",
			pre:  "'' contract",
			src:  "seed",
			want: 34,
		},
		{
			name: "self",
			src:  "self",
			want: 34,
		},
		{
			name: "caller",
			src:  "caller",
			want: 34,
		},
		{
			name: "contractprogram",
			src:  "contractprogram",
			want: 3,
		},
		{
			name: "timerange",
			pre:  "1 2",
			src:  "timerange",
			want: 6,
		},
		// prv
		// ext
		{
			name: "eq",
			pre:  "2 3",
			src:  "eq",
			want: 1,
		},
		{
			name: "dup",
			pre:  "5",
			src:  "dup",
			want: 1,
		},
		{
			name: "drop",
			pre:  "5 6",
			src:  "drop",
			want: 1,
		},
		{
			name: "peek",
			pre:  "4 5 6 7 0",
			src:  "peek",
			want: 1,
		},
		{
			name: "tuple",
			pre:  "'a' 'b' 'c' 3",
			src:  "tuple",
			want: 5,
		},
		{
			name: "untuple",
			pre:  "{'a', 'b', 'c'}",
			src:  "untuple",
			want: 4,
		},
		{
			name: "len",
			pre:  "{'a', 'b', 'c'}",
			src:  "len",
			want: 1,
		},
		{
			name: "field",
			pre:  "{'a', 'b', 'c'} 0",
			src:  "field",
			want: 3,
		},
		{
			name: "encode",
			pre:  "1",
			src:  "encode",
			want: 3,
		},
		{
			name: "cat",
			pre:  "'foo' 'bar'",
			src:  "cat",
			want: 8,
		},
		{
			name: "slice",
			pre:  "'foobar' 2 3",
			src:  "slice",
			want: 3,
		},
		{
			name: "bitnot",
			pre:  "x'01'",
			src:  "bitnot",
			want: 3,
		},
		{
			name: "bitand",
			pre:  "x'01' x'02'",
			src:  "bitand",
			want: 3,
		},
		{
			name: "bitor",
			pre:  "x'01' x'02'",
			src:  "bitor",
			want: 3,
		},
		{
			name: "bitxor",
			pre:  "x'01' x'02'",
			src:  "bitxor",
			want: 3,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var (
				preBytes, srcBytes []byte
				err                error
			)
			if c.pre != "" {
				preBytes, err = Assemble(c.pre)
				if err != nil {
					t.Fatal(err)
				}
			}
			srcBytes, err = Assemble(c.src)
			if err != nil {
				t.Fatal(err)
			}

			runlimit := int64(math.MaxInt64)
			if len(preBytes) > 0 {
				txvm.Validate(preBytes, 3, math.MaxInt64, txvm.GetRunlimit(&runlimit))
			}

			var runlimit2 int64
			txvm.Validate(append(preBytes, srcBytes...), 3, math.MaxInt64, txvm.GetRunlimit(&runlimit2))

			cost := int64(runlimit - runlimit2)
			if cost != c.want {
				t.Errorf("got %d, want %d", cost, c.want)
			}
		})
	}
}

type variant struct {
	name string
	pre  string // non-measured setup code
	src  string
}

// To generate the table:
// $ go test . -run=TestInstructionsRunlimit -v
// TODO(bobg): This isn't actually testing anything.
// Perhaps it should be generating its table via "go generate".
// Alternatively it can test that the runlimits are within acceptable
// bounds.
func TestInstructionsRunlimit(t *testing.T) {
	cases := []struct {
		opcode   string
		variants []variant
	}{
		{
			opcode: "mempreheat", // the first run increases mem use by 1888 bytes no matter what
			variants: []variant{
				{src: "0"},
			},
		},
		{
			opcode: "smallint",
			variants: []variant{
				{
					name: "1x10",
					src:  "1 1 1 1 1 1 1 1 1 1",
				},
				{
					name: "19x10",
					src:  "19 19 19 19 19 19 19 19 19 19",
				},
			},
		},
		{
			opcode: "int",
			variants: []variant{
				{
					name: "5x32",
					src:  "x'20' int x'20' int x'20' int x'20' int x'20' int",
				},
				{
					name: "5x-624485",
					src:  "x'9BF159' int x'9BF159' int x'9BF159' int x'9BF159' int x'9BF159' int",
				},
			},
		},
		{
			opcode: "add",
			variants: []variant{{
				name: "5x",
				src:  "1 1 add 1 add 1 add 1 add 1 add",
			}},
		},
		{
			opcode: "neg",
			variants: []variant{{
				name: "5x",
				src:  "1 neg neg neg neg neg",
			}},
		},
		{
			opcode: "mul",
			variants: []variant{{
				name: "5x",
				src:  "1 2 mul 2 mul 2 mul 2 mul 2 mul",
			}},
		},
		{
			opcode: "div",
			variants: []variant{{
				name: "5x",
				src:  "128 2 div 2 div 2 div 2 div 2 div",
			}},
		},
		{
			opcode: "mod",
			variants: []variant{{
				name: "5x",
				src:  "10 7 mod 6 mod 5 mod 4 mod 3 mod",
			}},
		},
		{
			opcode: "gt",
			variants: []variant{{
				name: "5x",
				src:  "1 1 gt 1 gt 1 gt 1 gt 1 gt",
			}},
		},
		{
			opcode: "not",
			variants: []variant{{
				name: "5x",
				src:  "0 not not not not not",
			}},
		},
		{
			opcode: "and",
			variants: []variant{{
				name: "5x",
				src:  "1 1 and 1 and 1 and 1 and 1 and",
			}},
		},
		{
			opcode: "and",
			variants: []variant{{
				name: "or",
				src:  "1 0 or",
			}},
		},
		{
			opcode: "vmhash",
			variants: []variant{{
				name: "1x",
				src:  "x'4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41' 'test' vmhash",
			}},
		},
		{
			opcode: "sha256",
			variants: []variant{{
				name: "1x",
				src:  "x'4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41' sha256",
			}},
		},
		{
			opcode: "sha3",
			variants: []variant{{
				name: "1x",
				src:  "x'4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41' sha3",
			}},
		},
		{
			opcode: "checksig",
			variants: []variant{{
				name: "1x",
				src: `x'4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41'
					  x'6f7475a7f7e486cbbd3ae1f4067426364a8c7156c129287f70b4d4ea0a27e5ed'
					  x'9adb80389eed222123717f013180e2edc2a613fb0082d3e7a254b22e640b6d91f9fe900c5a419c9c374cebb4bfae6b60b77f52345a56fa28ac692ee6b8fcb206'
					  checksig`,
			}},
		},
		{
			opcode: "roll",
			variants: []variant{
				{
					name: "1",
					pre:  "'a' 'b'",
					src:  "1 roll",
				},
				{
					name: "20",
					pre:  "'a' 'b' 'c' 'd' 'e' 'f' 'g' 'h' 'i' 'j' 'k' 'l' 'm' 'n' 'o' 'p' 'q' 'r' 's' 't' 'u' 'v' 'w' 'x' 'y' 'z'",
					src:  "20 roll",
				},
			},
		},
		{
			opcode: "bury",
			variants: []variant{
				{
					name: "1",
					pre:  "'a' 'b'",
					src:  "1 bury",
				},
				{
					name: "20",
					pre:  "'a' 'b' 'c' 'd' 'e' 'f' 'g' 'h' 'i' 'j' 'k' 'l' 'm' 'n' 'o' 'p' 'q' 'r' 's' 't' 'u' 'v' 'w' 'x' 'y' 'z'",
					src:  "20 bury",
				},
				{
					name: "40",
					pre: `'a' 'b' 'c' 'd' 'e' 'f' 'g' 'h' 'i' 'j' 'k' 'l' 'm' 'n' 'o' 'p' 'q' 'r' 's' 't' 'u' 'v' 'w' 'x' 'y' 'z'
						  'a' 'b' 'c' 'd' 'e' 'f' 'g' 'h' 'i' 'j' 'k' 'l' 'm' 'n' 'o' 'p' 'q' 'r' 's' 't' 'u' 'v' 'w' 'x' 'y' 'z'`,
					src: "40 bury",
				},
			},
		},
		{
			opcode: "reverse",
			variants: []variant{
				{
					name: "1",
					pre:  "'a' 'b'",
					src:  "1 reverse",
				},
				{
					name: "20",
					pre:  "'a' 'b' 'c' 'd' 'e' 'f' 'g' 'h' 'i' 'j' 'k' 'l' 'm' 'n' 'o' 'p' 'q' 'r' 's' 't' 'u' 'v' 'w' 'x' 'y' 'z'",
					src:  "20 reverse",
				},
				{
					name: "40",
					pre: `'a' 'b' 'c' 'd' 'e' 'f' 'g' 'h' 'i' 'j' 'k' 'l' 'm' 'n' 'o' 'p' 'q' 'r' 's' 't' 'u' 'v' 'w' 'x' 'y' 'z'
						  'a' 'b' 'c' 'd' 'e' 'f' 'g' 'h' 'i' 'j' 'k' 'l' 'm' 'n' 'o' 'p' 'q' 'r' 's' 't' 'u' 'v' 'w' 'x' 'y' 'z'`,
					src: "40 reverse",
				},
			},
		},
		{
			opcode: "get",
			variants: []variant{{
				name: "10x",
				pre:  "0 put 1 put 2 put 3 put 4 put 5 put 6 put 7 put 8 put 9 put",
				src:  "get get get get get get get get get get",
			}},
		},
		{
			opcode: "get",
			variants: []variant{{
				name: "10x",
				src:  "0 put 1 put 2 put 3 put 4 put 5 put 6 put 7 put 8 put 9 put",
			}},
		},
		{
			opcode: "depth",
			variants: []variant{{
				name: "5x",
				pre:  "0 1 2 3 4 5 6 7 8 9",
				src:  "depth depth depth depth depth",
			}},
		},
		// skip prv/ext
		{
			opcode: "verify",
			variants: []variant{{
				name: "8x",
				src:  "1 verify 1 verify 1 verify 1 verify 1 verify 1 verify 1 verify 1 verify",
			}},
		},
		{
			opcode: "jumpif",
			variants: []variant{{
				name: "1x",
				pre:  "1",
				src:  "jumpif:$end $end",
			}},
		},
		{
			opcode: "exec",
			variants: []variant{{
				name: "1x",
				pre:  "[]",
				src:  "exec",
			}},
		},
		{
			opcode: "call",
			variants: []variant{{
				name: "1x",
				pre:  "[] contract",
				src:  "call",
			}},
		},
		{
			opcode: "yield",
			variants: []variant{{
				name: "1x",
				pre:  "[[] yield] contract",
				src:  "call",
			}},
		},
		{
			opcode: "wrap",
			variants: []variant{{
				name: "1x",
				pre:  "[[] wrap] contract",
				src:  "call",
			}},
		},
		{
			opcode: "input",
			variants: []variant{
				{
					name: "tiny",
					src:  "{'C', x'4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41', []} input",
				},
				{
					name: "big",
					src: `{'C',
					'contractseed',
					  [put [txid get 0 checksig verify] yield],
					{'S', x'4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41'},
					{'V', x'3f4df5374fccec7f737ea6ba08e9c534d33246aba3a989589a57ee180cfda2cf0100000000000000000000000000000000000000000000000000000000000000', 
						  x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d0100000000000000000000000000000000000000000000000000000000000000',
						  x'69f72fb9c3d828e210e49ba0c78dc43f1f860828e210e49ba0c7de7a92bd040b'},
					{'V', x'3f4df5374fccec7f737ea6ba08e9c534d33246aba3a989589a57ee180cfda2cf0100000000000000000000000000000000000000000000000000000000000000', 
						  x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d0100000000000000000000000000000000000000000000000000000000000000',
						  x'c79ba0c78dc43f1f86082c3d8b69f72fb9028e210e410e49ba8e2de7a92bd040'},
					{'V', x'3f4df5374fccec7f737ea6ba08e9c534d33246aba3a989589a57ee180cfda2cf0100000000000000000000000000000000000000000000000000000000000000', 
						  x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d0100000000000000000000000000000000000000000000000000000000000000',
						  x'a9c79ba0c78dc43fc3d8b69f72fb9028e210e410e49ba8e2de72b1f86082d040'}
				  } input`,
				},
			},
		},
		{
			opcode: "output",
			variants: []variant{
				{
					name: "tiny",
					src:  "[] output",
				},
				{
					name: "big",
					pre:  "[[txid get 0 checksig verify] output] contract",
					src:  "call",
				},
			},
		},
		{
			opcode: "contract",
			variants: []variant{
				{
					name: "tiny",
					pre:  "",
					src:  "[] contract",
				},
				{
					name: "big",
					pre:  "",
					src:  "[x'dda403da62d29f1189507320df27ede9e0d4f21530db120f8a18eff7bb071e11' drop 1 verify jump:$end $end] contract",
				},
			},
		},
		{
			opcode: "seed",
			variants: []variant{{
				name: "5x",
				pre:  "[] contract",
				src:  "seed drop seed drop seed drop seed drop seed drop",
			}},
		},
		{
			opcode: "self",
			variants: []variant{
				{
					name: "outer 5x",
					src:  "self self self self self",
				},
				{
					name: "inner 5x",
					pre:  "[self self self self self] contract",
					src:  "call",
				},
			},
		},
		{
			opcode: "caller",
			variants: []variant{{
				name: "5x",
				pre:  "[caller caller caller caller caller] contract",
				src:  "call",
			}},
		},
		{
			opcode: "contractprogram",
			variants: []variant{
				{
					name: "outer 5x",
					src:  "contractprogram contractprogram contractprogram contractprogram contractprogram",
				},
				{
					name: "inner 5x",
					pre:  "[contractprogram contractprogram contractprogram contractprogram contractprogram] contract",
					src:  "call",
				},
			},
		},
		{
			opcode: "timerange",
			variants: []variant{{
				name: "1x",
				src:  "0 0 timerange",
			}},
		},
		{
			opcode: "log",
			variants: []variant{
				{
					name: "tiny",
					src:  "0 log",
				},
				{
					name: "big",
					src:  "{'string', 123, {{{'a', 'b', {{{'c'}}}}}}} log",
				},
			},
		},
		{
			opcode: "peeklog",
			variants: []variant{
				{
					name: "tiny 4x",
					pre:  "0 log {'string', 123, {{{'a', 'b', {{{'c'}}}}}}} log",
					src:  "0 peeklog 0 peeklog 0 peeklog 0 peeklog",
				},
				{
					name: "big 4x",
					pre:  "0 log {'string', 123, {{{'a', 'b', {{{'c'}}}}}}} log",
					src:  "1 peeklog 1 peeklog 1 peeklog 1 peeklog",
				},
			},
		},
		{
			opcode: "txid",
			variants: []variant{{
				name: "20x",
				pre: `{'C', x'cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc', [put],
					{'V', x'01000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000', 
						  x'01000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000',
						  x'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'}
				} input call get finalize`,
				src: "txid txid txid txid txid txid txid txid txid txid txid txid txid txid txid txid txid txid txid txid",
			}},
		},
		{
			opcode: "finalize",
			variants: []variant{{
				name: "1x",
				pre: `{'C', x'cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc', [put],
					{'V', x'01000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000', 
						x'01000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000',
						x'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'}
				} input call get`,
				src: "splitzero finalize", // reflects that commonly you'd have a splitzero to anchor the tx.
			}},
		},
		{
			opcode: "nonce",
			variants: []variant{{
				name: "1x",
				src:  "x'cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc' 341723392 nonce",
			}},
		},
		{
			opcode: "merge",
			variants: []variant{
				{
					name: "1x",
					pre: `{'C', x'cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc', [put put],
				{'V', x'3f4df5374fccec7f737ea6ba08e9c534d33246aba3a989589a57ee180cfda2cf0100000000000000000000000000000000000000000000000000000000000000', 
				x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d0100000000000000000000000000000000000000000000000000000000000000',
				x'1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1'},
				{'V', x'3f4df5374fccec7f737ea6ba08e9c534d33246aba3a989589a57ee180cfda2cf0100000000000000000000000000000000000000000000000000000000000000', 
				x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d0100000000000000000000000000000000000000000000000000000000000000',
				x'2aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa2'}
				} input call get get`,
					src: "merge",
				},
				{
					name: "3x",
					pre: `{'C', x'cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc', [put put put put],
					{'V', x'3f4df5374fccec7f737ea6ba08e9c534d33246aba3a989589a57ee180cfda2cf0100000000000000000000000000000000000000000000000000000000000000', 
					x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d0100000000000000000000000000000000000000000000000000000000000000',
					x'1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1'},
					{'V', x'3f4df5374fccec7f737ea6ba08e9c534d33246aba3a989589a57ee180cfda2cf0100000000000000000000000000000000000000000000000000000000000000', 
					x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d0100000000000000000000000000000000000000000000000000000000000000',
					x'2aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa2'},
					{'V', x'3f4df5374fccec7f737ea6ba08e9c534d33246aba3a989589a57ee180cfda2cf0100000000000000000000000000000000000000000000000000000000000000', 
					x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d0100000000000000000000000000000000000000000000000000000000000000',
					x'1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1'},
					{'V', x'3f4df5374fccec7f737ea6ba08e9c534d33246aba3a989589a57ee180cfda2cf0100000000000000000000000000000000000000000000000000000000000000', 
					x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d0100000000000000000000000000000000000000000000000000000000000000',
					x'2aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa2'}
					  } input call get get get get`,
					src: "merge merge merge",
				},
			},
		},
		{
			opcode: "split",
			variants: []variant{{
				name: "1x",
				pre: `{'C', x'cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc', [put],
			{'V', x'3f4df5374fccec7f737ea6ba08e9c534d33246aba3a989589a57ee180cfda2cf0100000000000000000000000000000000000000000000000000000000000000', 
			x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d0100000000000000000000000000000000000000000000000000000000000000',
			x'1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1'}
			  } input call get`,
				src: "x'3f4df5374fccec7f737ea6ba08e9c534d33246aba3a989589a57ee180cfda2cf0100000000000000000000000000000000000000000000000000000000000000' split",
			}},
		},
		{
			opcode: "splitzero",
			variants: []variant{{
				name: "1x",
				pre: `{'C', x'cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc', [put],
			{'V', x'3f4df5374fccec7f737ea6ba08e9c534d33246aba3a989589a57ee180cfda2cf0100000000000000000000000000000000000000000000000000000000000000', 
			x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d0100000000000000000000000000000000000000000000000000000000000000',
			x'1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1'}
			  } input call get`,
				src: "splitzero",
			}},
		},
		{
			opcode: "issue",
			variants: []variant{{
				name: "1x",
				pre:  `'' 1 ''`,
				src:  "issue",
			}},
		},
		{
			opcode: "retire",
			variants: []variant{{
				name: "1x",
				pre: `{'C', x'cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc', [put],
				{'V', x'3f4df5374fccec7f737ea6ba08e9c534d33246aba3a989589a57ee180cfda2cf0100000000000000000000000000000000000000000000000000000000000000',
					x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d0100000000000000000000000000000000000000000000000000000000000000',
					x'1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1'}
				} input call get`,
				src: "retire",
			}},
		},
		{
			opcode: "anchor",
			variants: []variant{{
				name: "5x",
				pre: `{'C', x'cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc', [put],
					{'V', x'3f4df5374fccec7f737ea6ba08e9c534d33246aba3a989589a57ee180cfda2cf0100000000000000000000000000000000000000000000000000000000000000', 
					x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d0100000000000000000000000000000000000000000000000000000000000000',
					x'1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1'}
					  } input call get`,
				src: "anchor drop anchor drop anchor drop anchor drop anchor",
			}},
		},
		{
			opcode: "eq",
			variants: []variant{
				{
					name: "int",
					src:  "1 1 eq",
				},
				{
					name: "str",
					src:  "'abc' 'xyz' eq",
				},
				{
					name: "tpl",
					src:  "{1,2,3} {4,5,6} eq",
				},
			},
		},
		// TBD: dup
		// TBD: drop
		// TBD: peek
		// TBD: tuple
		// TBD: untuple
		// TBD: len
		// TBD: field
		{
			opcode: "encode",
			variants: []variant{
				{
					name: "string",
					pre:  "'abc'",
					src:  "encode",
				},
				{
					name: "smallint",
					pre:  "1",
					src:  "encode",
				},
				{
					name: "int",
					pre:  "1000",
					src:  "encode",
				},
				{
					name: "tuple",
					pre:  "{'a', 2, x'030303'}",
					src:  "encode",
				},
			},
		},
		// TBD: cat
		// TBD: slice
		// TBD: bitnot
		// TBD: bitand
		// TBD: bitor
		// TBD: bitxor
		{
			opcode: "pushdata",
			variants: []variant{
				{
					name: "0b",
					src:  "x''",
				},
				{
					name: "32b",
					src:  "x'dda403da62d29f1189507320df27ede9e0d4f21530db120f8a18eff7bb071e11'",
				},
				{
					name: "64b",
					src:  "x'dda403da62d29f1189507320df27ede9e0d4f21530db120f8a18eff7bb071e110100000000000000000000000000000000000000000000000000000000000000'",
				},
			},
		},
	}
	t.Logf("\t%-24s %-10s %12s %12s %20s %20s %20s %20s %s\n",
		"Instruction",
		"Variant",
		"Bytes",
		"Nanoseconds",
		"Normalized time",
		"Normalized rent",
		"Runlimit consumed",
		"Cost deviation",
		"",
	)
	for _, c := range cases {
		var (
			maxdeviation        float64
			maxdeviationvariant variant
			maxavgmemusage      float64
			maxavgduration      float64
			maxnormalizedtime   float64
			maxnormalizedrent   float64
			maxruncost          int64
		)
		maxdeviation = 1.0
		maxdeviationvariant = c.variants[0]

		for _, v := range c.variants {
			// We run the example 1 time and measure the consumed runlimit.
			// If the consumed runlimit is X times smaller than 10K,
			// we measure the time over max(10,X) iterations.
			var err error
			var precode []byte
			var srccode []byte
			if len(v.pre) > 0 {
				precode, err = Assemble(v.pre)
				if err != nil {
					t.Fatal(err)
				}
			}
			srccode, err = Assemble(v.src)
			if err != nil {
				t.Fatal(err)
			}

			measurement := func(measuremem bool) (int64, int64, int64) {
				// We use Validate() function twice:
				// 1) with the setup only, without the measured code
				// 2) with the setup + measured code.
				// Then we subtract (1) from (2) to get a somewhat clean measurement.

				t1, m1, r1 := measureProgram(precode, measuremem)
				t2, m2, r2 := measureProgram(append(precode, srccode...), measuremem)

				if m2 < m1 { // use maximum used memory
					m2 = m1
				}
				return t2 - t1, m2 - m1, r2 - r1
			}

			_, memusage, runcost := measurement(true)
			budget := int64(20000)
			n := (budget / (runcost + 1)) + 20
			totalduration := int64(0)
			for i := int64(0); i < n; i++ {
				duration, _, _ := measurement(false)
				totalduration += int64(duration)
			}
			avgduration := float64(totalduration) / float64(n)
			avgmemusage := float64(memusage)
			avgrent := avgduration * avgmemusage
			normalizedrent := avgrent / 3700.0
			normalizedtime := avgduration / 100.0
			deviation := normalizedtime / float64(runcost)

			if (deviation < 1.0 && deviation < maxdeviation) ||
				(deviation > 1.0 && deviation > maxdeviation) {
				maxdeviation = deviation
				maxdeviationvariant = v
				maxavgmemusage = avgmemusage
				maxavgduration = avgduration
				maxnormalizedtime = normalizedtime
				maxnormalizedrent = normalizedrent
				maxruncost = runcost
			}
		}
		warning := ""
		overchargeallowed := 10.0
		underchargeallowed := 10.0
		if maxdeviation >= underchargeallowed || maxdeviation <= 1/overchargeallowed {
			warning = "⚠️"
		}
		if c.opcode != "mempreheat" {
			t.Logf("\t%-24s %-10s %12d %12d %20f %20f %20d %20fx %s\n",
				c.opcode,
				maxdeviationvariant.name,
				int64(maxavgmemusage),
				int64(maxavgduration),
				maxnormalizedtime,
				maxnormalizedrent,
				maxruncost,
				maxdeviation,
				warning,
			)
		}
	}
}

func measureProgram(prog []byte, measuremem bool) (int64, int64, int64) { // time, memory, runlimit
	memstats := new(runtime.MemStats)
	if measuremem {
		runtime.ReadMemStats(memstats)
	}
	start := time.Now()
	r := int64(1000000)

	var runlimitLeft int64
	txvm.Validate(prog, 3, r, txvm.GetRunlimit(&runlimitLeft))
	r = r - runlimitLeft
	t := time.Now()
	elapsed := t.Sub(start)
	allocs := memstats.TotalAlloc
	if measuremem {
		runtime.ReadMemStats(memstats)
	}
	allocs = memstats.TotalAlloc - allocs
	return int64(elapsed), int64(allocs), r
}
