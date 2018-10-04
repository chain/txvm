package asm

import (
	"bytes"
	"io/ioutil"
	"testing"

	"i10r.io/protocol/txvm/op"
)

func TestAssembler(t *testing.T) {
	cases := []struct {
		src  string
		want []byte
	}{
		{"verify", []byte{op.Verify}},
		{"0", []byte{0}},
		{"31", []byte{31}},
		{"32", []byte{op.MinPushdata + 1, 32, op.Int}},
		{"-1", []byte{1, op.Neg}},
		{"bool", []byte{op.Not, op.Not}},
		{"1 dup 1", []byte{1, op.Dup, 1}},
		{"x'00010203'", []byte{op.MinPushdata + 4, 0, 1, 2, 3}},
		{"'abcd'", []byte{op.MinPushdata + 4, 0x61, 0x62, 0x63, 0x64}},
		{"[verify]", []byte{op.MinPushdata + 1, op.Verify}},
		{"2 [1 dup 1] 2", []byte{2, op.MinPushdata + 3, 1, op.Dup, 1, 2}},
		{"{}", []byte{0, op.Tuple}},
		{"{1, 2}", []byte{1, 2, 2, op.Tuple}},
		{"{'abc', {5}, 'def'}", []byte{op.MinPushdata + 3, 0x61, 0x62, 0x63, 5, 1, op.Tuple, op.MinPushdata + 3, 0x64, 0x65, 0x66, 3, op.Tuple}},
		{"jumpif", []byte{op.JumpIf}},
		{"jumpif:$a $a", []byte{0, op.JumpIf}},
		{"jumpif:$a 5 $a", []byte{1, op.JumpIf, 5}},
		{"$a jumpif:$a", []byte{3, op.Neg, op.JumpIf}},
		{"$a 5 jumpif:$a", []byte{5, 4, op.Neg, op.JumpIf}},
		{"jump:$a $a", []byte{1, 0, op.JumpIf}},
		{"jump:$a 5 $a", []byte{1, 1, op.JumpIf, 5}},
		{"$a jump:$a", []byte{1, 4, op.Neg, op.JumpIf}},
		{"$a 5 jump:$a", []byte{5, 1, 5, op.Neg, op.JumpIf}},
		{"$a 5 jump:$b 6 jump:$a $b", []byte{5, 1, 5, op.JumpIf, 6, 1, 9, op.Neg, op.JumpIf}},
		// Detecting programs as programs:
		{"[1 verify] contract", []byte{op.MinPushdata + 2, 1, op.Verify, op.Contract}},
		{"[1 verify] exec", []byte{op.MinPushdata + 2, 1, op.Verify, op.Exec}},
		{"[1 verify] wrap", []byte{op.MinPushdata + 2, 1, op.Verify, op.Wrap}},
		{"[1 verify] yield", []byte{op.MinPushdata + 2, 1, op.Verify, op.Yield}},
		{"[1 verify] output", []byte{op.MinPushdata + 2, 1, op.Verify, op.Output}},
	}
	for _, c := range cases {
		got, err := Assemble(c.src)
		if err != nil {
			t.Errorf("case %s: error: %s", c.src, err)
			continue
		}
		if !bytes.Equal(got, c.want) {
			t.Errorf("case %s: got %x, want %x", c.src, got, c.want)
		}

		dis, err := Disassemble(got)
		if err != nil {
			t.Errorf("case %s: disassembling: %s", c.src, err)
		}
		t.Logf("Assemble+Disassemble: %-30s -> %s", c.src, dis)
		got2, err := Assemble(dis)
		if err != nil {
			t.Errorf("Assemble(Disassemble(Assemble(%s))): %s", c.src, err)
		}
		if !bytes.Equal(got2, c.want) {
			t.Errorf("Assemble(Disassemble(Assemble(%s))): got %x, want %x", c.src, got2, c.want)
		}
	}
}

func BenchmarkAssemble(b *testing.B) {
	b.StopTimer()
	prog, err := ioutil.ReadFile("exampletx.asm")
	if err != nil {
		b.Fatal(err)
	}
	s := string(prog)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_, err = Assemble(s)
		if err != nil {
			b.Fatal(err)
		}
	}
}
