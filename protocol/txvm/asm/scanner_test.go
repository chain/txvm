package asm

import (
	"reflect"
	"testing"
)

type scannedToken struct {
	tok token
	lit string
}

func TestScanner(t *testing.T) {
	cases := []struct {
		input string
		want  []scannedToken
	}{
		{
			input: `x"deadbeef"`,
			want:  []scannedToken{{tokHex, `x"deadbeef"`}},
		},
		{
			input: `hello`,
			want:  []scannedToken{{tokIdent, `hello`}},
		},
		{
			input: `-42`,
			want:  []scannedToken{{tokNumber, `-42`}},
		},
		{
			input: `  "hello"`,
			want:  []scannedToken{{tokString, `"hello"`}},
		},
		{
			input: `  'hello'     `,
			want:  []scannedToken{{tokString, `'hello'`}},
		},
		{
			input: `  "  'hello world'  "     `,
			want:  []scannedToken{{tokString, `"  'hello world'  "`}},
		},
		{
			input: `hello world`,
			want: []scannedToken{
				{tokIdent, `hello`},
				{tokIdent, `world`},
			},
		},
		{
			input: `hello 42 "worlds" x'ffff'`,
			want: []scannedToken{
				{tokIdent, `hello`},
				{tokNumber, `42`},
				{tokString, `"worlds"`},
				{tokHex, `x'ffff'`},
			},
		},
		{
			input: `$label: jump:$label`,
			want: []scannedToken{
				{tokLabel, `$label`},
				{tokColon, `:`},
				{tokJump, `jump:`},
				{tokLabel, `$label`},
			},
		},
	}

	for _, tc := range cases {
		var s scanner
		s.initString(tc.input)

		var scanned []scannedToken
		for _, tok, lit := s.scan(); tok != tokEOF; _, tok, lit = s.scan() {
			scanned = append(scanned, scannedToken{
				tok: tok,
				lit: lit,
			})
		}
		if !reflect.DeepEqual(scanned, tc.want) {
			t.Errorf("scanning %q = %#v, want %#v", tc.input, scanned, tc.want)
		}
	}
}
