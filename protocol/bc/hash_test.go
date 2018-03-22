package bc

import (
	"bytes"
	"testing"
)

func TestHashBytes(t *testing.T) {
	bits := [32]byte{
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	}
	hash := NewHash(bits)

	gotBits := hash.Byte32()
	if !bytes.Equal(bits[:], gotBits[:]) {
		t.Errorf("NewHash(%x).Byte32 = %x", bits, gotBits)
	}

	hash = HashFromBytes(bits[:])
	gotBits = hash.Byte32()
	if !bytes.Equal(bits[:], gotBits[:]) {
		t.Errorf("HashFromBytes(%x).Byte32 = %x", bits, gotBits)
	}
}

func TestHashMarshalText(t *testing.T) {
	hash := NewHash([32]byte{1})
	got, err := hash.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	want := []byte("0100000000000000000000000000000000000000000000000000000000000000")

	if !bytes.Equal(got, want) {
		t.Errorf("MarshalText(%x) = %x want %x", hash.Bytes(), got, want)
	}
}

func TestHashUnmarshalText(t *testing.T) {
	text := []byte("0100000000000000000000000000000000000000000000000000000000000000")
	var got Hash
	err := got.UnmarshalText(text)
	if err != nil {
		t.Fatal(err)
	}
	want := NewHash([32]byte{1})

	if got != want {
		t.Errorf("UnmarshalText(%x) = %x want %x", text, got.Bytes(), want.Bytes())
	}

	text = []byte("short")
	err = got.UnmarshalText(text)
	if err == nil {
		t.Error("expected error for short hash text")
	}
}

func TestHashUnmarshalJSON(t *testing.T) {
	cases := []struct {
		encoded []byte
		want    Hash
		wantErr bool
	}{{
		encoded: []byte("null"),
		want:    Hash{},
	}, {
		encoded: []byte("15"),
		wantErr: true,
	}, {
		encoded: []byte(`"0100000000000000000000000000000000000000000000000000000000000000"`),
		want:    NewHash([32]byte{1}),
	}}

	for _, c := range cases {
		var hash Hash
		err := hash.UnmarshalJSON(c.encoded)
		if err != nil && !c.wantErr {
			t.Errorf("UnmarshalJSON(%s) error = %v want nil", c.encoded, err)
		}

		if hash != c.want {
			t.Errorf("UnmarshalJSON(%s) = %x want %x", c.encoded, hash.Bytes(), c.want.Bytes())
		}
	}
}

func TestHashValue(t *testing.T) {
	hash := NewHash([32]byte{1})
	got, err := hash.Value()
	if err != nil {
		t.Fatal(err)
	}

	want := mustDecodeHex("0100000000000000000000000000000000000000000000000000000000000000")

	gotBytes, ok := got.([]byte)
	if !ok {
		t.Fatal("expected Value to return bytes")
	}

	if !bytes.Equal(gotBytes, want) {
		t.Errorf("Value = %x want %x", got, want)
	}
}

func TestHashScan(t *testing.T) {
	cases := []struct {
		encoded interface{}
		want    Hash
		wantErr bool
	}{{
		encoded: int64(1),
		wantErr: true,
	}, {
		encoded: []byte("short"),
		wantErr: true,
	}, {
		encoded: mustDecodeHex("0100000000000000000000000000000000000000000000000000000000000000"),
		want:    NewHash([32]byte{1}),
	}}

	for _, c := range cases {
		var got Hash
		err := got.Scan(c.encoded)
		if err != nil && !c.wantErr {
			t.Errorf("Scan(%v) error = %v want nil", c.encoded, err)
		}
		if got != c.want {
			t.Errorf("Scan(%v) = %x want %x", c.encoded, got.Bytes(), c.want.Bytes())
		}
	}
}

func TestHashReadFrom(t *testing.T) {
	cases := []struct {
		buf     []byte
		want    Hash
		wantErr bool
	}{{
		buf:     []byte("short"),
		wantErr: true,
	}, {
		buf:  mustDecodeHex("0100000000000000000000000000000000000000000000000000000000000000"),
		want: NewHash([32]byte{1}),
	}}

	for _, c := range cases {
		var buf bytes.Buffer
		buf.Write(c.buf[:])
		var got Hash
		_, err := got.ReadFrom(&buf)
		if err != nil && !c.wantErr {
			t.Errorf("ReadFrom(%x) error = %v want nil", c.buf, err)
		}
		if got != c.want {
			t.Errorf("ReadFrom(%x) = %x want %x", c.buf, got.Bytes(), c.want.Bytes())
		}
	}
}

func TestHashIsZero(t *testing.T) {
	var hash *Hash
	if !hash.IsZero() {
		t.Errorf("IsZero(nil) is false")
	}
	hash = &Hash{}
	if !hash.IsZero() {
		t.Errorf("IsZero(00..) is false")
	}
	val := NewHash([32]byte{1})
	*hash = val
	if hash.IsZero() {
		t.Errorf("IsZero(01..) is true")
	}
}
