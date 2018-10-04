package merkle

import (
	"encoding/hex"
	"testing"
)

type auditHashT struct {
	Val           string
	RightOperator bool
}

func TestEmptyTree(t *testing.T) {
	root := Root(nil)
	if root != emptyStringHash {
		t.Errorf("root of empty tree should equal %x", emptyStringHash[:])
	}
}

func TestDuplicateLeaves(t *testing.T) {
	items := make([][]byte, 6)
	for i := uint64(0); i < 6; i++ {
		items[i] = []byte{byte(i)}
	}

	// first, get the root of an unbalanced tree
	tItems := [][]byte{items[5], items[4], items[3], items[2], items[1], items[0]}
	root1 := Root(tItems)

	// now, get the root of a balanced tree that repeats leaves 0 and 1
	tItems = [][]byte{items[5], items[4], items[3], items[2], items[1], items[0], items[1], items[0]}
	root2 := Root(tItems)

	if root1 == root2 {
		t.Error("forged merkle tree by duplicating some leaves")
	}
}

func TestAllDuplicateLeaves(t *testing.T) {
	item := []byte{1}
	item1, item2, item3, item4, item5, item6 := item, item, item, item, item, item

	// first, get the root of an unbalanced tree
	items := [][]byte{item6, item5, item4, item3, item2, item1}
	root1 := Root(items)

	// now, get the root of a balanced tree that repeats leaves 5 and 6
	items = [][]byte{item6, item5, item6, item5, item4, item3, item2, item1}
	root2 := Root(items)

	if root1 == root2 {
		t.Error("forged merkle tree with all duplicate leaves")
	}
}

func TestVectors(t *testing.T) {
	cases := []struct {
		input [][]byte
		hex   string
	}{
		{nil, "a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a"},
		{[][]byte{{1}}, "76ab70dc46775b641a8e71507b07145aed11ae5efc0baa94ac06876af2b3bf5c"},
		{[][]byte{{1}, {2}}, "1dad5e07e988e0e446e2cce0b77d2ea44a1801efea272d2e2bc374037a5bc1a8"},
		{[][]byte{{1}, {2}, {3}}, "4f554b3aea550c2f7a86917c8c02a0ee842a813fadec1f4c87569cff27bccd14"},
		{[][]byte{{1}, {2}, {3}, {4}}, "c39898712f54df7e2ace99e3829c100c1aaff45c65312a674ba9e24b37c46bf4"},
		{[][]byte{{1}, {2}, {3}, {4}, {5}}, "49b61513bcc94c883a410c372f7dfa93456aed3c3c23223b0e5962bc44954c92"},
	}
	for _, c := range cases {
		if hash2hex(Root(c.input)) != c.hex {
			t.Errorf("Incorrect hash")
		}
	}
}

func TestTrees(t *testing.T) {
	type in struct {
		data [][]byte
		ind  int
	}
	type want struct {
		res       []auditHashT
		shouldErr bool
	}
	cases := []struct {
		i in
		w want
	}{{
		i: in{
			data: nil,
			ind:  0,
		},
		w: want{
			res:       nil,
			shouldErr: true,
		},
	}, {
		i: in{
			data: [][]byte{{1}},
			ind:  0,
		},
		w: want{
			res:       []auditHashT{},
			shouldErr: false,
		},
	}, {
		i: in{
			data: [][]byte{{1}},
			ind:  1,
		},
		w: want{
			res:       nil,
			shouldErr: true,
		},
	}, {
		i: in{
			data: [][]byte{{1}},
			ind:  2,
		},
		w: want{
			res:       nil,
			shouldErr: true,
		},
	}, {
		i: in{
			data: [][]byte{{1}, {2}},
			ind:  0,
		},
		w: want{
			res: []auditHashT{
				auditHashT{"60de076463ec7a8faaaf56fb815c013378e862b70526b2795eb65ca24025140a", true},
			},
			shouldErr: false,
		},
	}, {
		i: in{
			data: [][]byte{{1}, {2}},
			ind:  1,
		},
		w: want{
			res: []auditHashT{
				auditHashT{"76ab70dc46775b641a8e71507b07145aed11ae5efc0baa94ac06876af2b3bf5c", false},
			},
			shouldErr: false,
		},
	}, {
		i: in{
			data: [][]byte{{1}, {2}, {3}},
			ind:  0,
		},
		w: want{
			res: []auditHashT{
				auditHashT{"60de076463ec7a8faaaf56fb815c013378e862b70526b2795eb65ca24025140a", true},
				auditHashT{"a3f30948550805cb1a32ea5f3f7ede7112a4007b3b1834b4d4c254b8b7d58bd2", true},
			},
			shouldErr: false,
		},
	}, {
		i: in{
			data: [][]byte{{1}, {2}, {3}},
			ind:  1,
		},
		w: want{
			res: []auditHashT{
				auditHashT{"76ab70dc46775b641a8e71507b07145aed11ae5efc0baa94ac06876af2b3bf5c", false},
				auditHashT{"a3f30948550805cb1a32ea5f3f7ede7112a4007b3b1834b4d4c254b8b7d58bd2", true},
			},
			shouldErr: false,
		},
	}, {
		i: in{
			data: [][]byte{{1}, {2}, {3}},
			ind:  2,
		},
		w: want{
			res: []auditHashT{
				auditHashT{"1dad5e07e988e0e446e2cce0b77d2ea44a1801efea272d2e2bc374037a5bc1a8", false},
			},
			shouldErr: false,
		},
	}, {
		i: in{
			data: [][]byte{{1}, {2}, {3}},
			ind:  3,
		},
		w: want{
			res:       nil,
			shouldErr: true,
		},
	}, {
		i: in{
			data: [][]byte{{1}, {2}, {3}, {4}},
			ind:  0,
		},
		w: want{
			res: []auditHashT{
				auditHashT{"60de076463ec7a8faaaf56fb815c013378e862b70526b2795eb65ca24025140a", true},
				auditHashT{"402f1782496bb74c31c484db688b638f4f5fa2cfd585c254f8ccde49295b5163", true},
			},
			shouldErr: false,
		},
	}, {
		i: in{
			data: [][]byte{{1}, {2}, {3}, {4}, {5}},
			ind:  4,
		},
		w: want{
			res: []auditHashT{
				auditHashT{"c39898712f54df7e2ace99e3829c100c1aaff45c65312a674ba9e24b37c46bf4", false},
			},
			shouldErr: false,
		},
	}}

	for _, c := range cases {
		t.Logf("Running test case with input: %+v", c)
		got, err := Proof(c.i.data, c.i.ind)
		didErr := err != nil
		if c.w.shouldErr && !didErr {
			t.Errorf("expected error but got nil")
		}
		if !c.w.shouldErr && didErr {
			t.Errorf("unexpected error: %v", err)
		}
		validate(t, got, c.w.res)
	}
}

func validate(t *testing.T, actual []AuditHash, expected []auditHashT) {
	if len(actual) != len(expected) {
		t.Errorf("the proof length was expected to be %v and was instead %v", len(expected), len(actual))
		return
	}
	for i := range actual {
		if hash2hex(actual[i].Val) != expected[i].Val {
			t.Errorf("the %vth value was expected to be %v and was instead %v", i, expected[i].Val, actual[i].Val)
		}
		if actual[i].RightOperator != expected[i].RightOperator {
			t.Errorf("the %vth rightoperator was expected to be %v and was instead %v", i, expected[i].RightOperator, actual[i].RightOperator)
		}
	}
}

func hash2hex(hash [32]byte) string {
	return hex.EncodeToString(hash[:])
}
