package merkle

import "testing"
import "encoding/hex"

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

func hash2hex(hash [32]byte) string {
	return hex.EncodeToString(hash[:])
}
