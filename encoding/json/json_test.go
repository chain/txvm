package json

import (
	"bytes"
	"testing"
)

func TestMarshalJSONMap(t *testing.T) {
	cases := []struct {
		testName string
		input    Map
		want     []byte
	}{
		{
			testName: "test nil map",
			input:    nil,
			want:     []byte("{}"),
		},
		{
			testName: "test empty map",
			input:    Map{},
			want:     []byte("{}"),
		},
		{
			testName: "populated map",
			input:    Map("i am a map"),
			want:     []byte("i am a map"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(t *testing.T) {
			out, err := tc.input.MarshalJSON()

			if err != nil {
				t.Errorf("unexpected err %v", err)
			}

			if !bytes.Equal(out, tc.want) {
				t.Errorf("marshalJSON(%v)=%s, want:%s", tc.input, out, tc.want)
			}
		})
	}
}

func BenchmarkUnmarshalJSONMap(b *testing.B) {
	cases := []struct {
		name, data string
	}{
		{"null", `null`},
		{"empty object", `{}`},
		{"space before object", `   {}`},
		{"single-level object", `{"a":"b", "c":"d"}`},
		{"multi-level object", `{"a":{"b":"c"}, "d": [1,2,3]}`},
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			var m Map
			data := []byte(c.data)
			for i := 0; i < b.N; i++ {
				m.UnmarshalJSON(data)
			}
		})
	}
}
