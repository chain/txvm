package txvmutil

import (
	"bytes"
	"testing"

	"github.com/chain/txvm/protocol/txvm"
	"github.com/chain/txvm/protocol/txvm/op"
)

func newBuilder() *Builder {
	return new(Builder)
}

func TestBuilder(t *testing.T) {
	cases := []struct {
		b    *Builder
		want []byte
	}{
		{
			b:    newBuilder().PushdataInt64(0),
			want: []byte{0},
		},
		{
			b:    newBuilder().PushdataInt64(32),
			want: []byte{op.MinPushdata + 1, 32, op.Int},
		},
		{
			b: newBuilder().
				PushdataInt64(1).
				Op(op.Dup).
				PushdataInt64(1),
			want: []byte{1, op.Dup, 1},
		},
		{
			b: newBuilder().Tuple(func(tb *TupleBuilder) {
				tb.PushdataByte(txvm.IntCode)
				tb.PushdataInt64(32)
			}),
			want: []byte{op.MinPushdata + 1, txvm.IntCode, op.MinPushdata + 1, 32, op.Int, 2, op.Tuple},
		},
		{
			b: newBuilder().Tuple(func(outer *TupleBuilder) {
				outer.Tuple(func(inner *TupleBuilder) {
					inner.PushdataByte(txvm.IntCode)
					inner.PushdataInt64(32)
				})
				outer.Tuple(func(inner *TupleBuilder) {
					inner.PushdataByte(txvm.IntCode)
					inner.PushdataInt64(10)
				})
			}),
			want: []byte{
				op.MinPushdata + 1, txvm.IntCode, op.MinPushdata + 1, 32, op.Int, 2, op.Tuple,
				op.MinPushdata + 1, txvm.IntCode, 10, 2, op.Tuple,
				2, op.Tuple,
			},
		},
	}

	for _, tc := range cases {
		got := tc.b.Build()
		if !bytes.Equal(got, tc.want) {
			t.Errorf("got %x, want %x", got, tc.want)
		}
	}
}
