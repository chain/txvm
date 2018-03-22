package op

import "testing"

func TestOpcodeVals(t *testing.T) {
	cases := []struct {
		symbolic, numeric int
	}{
		{Int, 0x20},
		{Add, 0x21},
		{Neg, 0x22},
		{Mul, 0x23},
		{Div, 0x24},
		{Mod, 0x25},
		{GT, 0x26},
		{Not, 0x27},
		{And, 0x28},
		{Or, 0x29},
		{Roll, 0x2a},
		{Bury, 0x2b},
		{Reverse, 0x2c},
		{Get, 0x2d},
		{Put, 0x2e},
		{Depth, 0x2f},
		{Nonce, 0x30},
		{Merge, 0x31},
		{Split, 0x32},
		{Issue, 0x33},
		{Retire, 0x34},
		{Amount, 0x35},
		{AssetID, 0x36},
		{Anchor, 0x37},
		{VMHash, 0x38},
		{SHA256, 0x39},
		{SHA3, 0x3a},
		{CheckSig, 0x3b},
		{Log, 0x3c},
		{PeekLog, 0x3d},
		{TxID, 0x3e},
		{Finalize, 0x3f},
		{Verify, 0x40},
		{JumpIf, 0x41},
		{Exec, 0x42},
		{Call, 0x43},
		{Yield, 0x44},
		{Wrap, 0x45},
		{Input, 0x46},
		{Output, 0x47},
		{Contract, 0x48},
		{Seed, 0x49},
		{Self, 0x4a},
		{Caller, 0x4b},
		{ContractProgram, 0x4c},
		{TimeRange, 0x4d},
		{Prv, 0x4e},
		{Ext, 0x4f},
		{Eq, 0x50},
		{Dup, 0x51},
		{Drop, 0x52},
		{Peek, 0x53},
		{Tuple, 0x54},
		{Untuple, 0x55},
		{Len, 0x56},
		{Field, 0x57},
		{Encode, 0x58},
		{Cat, 0x59},
		{Slice, 0x5a},
		{BitNot, 0x5b},
		{BitAnd, 0x5c},
		{BitOr, 0x5d},
		{BitXor, 0x5e},
	}

	for _, c := range cases {
		if c.symbolic != c.numeric {
			t.Errorf("%s: %d is not %d\n", name[c.symbolic], c.symbolic, c.numeric)
		}
	}
	if MaxSmallInt != 0x1f {
		t.Errorf("MaxSmallInt is %d, want %d\n", MaxSmallInt, 0x1f)
	}
	if MinPushdata != 0x5f {
		t.Errorf("MinPushdata is %d, want %d\n", MinPushdata, 0x5f)
	}
}
