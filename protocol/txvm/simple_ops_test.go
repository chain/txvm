package txvm

import (
	"bytes"
	"encoding/hex"
	"testing"

	"math"

	"i10r.io/errors"
	"i10r.io/protocol/txvm/op"
)

func TestSimpleOpcodes(t *testing.T) {

	cases := []struct {
		name    string
		pre     stack // items on the vm stack before op is run
		opcode  byte  // instruction to be tested
		post    stack // contract stack after execution, from bottom -> top of stack
		wanterr error // check only if post is nil
	}{

		/* NUMERIC AND BOOLEAN INSTRUCTIONS */

		{
			name:   "int",
			pre:    stack{Bytes([]byte{0xE5, 0x8E, 0x26})},
			opcode: op.Int,
			post:   stack{Int(624485)},
		},
		{
			name:    "int fail",
			pre:     stack{Bytes([]byte{0xE5})},
			opcode:  op.Int,
			wanterr: ErrInt,
		},
		{
			name:    "int fail",
			pre:     stack{Int(63)},
			opcode:  op.Int,
			wanterr: ErrType,
		},
		{
			name:   "add small ints",
			pre:    stack{Int(1), Int(5)},
			opcode: op.Add,
			post:   stack{Int(6)},
		},
		{
			name:   "add large ints",
			pre:    stack{Int(1000000000000001), Int(99999999999999)},
			opcode: op.Add,
			post:   stack{Int(1100000000000000)},
		},
		{
			name:    "add fail type",
			pre:     stack{Bytes("hello"), Bytes("there")},
			opcode:  op.Add,
			wanterr: ErrType,
		},
		{
			name:    "add fail underflow",
			pre:     stack{Int(29859)},
			opcode:  op.Add,
			wanterr: ErrUnderflow,
		},
		{
			name:   "negate pos int",
			pre:    stack{Int(15)},
			opcode: op.Neg,
			post:   stack{Int(-15)},
		},
		{
			name:   "negate neg int",
			pre:    stack{Int(-7)},
			opcode: op.Neg,
			post:   stack{Int(7)},
		},
		{
			name:    "negate fail overflow",
			pre:     stack{Int(math.MinInt64)},
			opcode:  op.Neg,
			wanterr: ErrIntOverflow,
		},
		{
			name:    "negate fail string",
			pre:     stack{Bytes("hello")},
			opcode:  op.Neg,
			wanterr: ErrType,
		},
		{
			name:    "negate fail underflow",
			pre:     stack{},
			opcode:  op.Neg,
			wanterr: ErrUnderflow,
		},
		{
			name:   "multiply pos ints",
			pre:    stack{Int(29859), Int(871642)},
			opcode: op.Mul,
			post:   stack{Int(26026358478)},
		},
		{
			name:   "multiply neg ints",
			pre:    stack{Int(-29859), Int(-871642)},
			opcode: op.Mul,
			post:   stack{Int(26026358478)},
		},
		{
			name:   "multiply neg and pos ints",
			pre:    stack{Int(29859), Int(-871642)},
			opcode: op.Mul,
			post:   stack{Int(-26026358478)},
		},
		{
			name:    "multiply fail type",
			pre:     stack{Int(29859), Bytes("hello")},
			opcode:  op.Mul,
			wanterr: ErrType,
		},
		{
			name:    "multiply overflow fail",
			pre:     stack{Int(9999999999999), Int(99999999999999)},
			opcode:  op.Mul,
			wanterr: ErrIntOverflow,
		},
		{
			name:    "multiply fail underflow",
			pre:     stack{Int(29859)},
			opcode:  op.Mul,
			wanterr: ErrUnderflow,
		},
		{
			name:   "divide pos ints",
			pre:    stack{Int(871642), Int(29859)},
			opcode: op.Div,
			post:   stack{Int(29)},
		},
		{
			name:   "divide pos by neg int",
			pre:    stack{Int(871642), Int(-29859)},
			opcode: op.Div,
			post:   stack{Int(-29)},
		},
		{
			name:   "divide neg by neg int",
			pre:    stack{Int(-871642), Int(-29859)},
			opcode: op.Div,
			post:   stack{Int(29)},
		},
		{
			name:   "divide neg by pos int",
			pre:    stack{Int(-871642), Int(29859)},
			opcode: op.Div,
			post:   stack{Int(-29)},
		},
		{
			name:    "divide fail type",
			pre:     stack{Int(29859), Bytes("hello")},
			opcode:  op.Div,
			wanterr: ErrType,
		},
		{
			name:    "divide by zero fail",
			pre:     stack{Int(29859), Int(0)},
			opcode:  op.Div,
			wanterr: ErrIntOverflow,
		},
		{
			name:    "mod fail underflow",
			pre:     stack{Int(29859)},
			opcode:  op.Mod,
			wanterr: ErrUnderflow,
		},
		{
			name:   "mod pos ints",
			pre:    stack{Int(871642), Int(29859)},
			opcode: op.Mod,
			post:   stack{Int(5731)},
		},
		{
			name:   "mod pos by neg int",
			pre:    stack{Int(871642), Int(-29859)},
			opcode: op.Mod,
			post:   stack{Int(5731)},
		},
		{
			name:   "mod neg by neg int",
			pre:    stack{Int(-871642), Int(-29859)},
			opcode: op.Mod,
			post:   stack{Int(-5731)},
		},
		{
			name:   "mod neg by pos int",
			pre:    stack{Int(-871642), Int(29859)},
			opcode: op.Mod,
			post:   stack{Int(-5731)},
		},
		{
			name:    "mod fail type",
			pre:     stack{Int(29859), Bytes("hello")},
			opcode:  op.Mod,
			wanterr: ErrType,
		},
		{
			name:    "mod by zero fail",
			pre:     stack{Int(29859), Int(0)},
			opcode:  op.Mod,
			wanterr: ErrIntOverflow,
		},
		{
			name:    "mod fail underflow",
			pre:     stack{Int(29859)},
			opcode:  op.Mod,
			wanterr: ErrUnderflow,
		},
		{
			name:   "pos ints gt",
			pre:    stack{Int(871642), Int(29859)},
			opcode: op.GT,
			post:   stack{Int(1)},
		},
		{
			name:   "neg ints gt",
			pre:    stack{Int(-29859), Int(-871642)},
			opcode: op.GT,
			post:   stack{Int(1)},
		},
		{
			name:   "pos ints not gt",
			pre:    stack{Int(29859), Int(871642)},
			opcode: op.GT,
			post:   stack{Int(0)},
		},
		{
			name:   "neg ints not gt",
			pre:    stack{Int(-871642), Int(-29859)},
			opcode: op.GT,
			post:   stack{Int(0)},
		},
		{
			name:    "gt fail type",
			pre:     stack{Int(29859), Bytes("hello")},
			opcode:  op.GT,
			wanterr: ErrType,
		},
		{
			name:    "gt fail underflow",
			pre:     stack{Int(29859)},
			opcode:  op.GT,
			wanterr: ErrUnderflow,
		},
		{
			name:   "not 0",
			pre:    stack{Int(0)},
			opcode: op.Not,
			post:   stack{Int(1)},
		},
		{
			name:   "not 1",
			pre:    stack{Int(1)},
			opcode: op.Not,
			post:   stack{Int(0)},
		},
		{
			name:   "not 2",
			pre:    stack{Int(2)},
			opcode: op.Not,
			post:   stack{Int(0)},
		},
		{
			name:   "not empty string",
			pre:    stack{Bytes("")},
			opcode: op.Not,
			post:   stack{Int(0)},
		},
		{
			name:   "not abc",
			pre:    stack{Bytes("abc")},
			opcode: op.Not,
			post:   stack{Int(0)},
		},
		{
			name:   "not empty tuple",
			pre:    stack{Tuple{}},
			opcode: op.Not,
			post:   stack{Int(0)},
		},
		{
			name:   "not {0}",
			pre:    stack{Tuple{Int(0)}},
			opcode: op.Not,
			post:   stack{Int(0)},
		},
		{
			name:   "not {1}",
			pre:    stack{Tuple{Int(1)}},
			opcode: op.Not,
			post:   stack{Int(0)},
		},
		{
			name:    "not fail underflow",
			pre:     stack{},
			opcode:  op.Not,
			wanterr: ErrUnderflow,
		},
		{
			name:   "1 and 1",
			pre:    stack{Int(1), Int(1)},
			opcode: op.And,
			post:   stack{Int(1)},
		},
		{
			name:   "1 and 0",
			pre:    stack{Int(1), Int(0)},
			opcode: op.And,
			post:   stack{Int(0)},
		},
		{
			name:   "0 and 1",
			pre:    stack{Int(0), Int(1)},
			opcode: op.And,
			post:   stack{Int(0)},
		},
		{
			name:   "0 and 0",
			pre:    stack{Int(0), Int(0)},
			opcode: op.And,
			post:   stack{Int(0)},
		},
		{
			name:   "1 and empty string",
			pre:    stack{Int(1), Bytes("")},
			opcode: op.And,
			post:   stack{Int(1)},
		},
		{
			name:   "1 and abc",
			pre:    stack{Int(1), Bytes("abc")},
			opcode: op.And,
			post:   stack{Int(1)},
		},
		{
			name:   "1 and empty tuple",
			pre:    stack{Int(1), Tuple{}},
			opcode: op.And,
			post:   stack{Int(1)},
		},
		{
			name:   "1 and {0}",
			pre:    stack{Int(1), Tuple{Int(0)}},
			opcode: op.And,
			post:   stack{Int(1)},
		},
		{
			name:   "empty string and 1",
			pre:    stack{Bytes(""), Int(1)},
			opcode: op.And,
			post:   stack{Int(1)},
		},
		{
			name:   "abc and 1",
			pre:    stack{Bytes("abc"), Int(1)},
			opcode: op.And,
			post:   stack{Int(1)},
		},
		{
			name:   "empty tuple and 1",
			pre:    stack{Tuple{}, Int(1)},
			opcode: op.And,
			post:   stack{Int(1)},
		},
		{
			name:   "{0} and 1",
			pre:    stack{Tuple{Int(0)}, Int(1)},
			opcode: op.And,
			post:   stack{Int(1)},
		},
		{
			name:   "0 and empty string",
			pre:    stack{Int(0), Bytes("")},
			opcode: op.And,
			post:   stack{Int(0)},
		},
		{
			name:   "0 and abc",
			pre:    stack{Int(0), Bytes("abc")},
			opcode: op.And,
			post:   stack{Int(0)},
		},
		{
			name:   "0 and empty tuple",
			pre:    stack{Int(0), Tuple{}},
			opcode: op.And,
			post:   stack{Int(0)},
		},
		{
			name:   "0 and {0}",
			pre:    stack{Int(0), Tuple{Int(0)}},
			opcode: op.And,
			post:   stack{Int(0)},
		},
		{
			name:    "and fail underflow",
			pre:     stack{Int(29859)},
			opcode:  op.And,
			wanterr: ErrUnderflow,
		},
		{
			name:   "1 or 1",
			pre:    stack{Int(1), Int(1)},
			opcode: op.Or,
			post:   stack{Int(1)},
		},
		{
			name:   "1 or 0",
			pre:    stack{Int(1), Int(0)},
			opcode: op.Or,
			post:   stack{Int(1)},
		},
		{
			name:   "0 or 1",
			pre:    stack{Int(0), Int(1)},
			opcode: op.Or,
			post:   stack{Int(1)},
		},
		{
			name:   "0 or 0",
			pre:    stack{Int(0), Int(0)},
			opcode: op.Or,
			post:   stack{Int(0)},
		},
		{
			name:   "1 or empty string",
			pre:    stack{Int(1), Bytes("")},
			opcode: op.Or,
			post:   stack{Int(1)},
		},
		{
			name:   "1 or abc",
			pre:    stack{Int(1), Bytes("abc")},
			opcode: op.Or,
			post:   stack{Int(1)},
		},
		{
			name:   "1 or empty tuple",
			pre:    stack{Int(1), Tuple{}},
			opcode: op.Or,
			post:   stack{Int(1)},
		},
		{
			name:   "1 or {0}",
			pre:    stack{Int(1), Tuple{Int(0)}},
			opcode: op.Or,
			post:   stack{Int(1)},
		},
		{
			name:   "empty string or 1",
			pre:    stack{Bytes(""), Int(1)},
			opcode: op.Or,
			post:   stack{Int(1)},
		},
		{
			name:   "abc or 1",
			pre:    stack{Bytes("abc"), Int(1)},
			opcode: op.Or,
			post:   stack{Int(1)},
		},
		{
			name:   "empty tuple or 1",
			pre:    stack{Tuple{}, Int(1)},
			opcode: op.Or,
			post:   stack{Int(1)},
		},
		{
			name:   "{0} or 1",
			pre:    stack{Tuple{Int(0)}, Int(1)},
			opcode: op.Or,
			post:   stack{Int(1)},
		},
		{
			name:   "0 or empty string",
			pre:    stack{Int(0), Bytes("")},
			opcode: op.Or,
			post:   stack{Int(1)},
		},
		{
			name:   "0 or abc",
			pre:    stack{Int(0), Bytes("abc")},
			opcode: op.Or,
			post:   stack{Int(1)},
		},
		{
			name:   "0 or empty tuple",
			pre:    stack{Int(0), Tuple{}},
			opcode: op.Or,
			post:   stack{Int(1)},
		},
		{
			name:   "0 or {0}",
			pre:    stack{Int(0), Tuple{Int(0)}},
			opcode: op.Or,
			post:   stack{Int(1)},
		},

		{
			name:    "or fail underflow",
			pre:     stack{Int(29859)},
			opcode:  op.Or,
			wanterr: ErrUnderflow,
		},

		/* CRYPTO INSTRUCTIONS */

		{
			name:   "vmhash",
			pre:    stack{Bytes("f value"), Bytes("x value")},
			opcode: op.VMHash,
			post:   stack{Bytes(mustDecodeHex("73de9ff7510977226f8474cc617d30accf4eba3cc0deadcd809b3a38e70e914e"))},
		},
		{
			name:    "vmhash fail not string",
			pre:     stack{Int(5), Int(8)},
			opcode:  op.VMHash,
			wanterr: ErrType,
		},
		{
			name:   "sha256",
			pre:    stack{Bytes("x value")},
			opcode: op.SHA256,
			post:   stack{Bytes(mustDecodeHex("e8125a72205b4cad517142edf11c79a42fa66a58891f8bb803ad1cc90f80bcb6"))},
		},
		{
			name:    "sha256 fail not string",
			pre:     stack{Int(5)},
			opcode:  op.SHA256,
			wanterr: ErrType,
		},
		{
			name:   "sha3",
			pre:    stack{Bytes("x value")},
			opcode: op.SHA3,
			post:   stack{Bytes(mustDecodeHex("4828b0cb99c24327650da57e64c7bf7d6debdf654500e3fb900e54c59be675ef"))},
		},
		{
			name:    "sha3 fail not string",
			pre:     stack{Int(5)},
			opcode:  op.SHA3,
			wanterr: ErrType,
		},
		{
			name: "checksig success",
			pre: stack{
				Bytes(mustDecodeHex("f6c0dadc897db49d891190d6cd9a41f614c17db8189320bfa7dc8d55758ed4ce")),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(mustDecodeHex("502a55ab70f4f921cb88650db040dcc93dc07707892aab41b3c12e5a929e2e2750fe557b197ce9bec337fbee8c020c1aa59d7790c3139728ed8ad54708be710e")),
				Int(0),
			},
			opcode: op.CheckSig,
			post:   stack{Int(1)},
		},
		{
			name: "checksig fail empty",
			pre: stack{
				Bytes(mustDecodeHex("f6c0dadc897db49d891190d6cd9a41f614c17db8189320bfa7dc8d55758ed4ce")),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(""),
				Int(0),
			},
			opcode: op.CheckSig,
			post:   stack{Int(0)},
		},
		{
			name: "checksig fail wrong sig",
			pre: stack{
				Bytes(mustDecodeHex("f6c0dadc897db49d891190d6cd9a41f614c17db8189320bfa7dc8d55758ed4ce")),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(mustDecodeHex("102a55ab70f4f921cb88650db040dcc93dc07707892aab41b3c12e5a929e2e2750fe557b197ce9bec337fbee8c020c1aa59d7790c3139728ed8ad54708be710e")),
				Int(0),
			},
			opcode:  op.CheckSig,
			wanterr: ErrSignature,
		},
		{
			name: "checksig fail pubkey wrong length",
			pre: stack{
				Bytes(mustDecodeHex("f6c0dadc897db49d891190d6cd9a41f614c17db8189320bfa7dc8d55758ed4ce")),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d")),
				Bytes(mustDecodeHex("102a55ab70f4f921cb88650db040dcc93dc07707892aab41b3c12e5a929e2e2750fe557b197ce9bec337fbee8c020c1aa59d7790c3139728ed8ad54708be710e")),
				Int(0),
			},
			opcode:  op.CheckSig,
			wanterr: ErrPubSize,
		},
		{
			name: "checksig fail sig wrong length",
			pre: stack{
				Bytes(mustDecodeHex("f6c0dadc897db49d891190d6cd9a41f614c17db8189320bfa7dc8d55758ed4ce")),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(mustDecodeHex("502a55ab70f4f921cb88650db040dcc93dc07707892aab41b3c12e5a929e2e2750fe557b197ce9bec337fbee8c020c1aa59d7790c3139728ed8ad54708be71")),
				Int(0),
			},
			opcode:  op.CheckSig,
			wanterr: ErrSigSize,
		},
		{
			name: "checksig fail unrecognized extension",
			pre: stack{
				Bytes(mustDecodeHex("f6c0dadc897db49d891190d6cd9a41f614c17db8189320bfa7dc8d55758ed4ce")),
				Bytes(mustDecodeHex("4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")),
				Bytes(mustDecodeHex("502a55ab70f4f921cb88650db040dcc93dc07707892aab41b3c12e5a929e2e2750fe557b197ce9bec337fbee8c020c1aa59d7790c3139728ed8ad54708be71")),
				Int(2),
			},
			opcode:  op.CheckSig,
			wanterr: ErrExt,
		},

		/* STACK INSTRUCTIONS */

		{
			name:   "0 roll",
			pre:    stack{Int(5), Int(0)},
			opcode: op.Roll,
			post:   stack{Int(5)},
		},
		{
			name:   "1 roll",
			pre:    stack{Int(7), Int(5), Int(1)},
			opcode: op.Roll,
			post:   stack{Int(5), Int(7)},
		},
		{
			name:   "1 roll tuple",
			pre:    stack{Tuple{Int(7), Int(0), Int(8)}, Int(5), Int(1)},
			opcode: op.Roll,
			post:   stack{Int(5), Tuple{Int(7), Int(0), Int(8)}},
		},
		{
			name:   "1 roll bytes",
			pre:    stack{Bytes("BAZ"), Bytes("ALLO"), Int(1)},
			opcode: op.Roll,
			post:   stack{Bytes("ALLO"), Bytes("BAZ")},
		},
		{
			name:   "5 roll",
			pre:    stack{Int(7), Int(5), Int(1), Int(4), Int(2000), Int(10), Int(5)},
			opcode: op.Roll,
			post:   stack{Int(5), Int(1), Int(4), Int(2000), Int(10), Int(7)},
		},
		{
			name:   "5 roll tuple",
			pre:    stack{Tuple{Int(7), Int(0), Int(8)}, Int(5), Int(1), Int(4), Int(2000), Int(13), Int(5)},
			opcode: op.Roll,
			post:   stack{Int(5), Int(1), Int(4), Int(2000), Int(13), Tuple{Int(7), Int(0), Int(8)}},
		},
		{
			name:   "5 roll bytes",
			pre:    stack{Bytes("BAZ"), Bytes("ALLO"), Bytes("BAR"), Bytes("FOO"), Bytes("DEADBEEF"), Bytes("hi"), Int(5)},
			opcode: op.Roll,
			post:   stack{Bytes("ALLO"), Bytes("BAR"), Bytes("FOO"), Bytes("DEADBEEF"), Bytes("hi"), Bytes("BAZ")},
		},
		{
			name:    "roll fail",
			pre:     stack{Int(5), Int(1)},
			opcode:  op.Roll,
			wanterr: ErrStackRange,
		},
		{
			name:   "0 bury",
			pre:    stack{Int(5), Int(0)},
			opcode: op.Bury,
			post:   stack{Int(5)},
		},
		{
			name:   "1 bury",
			pre:    stack{Int(7), Int(5), Int(1)},
			opcode: op.Bury,
			post:   stack{Int(5), Int(7)},
		},
		{
			name:   "1 bury tuple",
			pre:    stack{Int(7), Tuple{Int(7), Int(0), Int(8)}, Int(1)},
			opcode: op.Bury,
			post:   stack{Tuple{Int(7), Int(0), Int(8)}, Int(7)},
		},
		{
			name:   "1 bury bytes",
			pre:    stack{Int(7), Int(5), Int(1)},
			opcode: op.Bury,
			post:   stack{Int(5), Int(7)},
		},
		{
			name:   "5 bury",
			pre:    stack{Int(7), Int(5), Int(1), Int(4), Int(2000), Int(10), Int(5)},
			opcode: op.Bury,
			post:   stack{Int(10), Int(7), Int(5), Int(1), Int(4), Int(2000)},
		},
		{
			name:   "5 bury tuple",
			pre:    stack{Int(7), Int(5), Int(1), Int(4), Int(2000), Tuple{Int(7), Int(0), Int(8)}, Int(5)},
			opcode: op.Bury,
			post:   stack{Tuple{Int(7), Int(0), Int(8)}, Int(7), Int(5), Int(1), Int(4), Int(2000)},
		},
		{
			name:   "5 bury bytes",
			pre:    stack{Bytes("BAZ"), Bytes("ALLO"), Bytes("BAR"), Bytes("FOO"), Bytes("DEADBEEF"), Bytes("hi"), Int(5)},
			opcode: op.Bury,
			post:   stack{Bytes("hi"), Bytes("BAZ"), Bytes("ALLO"), Bytes("BAR"), Bytes("FOO"), Bytes("DEADBEEF")},
		},
		{
			name:    "bury fail",
			pre:     stack{Int(7), Int(5), Int(1), Int(4), Int(2000), Int(10), Int(6)},
			opcode:  op.Bury,
			wanterr: ErrStackRange,
		},
		{
			name:   "0 reverse",
			pre:    stack{Int(5), Int(0)},
			opcode: op.Reverse,
			post:   stack{Int(5)},
		},
		{
			name:   "1 reverse",
			pre:    stack{Int(7), Int(5), Int(1)},
			opcode: op.Reverse,
			post:   stack{Int(7), Int(5)},
		},
		{
			name:   "1 reverse tuple",
			pre:    stack{Int(7), Tuple{Int(7), Int(0), Int(8)}, Int(1)},
			opcode: op.Reverse,
			post:   stack{Int(7), Tuple{Int(7), Int(0), Int(8)}},
		},
		{
			name:   "1 reverse bytes",
			pre:    stack{Bytes("BAZ"), Bytes("ALLO"), Bytes("BAR"), Bytes("FOO"), Bytes("DEADBEEF"), Bytes("hi"), Int(1)},
			opcode: op.Reverse,
			post:   stack{Bytes("BAZ"), Bytes("ALLO"), Bytes("BAR"), Bytes("FOO"), Bytes("DEADBEEF"), Bytes("hi")},
		},
		{
			name:   "2 reverse",
			pre:    stack{Int(7), Int(5), Int(2)},
			opcode: op.Reverse,
			post:   stack{Int(5), Int(7)},
		},
		{
			name:   "2 reverse bytes",
			pre:    stack{Bytes("BAZ"), Bytes("ALLO"), Bytes("BAR"), Bytes("FOO"), Bytes("DEADBEEF"), Bytes("hi"), Int(2)},
			opcode: op.Reverse,
			post:   stack{Bytes("BAZ"), Bytes("ALLO"), Bytes("BAR"), Bytes("FOO"), Bytes("hi"), Bytes("DEADBEEF")},
		},
		{
			name:   "5 reverse",
			pre:    stack{Int(7), Int(5), Int(1), Int(4), Int(2000), Int(10), Int(5)},
			opcode: op.Reverse,
			post:   stack{Int(7), Int(10), Int(2000), Int(4), Int(1), Int(5)},
		},
		{
			name:   "5 reverse bytes",
			pre:    stack{Bytes("BAZ"), Bytes("ALLO"), Bytes("BAR"), Bytes("FOO"), Bytes("DEADBEEF"), Bytes("hi"), Int(5)},
			opcode: op.Reverse,
			post:   stack{Bytes("BAZ"), Bytes("hi"), Bytes("DEADBEEF"), Bytes("FOO"), Bytes("BAR"), Bytes("ALLO")},
		},
		{
			name:    "reverse empty fail",
			pre:     stack{},
			opcode:  op.Reverse,
			wanterr: ErrUnderflow,
		},
		{
			name:    "reverse not enough on stack fail",
			pre:     stack{Int(2)},
			opcode:  op.Reverse,
			wanterr: ErrStackRange,
		},
		{
			name:   "depth 0",
			pre:    stack{},
			opcode: op.Depth,
			post:   stack{Int(0)},
		},
		{
			name:    "ext",
			pre:     stack{Int(10)},
			opcode:  op.Ext,
			wanterr: ErrExt,
		},
		{
			name:    "prv",
			pre:     stack{},
			opcode:  op.Prv,
			wanterr: ErrPrv,
		},

		/* DATA INSTRUCTIONS */

		{
			name:   "int equal",
			pre:    stack{Int(7), Int(7)},
			opcode: op.Eq,
			post:   stack{Int(1)},
		},
		{
			name:   "bytes equal",
			pre:    stack{Bytes("hi"), Bytes("hi")},
			opcode: op.Eq,
			post:   stack{Int(1)},
		},
		{
			name:   "int not equal",
			pre:    stack{Int(5), Int(7)},
			opcode: op.Eq,
			post:   stack{Int(0)},
		},
		{
			name:   "bytes not equal",
			pre:    stack{Bytes("hi"), Bytes("byte")},
			opcode: op.Eq,
			post:   stack{Int(0)},
		},
		{
			name:   "tuples equal",
			pre:    stack{Tuple{Int(7), Int(0), Int(8)}, Tuple{Int(7), Int(0), Int(8)}},
			opcode: op.Eq,
			post:   stack{Int(0)},
		},
		{
			name:   "tuples not equal",
			pre:    stack{Tuple{Int(7), Int(0), Int(8)}, Tuple{Int(7), Int(0), Int(7)}},
			opcode: op.Eq,
			post:   stack{Int(0)},
		},
		{
			name:   "different types not equal",
			pre:    stack{Int(7), Bytes("hi")},
			opcode: op.Eq,
			post:   stack{Int(0)},
		},
		{
			name:    "equal fail",
			pre:     stack{Tuple{Int(7), Int(0), Int(8)}},
			opcode:  op.Eq,
			wanterr: ErrUnderflow,
		},
		{
			name:   "dup int",
			pre:    stack{Int(1000)},
			opcode: op.Dup,
			post:   stack{Int(1000), Int(1000)},
		},
		{
			name:   "dup bytes",
			pre:    stack{Bytes("hello")},
			opcode: op.Dup,
			post:   stack{Bytes("hello"), Bytes("hello")},
		},
		{
			name:   "dup tuple",
			pre:    stack{Tuple{Int(7), Int(0), Int(8)}},
			opcode: op.Dup,
			post:   stack{Tuple{Int(7), Int(0), Int(8)}, Tuple{Int(7), Int(0), Int(8)}},
		},
		{
			name: "dup fail not data",
			pre: stack{
				&value{
					amount:  0,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			opcode:  op.Dup,
			wanterr: ErrType,
		},
		{
			name:    "dup fail underflow",
			pre:     stack{},
			opcode:  op.Dup,
			wanterr: ErrUnderflow,
		},
		{
			name:   "drop int",
			pre:    stack{Int(1000)},
			opcode: op.Drop,
			post:   stack{},
		},
		{
			name:   "drop bytes",
			pre:    stack{Int(1000), Bytes("hello")},
			opcode: op.Drop,
			post:   stack{Int(1000)},
		},
		{
			name:   "drop tuple",
			pre:    stack{Bytes("hello"), Tuple{Int(7), Int(0), Int(8)}},
			opcode: op.Drop,
			post:   stack{Bytes("hello")},
		},
		{
			name: "drop zero value",
			pre: stack{
				&value{
					amount:  0,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			opcode: op.Drop,
			post:   stack{},
		},
		{
			name: "drop nonzero value fail",
			pre: stack{
				&value{
					amount:  10,
					assetID: Bytes("apples"),
					anchor:  mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
				},
			},
			opcode:  op.Drop,
			wanterr: ErrType,
		},
		{
			name: "drop fail contract",
			pre: stack{
				&contract{
					seed:     mustDecodeHex("864ae6a14ffddc0741743aa862283dfaf7f8aa81e5c3b0dfec36d65a66ccab88"),
					typecode: ContractCode,
					program:  []byte{op.Mod, op.Put},
					stack:    stack{Int(12), Int(10)},
				},
			},
			opcode:  op.Drop,
			wanterr: ErrType,
		},
		{
			name:    "drop fail underflow",
			pre:     stack{},
			opcode:  op.Drop,
			wanterr: ErrUnderflow,
		},
		{
			name:   "0 peek int",
			pre:    stack{Int(1000), Bytes("foo"), Int(7), Int(0)},
			opcode: op.Peek,
			post:   stack{Int(1000), Bytes("foo"), Int(7), Int(7)},
		},
		{
			name:   "1 peek bytes",
			pre:    stack{Int(1000), Bytes("foo"), Int(7), Int(1)},
			opcode: op.Peek,
			post:   stack{Int(1000), Bytes("foo"), Int(7), Bytes("foo")},
		},
		{
			name:   "2 peek int",
			pre:    stack{Int(1000), Bytes("foo"), Int(7), Int(2)},
			opcode: op.Peek,
			post:   stack{Int(1000), Bytes("foo"), Int(7), Int(1000)},
		},
		{
			name:    "peek fail",
			pre:     stack{},
			opcode:  op.Peek,
			wanterr: ErrUnderflow,
		},
		{
			name:   "tuple of ints",
			pre:    stack{Int(1000), Int(3), Int(7), Int(3)},
			opcode: op.Tuple,
			post:   stack{Tuple{Int(1000), Int(3), Int(7), Int(3)}},
		},
		{
			name:   "tuple of bytes",
			pre:    stack{Bytes("hi"), Bytes("bye"), Int(2)},
			opcode: op.Tuple,
			post:   stack{Tuple{Bytes("hi"), Bytes("bye")}},
		},
		{
			name:   "tuple of tuples",
			pre:    stack{Tuple{Bytes("hi"), Bytes("bye")}, Tuple{Bytes("foo"), Bytes("bar")}, Int(2)},
			opcode: op.Tuple,
			post:   stack{Tuple{Tuple{Bytes("hi"), Bytes("bye")}, Tuple{Bytes("foo"), Bytes("bar")}}},
		},
		{
			name:   "mixed tuple",
			pre:    stack{Bytes("hi"), Int(0), Tuple{Bytes("hi"), Bytes("bye")}, Int(3)},
			opcode: op.Tuple,
			post:   stack{Tuple{Bytes("hi"), Int(0), Tuple{Bytes("hi"), Bytes("bye")}}},
		},
		{
			name:    "tuple fail",
			pre:     stack{},
			opcode:  op.Tuple,
			wanterr: ErrUnderflow,
		},
		{
			name:   "untuple ints",
			pre:    stack{Tuple{Int(1000), Int(3), Int(7)}},
			opcode: op.Untuple,
			post:   stack{Int(1000), Int(3), Int(7), Int(3)},
		},
		{
			name:   "untuple bytes",
			pre:    stack{Tuple{Bytes("hi"), Bytes("bye")}},
			opcode: op.Untuple,
			post:   stack{Bytes("hi"), Bytes("bye"), Int(2)},
		},
		{
			name:    "untuple fail",
			pre:     stack{},
			opcode:  op.Untuple,
			wanterr: ErrUnderflow,
		},
		{
			name:   "len bytes",
			pre:    stack{Bytes("hello!")},
			opcode: op.Len,
			post:   stack{Int(6)},
		},
		{
			name:   "len tuple",
			pre:    stack{Tuple{Bytes("hi"), Bytes("bye")}},
			opcode: op.Len,
			post:   stack{Int(2)},
		},
		{
			name:    "len fail",
			pre:     stack{},
			opcode:  op.Len,
			wanterr: ErrUnderflow,
		},
		{
			name:   "0 field",
			pre:    stack{Tuple{Bytes("hi"), Bytes("bye")}, Int(0)},
			opcode: op.Field,
			post:   stack{Bytes("hi")},
		},
		{
			name:   "1 field",
			pre:    stack{Tuple{Bytes("hi"), Bytes("bye")}, Int(1)},
			opcode: op.Field,
			post:   stack{Bytes("bye")},
		},
		{
			name:    "field not tuple fail",
			pre:     stack{Bytes("hi"), Bytes("bye"), Int(1)},
			opcode:  op.Field,
			wanterr: ErrType,
		},
		{
			name:    "field not int fail",
			pre:     stack{Tuple{Bytes("hi"), Bytes("bye")}, Tuple{Bytes("hi"), Bytes("bye")}},
			opcode:  op.Field,
			wanterr: ErrType,
		},
		{
			name:    "field index not found fail",
			pre:     stack{Tuple{Bytes("hi"), Bytes("bye")}, Int(2)},
			opcode:  op.Field,
			wanterr: ErrRange,
		},
		{
			name:    "field empty fail",
			pre:     stack{Int(2)},
			opcode:  op.Field,
			wanterr: ErrUnderflow,
		},
		{
			name:   "encode small int",
			pre:    stack{Int(11)},
			opcode: op.Encode,
			post:   stack{Bytes([]byte{0x0b})},
		},
		{
			name:   "encode large int",
			pre:    stack{Int(1000)},
			opcode: op.Encode,
			post:   stack{Bytes([]byte{0x61, 0xe8, 0x07, op.Int})},
		},
		{
			name:   "encode bytes",
			pre:    stack{Bytes("hello there")},
			opcode: op.Encode,
			post:   stack{Bytes("jhello there")},
		},
		{
			name:   "encode tuple",
			pre:    stack{Tuple{Bytes("hi"), Bytes("bye")}},
			opcode: op.Encode,
			post:   stack{Bytes(mustDecodeHex("616869626279650254"))},
		},
		{
			name:    "encode empty fail",
			pre:     stack{},
			opcode:  op.Encode,
			wanterr: ErrUnderflow,
		},
		{
			name:   "cat",
			pre:    stack{Bytes("hello"), Bytes("there")},
			opcode: op.Cat,
			post:   stack{Bytes("hellothere")},
		},
		{
			name:    "cat fail int",
			pre:     stack{Int(9), Bytes("there")},
			opcode:  op.Cat,
			wanterr: ErrType,
		},
		{
			name:    "cat fail tuple",
			pre:     stack{Tuple{Bytes("hello")}, Bytes("there")},
			opcode:  op.Cat,
			wanterr: ErrType,
		},
		{
			name:    "cat fail one item",
			pre:     stack{Bytes("there")},
			opcode:  op.Cat,
			wanterr: ErrUnderflow,
		},
		{
			name:   "slice",
			pre:    stack{Bytes("hello there"), Int(2), Int(10)},
			opcode: op.Slice,
			post:   stack{Bytes("llo ther")},
		},
		{
			name:    "slice fail end<start",
			pre:     stack{Bytes("hello there"), Int(2), Int(1)},
			opcode:  op.Slice,
			wanterr: ErrSliceRange,
		},
		{
			name:    "slice fail start<0",
			pre:     stack{Bytes("hello there"), Int(-2), Int(10)},
			opcode:  op.Slice,
			wanterr: ErrSliceRange,
		},
		{
			name:    "slice fail end>len(str)",
			pre:     stack{Bytes("hello there"), Int(2), Int(20)},
			opcode:  op.Slice,
			wanterr: ErrSliceRange,
		},
		{
			name:    "slice fail not int",
			pre:     stack{Bytes("hello there"), Int(2)},
			opcode:  op.Slice,
			wanterr: ErrType,
		},
		{
			name:    "slice fail no args",
			pre:     stack{},
			opcode:  op.Slice,
			wanterr: ErrUnderflow,
		},
		{
			name:   "bitnot",
			pre:    stack{Bytes("hello")},
			opcode: op.BitNot,
			post:   stack{Bytes(mustDecodeHex("979a939390"))},
		},
		{
			name:    "bitnot fail not string",
			pre:     stack{Int(10)},
			opcode:  op.BitNot,
			wanterr: ErrType,
		},
		{
			name:    "bitnot fail empty",
			pre:     stack{},
			opcode:  op.BitNot,
			wanterr: ErrUnderflow,
		},
		{
			name:   "bitand",
			pre:    stack{Bytes("hello"), Bytes("there")},
			opcode: op.BitAnd,
			post:   stack{Bytes("``d`e")},
		},
		{
			name:    "bitand fail not string",
			pre:     stack{Int(10), Bytes("hello")},
			opcode:  op.BitAnd,
			wanterr: ErrType,
		},
		{
			name:    "bitand fail empty",
			pre:     stack{},
			opcode:  op.BitAnd,
			wanterr: ErrUnderflow,
		},
		{
			name:   "bitor",
			pre:    stack{Bytes("hello"), Bytes("there")},
			opcode: op.BitOr,
			post:   stack{Bytes("|mm~o")},
		},
		{
			name:    "bitor fail not string",
			pre:     stack{Int(10), Bytes("hello")},
			opcode:  op.BitOr,
			wanterr: ErrType,
		},
		{
			name:    "bitor fail empty",
			pre:     stack{},
			opcode:  op.BitOr,
			wanterr: ErrUnderflow,
		},
		{
			name:   "bitxor",
			pre:    stack{Bytes("hello"), Bytes("there")},
			opcode: op.BitXor,
			post:   stack{Bytes(mustDecodeHex("1c0d091e0a"))},
		},
		{
			name:    "bitxor fail not string",
			pre:     stack{Int(10), Bytes("hello")},
			opcode:  op.BitXor,
			wanterr: ErrType,
		},
		{
			name:    "bitxor fail empty",
			pre:     stack{},
			opcode:  op.BitXor,
			wanterr: ErrUnderflow,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			op := []byte{c.opcode}

			// Construct VM with c.pre on stack, validate op
			vm := &VM{
				txVersion: 3,
				runlimit:  int64(1000000),
				contract:  &contract{seed: make([]byte, 32), program: op, stack: c.pre},
			}
			err := vm.recoverExec(op)
			vmerr := errors.Root(err)

			if c.post != nil {
				if vmerr != nil {
					t.Fatal(err)
				}
				// Check contents on stack against expected contents
				compareStacks(t, vm.contract.stack, c.post)
			} else {
				if vmerr != c.wanterr {
					t.Fatalf("Error mismatch: Got '%s', wanted '%s'", vmerr, c.wanterr)
				}
			}
		})
	}
}

// a is the stack returned by the test, b is the expected stack.
func compareStacks(t *testing.T, a, b stack) {
	if a.Len() != b.Len() {
		t.Fatalf("Stack length mismatch. Got %v, wanted %v", a, b)
	}

	for i, aItem := range a {
		switch aa := aItem.(type) {
		case Int:
			bb, ok := b[i].(Int)
			checkTypeOk(t, ok, i, a[i], b[i])
			if aa != bb {
				t.Fatalf("Stack mismatch of Ints at location %d. Got %v, wanted %v", i, a[i], b[i])
			}
		case Bytes:
			bb, ok := b[i].(Bytes)
			checkTypeOk(t, ok, i, a[i], b[i])
			for j := 0; j < len(aa); j++ {
				if aa[j] != bb[j] {
					t.Fatalf("Stack mismatch of Bytes at location %d. Got %v, wanted %v", i, a[i], b[i])
				}
			}
		case Tuple:
			bb, ok := b[i].(Tuple)
			checkTypeOk(t, ok, i, a[i], b[i])
			for j := 0; j < len(aa); j++ {
				compareStacks(t, stack{aa[j]}, stack{bb[j]})
			}
		case *value:
			bb, ok := b[i].(*value)
			checkTypeOk(t, ok, i, a[i], b[i])
			if bb.amount != aa.amount || !bytes.Equal(bb.assetID, aa.assetID) || !bytes.Equal(bb.anchor, aa.anchor) {
				t.Fatalf("Mismatch of fields in value. Got %v, wanted %v", aa, bb)
			}
		case *contract:
			bb, ok := b[i].(*contract)
			checkTypeOk(t, ok, i, a[i], b[i])
			if bb.typecode != aa.typecode || !bytes.Equal(bb.seed, aa.seed) || !bytes.Equal(bb.program, aa.program) {
				t.Fatalf("Mismatch of fields in contract. Got %v, wanted %v", aa, bb)
			}
			compareStacks(t, aa.stack, bb.stack)
		default:
			t.Fatalf("Stack item at location %d is of an unrecognized type. Got %v, wanted %v", i, a[i], b[i])
		}
	}
}

func checkTypeOk(t *testing.T, ok bool, i int, a, b Item) {
	if !ok {
		t.Fatalf("Stack item at location %d is a different type than expected. Got %v, wanted %v", i, a, b)
	}
}

func mustDecodeHex(s string) []byte {
	out, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return out
}
