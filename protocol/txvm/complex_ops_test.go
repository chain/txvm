package txvm

import (
	"math"
	"testing"

	"i10r.io/errors"
	"i10r.io/protocol/txvm/op"
)

func TestComplexOpcodes(t *testing.T) {
	var emptySeed = make([]byte, 32)

	cases := []struct {
		name   string
		preCon stack // items on the vm stack before op is run
		preArg stack // items on the arg stack

		prog []byte // code to be run

		postCon stack // contract stack after execution, from bottom -> top of stack
		postArg stack
		log     []string // storing as a []string for ease of populating test cases, actual log is []Tuple
		wanterr error    // check only if postCon, postArg, log are nil
	}{
		/* STACK INSTRUCTIONS */
		{
			name:    "put bytes",
			preCon:  stack{Bytes("hello")},
			preArg:  stack{Bytes("bye")},
			prog:    []byte{op.Put},
			postArg: stack{Bytes("bye"), Bytes("hello")},
		},
		{
			name:    "put ints",
			preCon:  stack{Int(2)},
			preArg:  stack{Int(3)},
			prog:    []byte{op.Put},
			postArg: stack{Int(3), Int(2)},
		},
		{
			name:    "put tuples",
			preCon:  stack{Tuple{Int(2), Bytes("tew")}},
			preArg:  stack{Tuple{Bytes("i am argstack")}},
			prog:    []byte{op.Put},
			postArg: stack{Tuple{Bytes("i am argstack")}, Tuple{Int(2), Bytes("tew")}},
		},
		{
			name:    "put fail",
			prog:    []byte{op.Put},
			wanterr: ErrUnderflow,
		},
		{
			name:    "get bytes",
			preCon:  stack{Bytes("hello")},
			preArg:  stack{Bytes("bye")},
			prog:    []byte{op.Get},
			postCon: stack{Bytes("hello"), Bytes("bye")},
		},
		{
			name:    "get ints",
			preCon:  stack{Int(2)},
			preArg:  stack{Int(3)},
			prog:    []byte{op.Get},
			postCon: stack{Int(2), Int(3)},
		},
		{
			name:    "get tuples",
			preCon:  stack{Tuple{Int(2), Bytes("tew")}},
			preArg:  stack{Tuple{Bytes("i am argstack")}},
			prog:    []byte{op.Get},
			postCon: stack{Tuple{Int(2), Bytes("tew")}, Tuple{Bytes("i am argstack")}},
		},
		{
			name:    "get fail",
			prog:    []byte{op.Get},
			wanterr: ErrUnderflow,
		},
		{
			name:    "depth bytes",
			preCon:  stack{Bytes("hello")},
			preArg:  stack{Bytes("count me!")},
			prog:    []byte{op.Depth},
			postCon: stack{Bytes("hello"), Int(1)},
			postArg: stack{Bytes("count me!")},
		},
		/* VALUE INSTRUCTIONS */
		{
			name:   "nonce",
			preCon: stack{Bytes("blockid"), Int(20)},
			prog:   []byte{op.Nonce},
			postCon: stack{&value{
				amount:  0,
				assetID: emptySeed,
				anchor:  mustDecodeHex("d666fd3aa6411977ed1dcf83c30ada7198a491aab6e6442800d798548f28b336"),
			}},
			log: []string{
				"{'N', x'0000000000000000000000000000000000000000000000000000000000000000', x'0000000000000000000000000000000000000000000000000000000000000000', 'blockid', 20}",
				"{'R', x'0000000000000000000000000000000000000000000000000000000000000000', 0, 20}",
			},
		},
		{
			name: "nonce fail: vm finalized",
			preCon: stack{Bytes("blockid"), Int(20),
				&value{
					amount:  0,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("e967354f50dd5d786cb9e11c3e362d6a2031b3d28c2db7eb831225002b2ed9d9"),
				},
			},
			prog:    []byte{op.Finalize, op.Nonce},
			wanterr: ErrFinalized,
		},
		{
			name: "merge",
			preCon: stack{
				&value{
					amount:  10,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("e967354f50dd5d786cb9e11c3e362d6a2031b3d28c2db7eb831225002b2ed9d9"),
				},
				&value{
					amount:  20,
					assetID: Bytes("apples"),
					anchor:  emptySeed,
				},
			},
			prog: []byte{op.Merge},
			postCon: stack{
				&value{
					amount:  30,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("bf7ab7c2da60a6aff4a11acb5aff9006c8bc640d88d23cb0287235dffc2aae23"),
				},
			},
		},
		{
			name: "merge fail comparing apples and oranges",
			preCon: stack{
				&value{
					amount:  10,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("e967354f50dd5d786cb9e11c3e362d6a2031b3d28c2db7eb831225002b2ed9d9"),
				},
				&value{
					amount:  20,
					assetID: Bytes("oranges"),
					anchor:  mustDecodeHex("bf7ab7c2da60a6aff4a11acb5aff9006c8bc640d88d23cb0287235dffc2aae23"),
				},
			},
			prog:    []byte{op.Merge},
			wanterr: ErrMergeAsset,
		},
		{
			name: "merge fail int overflow",
			preCon: stack{
				&value{
					amount:  math.MaxInt64 - 100,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("e967354f50dd5d786cb9e11c3e362d6a2031b3d28c2db7eb831225002b2ed9d9"),
				},
				&value{
					amount:  999,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("bf7ab7c2da60a6aff4a11acb5aff9006c8bc640d88d23cb0287235dffc2aae23"),
				},
			},
			prog:    []byte{op.Merge},
			wanterr: ErrIntOverflow,
		},
		{
			name: "split int",
			preCon: stack{
				&value{
					amount:  10,
					assetID: emptySeed,
					anchor:  mustDecodeHex("e967354f50dd5d786cb9e11c3e362d6a2031b3d28c2db7eb831225002b2ed9d9"),
				},
				Int(7),
			},
			prog: []byte{op.Split},
			postCon: stack{
				&value{
					amount:  3,
					assetID: emptySeed,
					anchor:  mustDecodeHex("8b88fe2be0b33a4f5df94190847eddb800114710bcd1c180d622ca231d74b965"),
				},
				&value{
					amount:  7,
					assetID: emptySeed,
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
		},
		{
			name: "split zero",
			preCon: stack{
				&value{
					amount:  10,
					assetID: emptySeed,
					anchor:  mustDecodeHex("e967354f50dd5d786cb9e11c3e362d6a2031b3d28c2db7eb831225002b2ed9d9"),
				},
				Int(0),
			},
			prog: []byte{op.Split},
			postCon: stack{
				&value{
					amount:  10,
					assetID: emptySeed,
					anchor:  mustDecodeHex("8b88fe2be0b33a4f5df94190847eddb800114710bcd1c180d622ca231d74b965"),
				},
				&value{
					amount:  0,
					assetID: emptySeed,
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
		},
		{
			name: "split fail amount > a.amount",
			preCon: stack{
				&value{
					amount:  10,
					assetID: emptySeed,
					anchor:  mustDecodeHex("e967354f50dd5d786cb9e11c3e362d6a2031b3d28c2db7eb831225002b2ed9d9"),
				},
				Int(12),
			},
			prog:    []byte{op.Split},
			wanterr: ErrSplit,
		},
		{
			name: "split fail amount neg",
			preCon: stack{
				&value{
					amount:  10,
					assetID: emptySeed,
					anchor:  mustDecodeHex("e967354f50dd5d786cb9e11c3e362d6a2031b3d28c2db7eb831225002b2ed9d9"),
				},
				Int(-1),
			},
			prog:    []byte{op.Split},
			wanterr: ErrNegAmount,
		},
		{
			name: "issue",
			preCon: stack{
				&value{
					amount:  0,
					assetID: emptySeed,
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
				Int(100),
				Bytes("assettag"),
			},
			prog: []byte{op.Issue},
			postCon: stack{
				&value{
					amount:  100,
					assetID: mustDecodeHex("abc23cda44828e2db493c6116d85b729c192893fb7083c6609cc6f0932403f15"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			log: []string{"{'A', x'0000000000000000000000000000000000000000000000000000000000000000', 100, x'abc23cda44828e2db493c6116d85b729c192893fb7083c6609cc6f0932403f15', x'864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88'}"},
		},
		{
			name: "issue fail amount neg",
			preCon: stack{
				&value{
					amount:  0,
					assetID: emptySeed,
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
				Int(-2),
				Bytes("assettag"),
			},
			prog:    []byte{op.Issue},
			wanterr: ErrNegAmount,
		},
		{
			name: "issue fail avalue not zero-amount",
			preCon: stack{
				&value{
					amount:  99,
					assetID: emptySeed,
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
				Int(99),
				Bytes("assettag"),
			},
			prog:    []byte{op.Issue},
			wanterr: ErrAnchorVal,
		},
		{
			name: "issue fail finalized",
			preCon: stack{
				&value{
					amount:  0,
					assetID: emptySeed,
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
				Int(99),
				Bytes("assettag"),
				&value{
					amount:  0,
					assetID: emptySeed,
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			prog:    []byte{op.Finalize, op.Issue},
			wanterr: ErrFinalized,
		},
		{
			name: "retire",
			preCon: stack{
				&value{
					amount:  100,
					assetID: emptySeed,
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			prog: []byte{op.Retire},
			log:  []string{"{'X', x'0000000000000000000000000000000000000000000000000000000000000000', 100, x'0000000000000000000000000000000000000000000000000000000000000000', x'864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88'}"},
		},
		{
			name: "retire fail finalized",
			preCon: stack{
				&value{
					amount:  100,
					assetID: emptySeed,
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
				&value{
					amount:  0,
					assetID: emptySeed,
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			prog:    []byte{op.Finalize, op.Retire},
			wanterr: ErrFinalized,
		},
		{
			name: "amount",
			preCon: stack{
				&value{
					amount:  100,
					assetID: emptySeed,
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			prog: []byte{op.Amount},
			postCon: stack{
				&value{
					amount:  100,
					assetID: emptySeed,
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
				Int(100),
			},
		},
		{
			name:    "amount fail not value",
			preCon:  stack{Int(100)},
			prog:    []byte{op.Amount},
			wanterr: ErrType,
		},
		{
			name: "assetid",
			preCon: stack{
				&value{
					amount:  100,
					assetID: mustDecodeHex("abc23cda44828e2db493c6116d85b729c192893fb7083c6609cc6f0932403f15"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			prog: []byte{op.AssetID},
			postCon: stack{
				&value{
					amount:  100,
					assetID: mustDecodeHex("abc23cda44828e2db493c6116d85b729c192893fb7083c6609cc6f0932403f15"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
				Bytes(mustDecodeHex("abc23cda44828e2db493c6116d85b729c192893fb7083c6609cc6f0932403f15")),
			},
		},
		{
			name:    "assetid fail not value",
			preCon:  stack{Int(100)},
			prog:    []byte{op.AssetID},
			wanterr: ErrType,
		},
		{
			name: "anchor",
			preCon: stack{
				&value{
					amount:  100,
					assetID: emptySeed,
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			prog: []byte{op.Anchor},
			postCon: stack{
				&value{
					amount:  100,
					assetID: emptySeed,
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
				Bytes(mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88")),
			},
		},
		{
			name:    "anchor fail not value",
			preCon:  stack{Int(100)},
			prog:    []byte{op.Anchor},
			wanterr: ErrType,
		},
		/* CONTROL FLOW INSTRUCTIONS */
		{
			name:    "verify false",
			preCon:  stack{Int(0)},
			prog:    []byte{op.Verify},
			wanterr: ErrVerifyFail,
		},
		{
			name:   "verify int",
			preCon: stack{Int(1)},
			prog:   []byte{op.Verify},
		},
		{
			name:   "verify tuple",
			preCon: stack{Tuple{Int(20), Bytes("words")}},
			prog:   []byte{op.Verify},
		},
		{
			name:   "jumpif false",
			preCon: stack{Int(0), Int(25)},
			prog:   []byte{op.JumpIf},
		},
		{
			name:    "jumpif",
			preCon:  stack{Int(1), Int(0)},
			prog:    []byte{op.JumpIf},
			postCon: stack{},
			// TODO: check program counter with int > 0
		},
		{
			name:    "jumpif fail int overflow",
			preCon:  stack{Int(1), Int(math.MaxInt64)},
			prog:    []byte{op.JumpIf},
			wanterr: ErrIntOverflow,
		},
		{
			name:    "jumpif fail invalid destination",
			preCon:  stack{Int(1), Int(100)},
			prog:    []byte{op.JumpIf},
			wanterr: ErrJump,
		},
		// TODO: write more tests with programs, using more realistic programs (eg "pubkey match")
		{
			name:    "exec",
			preCon:  stack{Bytes([]byte{0x0c, 0x0a, op.Mod, op.Put})},
			prog:    []byte{op.Exec},
			postArg: stack{Int(2)},
		},
		{
			name: "call",
			preCon: stack{
				&contract{
					seed:     mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
					typecode: ContractCode,
					program:  []byte{op.Mod, op.Put},
					stack:    stack{Int(12), Int(10)},
				},
			},
			prog:    []byte{op.Call},
			postArg: stack{Int(2)},
		},
		{
			name: "call fail items on stack",
			preCon: stack{
				&contract{
					seed:     mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
					typecode: ContractCode,
					program:  []byte{op.Mod},
					stack:    stack{Int(12), Int(10)},
				},
			},
			prog:    []byte{op.Call},
			wanterr: ErrNonEmpty,
		},
		{
			name:   "yield",
			preCon: stack{Bytes([]byte{0x0c, 0x0a, op.Mod, op.Put})},
			prog:   []byte{op.Yield},
			postArg: stack{
				&contract{
					typecode: ContractCode,
					seed:     emptySeed,
					program:  []byte{0x0c, 0x0a, op.Mod, op.Put},
				},
			},
		},
		{
			name:   "wrap",
			preCon: stack{Bytes([]byte{0x0c, 0x0a, op.Mod, op.Put})},
			prog:   []byte{op.Wrap},
			postArg: stack{
				&contract{
					typecode: WrappedContractCode,
					seed:     emptySeed,
					program:  []byte{0x0c, 0x0a, op.Mod, op.Put},
				},
			},
		},
		{
			name:    "wrap with portable item on stack",
			preCon:  stack{Int(10), Bytes([]byte{0x0c, 0x0a, op.Mod, op.Put})},
			prog:    []byte{op.Wrap},
			postCon: stack{Int(10)},
			postArg: stack{
				&contract{
					typecode: WrappedContractCode,
					seed:     emptySeed,
					program:  []byte{0x0c, 0x0a, op.Mod, op.Put},
					stack:    stack{Int(10)},
				},
			},
		},
		{
			name: "wrap with non-portable item on stack",
			preCon: stack{
				&contract{
					typecode: ContractCode,
					seed:     mustDecodeHex("6fb12dcb408e113c56abdbdf3b42f39fa1c58fc812db45c2b32032b7437c00cd"),
					program:  []byte{0x0c, 0x0a, op.Mod, op.Put},
				},
				Bytes([]byte{0x0c, 0x0a, op.Mod, op.Put}),
			},
			prog:    []byte{op.Wrap},
			wanterr: ErrUnportable,
		},
		{
			name:   "input",
			preCon: stack{Tuple{Bytes("C"), Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")), Bytes("")}},
			prog:   []byte{op.Input},
			postCon: stack{
				&contract{
					typecode: ContractCode,
					seed:     mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41"),
				},
			},
			log: []string{"{'I', x'0000000000000000000000000000000000000000000000000000000000000000', x'5ef681b324b71cf473df29f9c761220863d2d1f199ab9150fbcfd836cc265741'}"},
		},
		{
			name: "input with stack items",
			preCon: stack{Tuple{
				Bytes("C"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Tuple{Bytes("Z"), Int(7)},
				Tuple{Bytes("S"), Bytes("stack item")},
				Tuple{Bytes("T"), Tuple{Int(7)}},
				(&value{
					amount:  0,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				}).inspect(),
				(&contract{
					typecode: ContractCode,
					seed:     mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41"),
					stack:    stack{Int(7), Bytes("stack item"), Tuple{Int(7)}},
				}).inspect(),
			}},
			prog: []byte{op.Input},
			postCon: stack{
				&contract{
					typecode: ContractCode,
					seed:     mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41"),
					stack: stack{
						Int(7),
						Bytes("stack item"),
						Tuple{Int(7)},
						&value{
							amount:  0,
							assetID: Bytes("apples"),
							anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
						},
						&contract{
							typecode: ContractCode,
							seed:     mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41"),
							stack:    stack{Int(7), Bytes("stack item"), Tuple{Int(7)}},
						},
					},
				},
			},
			log: []string{"{'I', x'0000000000000000000000000000000000000000000000000000000000000000', x'00fc1817d4da95ad9a7ba7f064e962a6fc21dd69677c38a7c4ed796ea5c8d65d'}"},
		},
		{
			name: "input fail empty stack items",
			preCon: stack{Tuple{
				Bytes("C"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Tuple{},
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail contract fields",
			preCon: stack{Tuple{
				Bytes("C"),
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail contract type",
			preCon: stack{Tuple{
				Bytes("EE"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail contract program",
			preCon: stack{Tuple{
				Bytes("C"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Tuple{
					Bytes("C"),
					Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
					Tuple{Int(6)},
				},
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail contract seed",
			preCon: stack{Tuple{
				Bytes("C"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Tuple{
					Bytes("C"),
					Int(9), // not a valid seed
					Bytes("contract program"),
				},
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail contract stack",
			preCon: stack{Tuple{
				Bytes("C"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Bytes("not a valid stack"),
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail value fields",
			preCon: stack{Tuple{
				Bytes("C"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Tuple{
					Bytes("V"),
				},
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail value typecode",
			preCon: stack{Tuple{
				Bytes("C"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Tuple{
					Bytes("E"),
					Int(7),
					Bytes("asset id"),
					Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				},
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail value value",
			preCon: stack{Tuple{
				Bytes("C"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Tuple{
					Bytes("V"),
					Bytes("invalid value"),
					Bytes("asset id"),
					Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				},
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail value asset",
			preCon: stack{Tuple{
				Bytes("C"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Tuple{
					Bytes("V"),
					Int(7),
					Int(10), // invalid asset ID
					Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				},
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail value anchor",
			preCon: stack{Tuple{
				Bytes("C"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Tuple{
					Bytes("V"),
					Int(7),
					Bytes("asset id"),
					Int(10), // invalid anchor
				},
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail tuple typecode",
			preCon: stack{Tuple{
				Bytes("C"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Tuple{Bytes("not typecode"), Bytes("some value")},
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail tuple not Int",
			preCon: stack{Tuple{
				Bytes("C"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Tuple{Bytes("Z"), Bytes("not int")},
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail tuple not Bytes",
			preCon: stack{Tuple{
				Bytes("C"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Tuple{Bytes("S"), Int(10)},
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail tuple not Tuple",
			preCon: stack{Tuple{
				Bytes("C"),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Tuple{Bytes("T"), Bytes("not tuple")},
			}},
			prog:    []byte{op.Input},
			wanterr: ErrFields,
		},
		{
			name: "input fail finalized",
			preCon: stack{Tuple{Bytes("C"), Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")), Bytes("")},
				&value{
					amount:  0,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			prog:    []byte{op.Finalize, op.Input},
			wanterr: ErrFinalized,
		},
		{
			name:   "output",
			preCon: stack{Bytes([]byte{0x0c, 0x0a, op.Mod, op.Put})},
			prog:   []byte{op.Output},
			log:    []string{"{'O', x'0000000000000000000000000000000000000000000000000000000000000000', x'f84f5d66e33e7b0384b9ec076a99bb1d96d5c89366b8c3476ef423b3c0a2b89f'}"},
		},
		{
			name:    "output with portable item on stack",
			preCon:  stack{Int(10), Bytes([]byte{0x0c, 0x0a, op.Mod, op.Put})},
			prog:    []byte{op.Output},
			postCon: stack{Int(10)},
			log:     []string{"{'O', x'0000000000000000000000000000000000000000000000000000000000000000', x'ac3c8c8bb09cca294b459194d1040d48b136eec2f98f0c85b735d25fa6c2a2f3'}"},
		},
		{
			name:    "output with tuple on stack",
			preCon:  stack{Tuple{Int(10), Bytes("hi")}, Bytes([]byte{0x0c, 0x0a, op.Mod, op.Put})},
			prog:    []byte{op.Output},
			postCon: stack{Tuple{Int(10), Bytes("hi")}},
			log:     []string{"{'O', x'0000000000000000000000000000000000000000000000000000000000000000', x'f04773b57c39c413c535abe77930a7a6021e4e83c225cfe1471425e8ee2bab55'}"},
		},
		{
			name: "output with non-portable item on stack",
			preCon: stack{
				&contract{
					typecode: ContractCode,
					seed:     mustDecodeHex("6fb12dcb408e113c56abdbdf3b42f39fa1c58fc812db45c2b32032b7437c00cd"),
					program:  []byte{0x0c, 0x0a, op.Mod, op.Put},
				},
				Bytes([]byte{0x0c, 0x0a, op.Mod, op.Put}),
			},
			prog:    []byte{op.Output},
			wanterr: ErrUnportable,
		},
		{
			name: "output fail finalized",
			preCon: stack{Bytes([]byte{0x0c, 0x0a, op.Mod, op.Put}),
				&value{
					amount:  0,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			prog:    []byte{op.Finalize, op.Output},
			wanterr: ErrFinalized,
		},
		{
			name:   "contract",
			preCon: stack{Bytes([]byte{0x0c, 0x0a, op.Mod, op.Put})},
			prog:   []byte{op.Contract},
			postCon: stack{
				&contract{
					typecode: ContractCode,
					seed:     mustDecodeHex("6fb12dcb408e113c56abdbdf3b42f39fa1c58fc812db45c2b32032b7437c00cd"),
					program:  []byte{0x0c, 0x0a, op.Mod, op.Put},
				},
			},
		},
		{
			name: "seed wrapped contract",
			preCon: stack{
				&contract{
					typecode: WrappedContractCode,
					seed:     mustDecodeHex("6fb12dcb408e113c56abdbdf3b42f39fa1c58fc812db45c2b32032b7437c00cd"),
					program:  []byte{0x0c, 0x0a, op.Mod, op.Put},
				},
			},
			prog: []byte{op.Seed},
			postCon: stack{
				&contract{
					typecode: WrappedContractCode,
					seed:     mustDecodeHex("6fb12dcb408e113c56abdbdf3b42f39fa1c58fc812db45c2b32032b7437c00cd"),
					program:  []byte{0x0c, 0x0a, op.Mod, op.Put},
				},
				Bytes(mustDecodeHex("6fb12dcb408e113c56abdbdf3b42f39fa1c58fc812db45c2b32032b7437c00cd"))},
		},
		{
			name: "seed non-wrapped contract",
			preCon: stack{
				&contract{
					typecode: ContractCode,
					seed:     mustDecodeHex("6fb12dcb408e113c56abdbdf3b42f39fa1c58fc812db45c2b32032b7437c00cd"),
					program:  []byte{0x0c, 0x0a, op.Mod, op.Put},
				},
			},
			prog: []byte{op.Seed},
			postCon: stack{
				&contract{
					typecode: ContractCode,
					seed:     mustDecodeHex("6fb12dcb408e113c56abdbdf3b42f39fa1c58fc812db45c2b32032b7437c00cd"),
					program:  []byte{0x0c, 0x0a, op.Mod, op.Put},
				},
				Bytes(mustDecodeHex("6fb12dcb408e113c56abdbdf3b42f39fa1c58fc812db45c2b32032b7437c00cd"))},
		},
		{
			name:    "self",
			prog:    []byte{op.Self},
			postCon: stack{Bytes(emptySeed)},
		},
		{
			name:    "self in sub-program",
			preCon:  stack{Bytes([]byte{op.Self, op.Put})},
			prog:    []byte{op.Contract, op.Call},
			postArg: stack{Bytes(mustDecodeHex("3b9ae08a95df6a1f0cc481766416bb06d5b5bc879388b6e7e0d049cb30834fe0"))},
		},
		{
			name:    "caller",
			prog:    []byte{op.Caller},
			postCon: stack{Bytes(emptySeed)},
		},
		{
			name:    "caller in sub-program",
			preCon:  stack{Bytes([]byte{op.Caller, op.Put})},
			prog:    []byte{op.Contract, op.Call},
			postArg: stack{Bytes(emptySeed)},
		},
		{
			name:    "contractprogram",
			prog:    []byte{op.ContractProgram},
			postCon: stack{Bytes([]byte{op.ContractProgram})},
		},
		{
			name:   "timerange",
			preCon: stack{Int(5), Int(27)},
			prog:   []byte{op.TimeRange},
			log:    []string{"{'R', x'0000000000000000000000000000000000000000000000000000000000000000', 5, 27}"},
		},
		{ // this transaction will be impossible to apply to any state
			name:   "timerange min > max",
			preCon: stack{Int(7), Int(2)},
			prog:   []byte{op.TimeRange},
			log:    []string{"{'R', x'0000000000000000000000000000000000000000000000000000000000000000', 7, 2}"},
		},
		{
			name: "timerange fail finalized",
			preCon: stack{Int(5), Int(27),
				&value{
					amount:  0,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			prog:    []byte{op.Finalize, op.TimeRange},
			wanterr: ErrFinalized,
		},
		{
			name:   "log",
			preCon: stack{Int(27)},
			prog:   []byte{op.Log},
			log:    []string{"{'L', x'0000000000000000000000000000000000000000000000000000000000000000', 27}"},
		},
		{
			name: "log fail finalize",
			preCon: stack{Int(27),
				&value{
					amount:  0,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			prog:    []byte{op.Finalize, op.Log},
			wanterr: ErrFinalized,
		},
		{
			name:    "peeklog",
			preCon:  stack{Int(1), Int(7), Bytes("hello")},
			prog:    []byte{op.Log, op.Log, op.PeekLog},
			postCon: stack{Tuple{Bytes("L"), Bytes(emptySeed), Int(7)}},
		},
		{
			name:    "peeklog fail i negative",
			preCon:  stack{Int(-1), Int(7), Bytes("hello")},
			prog:    []byte{op.Log, op.Log, op.PeekLog},
			wanterr: ErrRange,
		},
		{
			name:    "peeklog fail i > log length",
			preCon:  stack{Int(3), Int(7), Bytes("hello")},
			prog:    []byte{op.Log, op.Log, op.PeekLog},
			wanterr: ErrRange,
		},
		{
			name: "txid",
			preCon: stack{
				&value{
					amount:  0,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			prog:    []byte{op.Finalize, op.TxID},
			postCon: stack{Bytes(mustDecodeHex("bd62db7c60e887d5e853ad0a74d304e93e0e597fc13d6be1b6144e4364c3024a"))},
			log: []string{
				"{'F', x'0000000000000000000000000000000000000000000000000000000000000000', 2, x'864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88'}",
			},
		},
		{
			name: "txid with log items",
			preCon: stack{
				&value{
					amount:  0,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
				Bytes("hello"),
				Int(27),
			},
			prog:    []byte{op.Log, op.Log, op.Finalize, op.TxID},
			postCon: stack{Bytes(mustDecodeHex("dd676584bf1f533607fd7a47671dde4f09a9efae4bcd09d53476e5a82d320472"))},
			log: []string{
				"{'L', x'0000000000000000000000000000000000000000000000000000000000000000', 27}",
				"{'L', x'0000000000000000000000000000000000000000000000000000000000000000', 'hello'}",
				"{'F', x'0000000000000000000000000000000000000000000000000000000000000000', 2, x'864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88'}",
			},
		},
		{
			name: "txid fail not finalized",
			preCon: stack{
				Bytes("hello"),
				Int(27),
			},
			prog:    []byte{op.Log, op.Log, op.TxID},
			wanterr: ErrUnfinalized,
		},
		{
			name: "finalize",
			preCon: stack{
				&value{
					amount:  0,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			prog: []byte{op.Finalize},
			log:  []string{"{'F', x'0000000000000000000000000000000000000000000000000000000000000000', 2, x'864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88'}"},
		},
		{
			name: "finalize fail nonzero",
			preCon: stack{
				&value{
					amount:  10,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			prog:    []byte{op.Finalize},
			wanterr: ErrAnchorVal,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Construct VM with c.preCon and c.preArg stacks
			vm := &VM{
				txVersion: 2,
				runlimit:  int64(1000000),
				contract: &contract{
					seed:     make([]byte, 32),
					program:  c.prog,
					stack:    c.preCon,
					typecode: ContractCode,
				},
				argstack: c.preArg,
				caller:   emptySeed,
			}
			err := vm.recoverExec(c.prog)
			vmerr := errors.Root(err)

			if c.wanterr != nil {
				if errors.Root(vmerr) != c.wanterr {
					t.Fatalf("Error mismatch: Got '%s', wanted '%s'", vmerr, c.wanterr)
				}
			} else {
				if vmerr != nil {
					t.Fatal(err)
				}
				// Check contents on stack against expected contents
				compareStacks(t, vm.contract.stack, c.postCon)
				compareStacks(t, vm.argstack, c.postArg)

				// Check log against expected log
				if c.log != nil {
					compareLogs(t, vm.Log, c.log)
				}
			}
		})
	}
}

// a is the log returned by the test, b is the expected log
func compareLogs(t *testing.T, a []Tuple, b []string) {
	if len(a) != len(b) {
		t.Fatalf("log lengths don't match. Got %v, wanted %v", a, b)
	}
	for i, aa := range a {
		if aa.String() != b[i] {
			t.Fatalf("Log mismatch at location %d. Got %v, wanted %v", i, aa, b[i])
		}
	}
}

func (vm *VM) recoverExec(prog []byte) (err error) {
	defer vm.recoverError(&err)
	vm.exec(prog)
	return err
}
