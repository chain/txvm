package patricia

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"testing"
	"testing/quick"

	"github.com/chain/txvm/crypto/sha3"
	"github.com/chain/txvm/protocol/bc"
	"github.com/chain/txvm/testutil"
)

func BenchmarkSingleInsert(b *testing.B) {
	baseTree := new(Tree)
	baseTree.Insert(make([]byte, 32))
	tr := new(Tree)
	h := [32]byte{1}
	for i := 0; i < b.N; i++ {
		*tr = *baseTree
		tr.Insert(h[:])
	}
}

func BenchmarkInserts(b *testing.B) {
	const nodes = 10000
	for i := 0; i < b.N; i++ {
		tr := new(Tree)
		for j := uint64(0); j < nodes; j++ {
			var h [32]byte
			binary.LittleEndian.PutUint64(h[:], j)

			err := tr.Insert(h[:])
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkInsertsRootHash(b *testing.B) {
	const nodes = 10000
	for i := 0; i < b.N; i++ {
		tr := new(Tree)
		for j := uint64(0); j < nodes; j++ {
			var h [32]byte
			binary.LittleEndian.PutUint64(h[:], j)

			err := tr.Insert(h[:])
			if err != nil {
				b.Fatal(err)
			}
		}
		tr.RootHash()
	}
}

func TestRootHashBug(t *testing.T) {
	tr := new(Tree)

	err := tr.Insert([]byte{0x94})
	if err != nil {
		t.Fatal(err)
	}
	err = tr.Insert([]byte{0x36})
	if err != nil {
		t.Fatal(err)
	}
	before := tr.RootHash()
	err = tr.Insert([]byte{0xba})
	if err != nil {
		t.Fatal(err)
	}
	if tr.RootHash() == before {
		t.Errorf("before and after root hash is %s", before.String())
	}
}

func TestLeafVsInternalNodes(t *testing.T) {
	tr0 := new(Tree)

	err := tr0.Insert([]byte{0x01})
	if err != nil {
		t.Fatal(err)
	}
	err = tr0.Insert([]byte{0x02})
	if err != nil {
		t.Fatal(err)
	}
	err = tr0.Insert([]byte{0x03})
	if err != nil {
		t.Fatal(err)
	}
	err = tr0.Insert([]byte{0x04})
	if err != nil {
		t.Fatal(err)
	}

	// Force calculation of all the hashes.
	tr0.RootHash()
	t.Logf("first child = %s, %t", tr0.root.children[0].hash, tr0.root.children[0].isLeaf)
	t.Logf("second child = %s, %t", tr0.root.children[1].hash, tr0.root.children[1].isLeaf)

	// Create a second tree using an internal node from tr1.
	tr1 := new(Tree)
	err = tr1.Insert(tr0.root.children[0].hash.Bytes()) // internal node of tr0
	if err != nil {
		t.Fatal(err)
	}
	err = tr1.Insert(tr0.root.children[1].hash.Bytes()) // sibling leaf node of above node ^
	if err != nil {
		t.Fatal(err)
	}

	if tr1.RootHash() == tr0.RootHash() {
		t.Errorf("tr0 and tr1 have matching root hashes: %x", tr1.RootHash().Bytes())
	}
}

func TestRootHashInsertQuickCheck(t *testing.T) {
	tr := new(Tree)

	f := func(b [32]byte) bool {
		before := tr.RootHash()
		err := tr.Insert(b[:])
		if err != nil {
			return false
		}
		return before != tr.RootHash()
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestLookup(t *testing.T) {
	tr := &Tree{
		root: &node{key: bits("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true, keybit: 7},
	}
	got := lookup(tr.root, bits("11111111"))
	if !testutil.DeepEqual(got, tr.root) {
		t.Log("lookup on 1-node tree")
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(tr.root, 0))
	}

	tr = &Tree{
		root: &node{key: bits("11111110"), hash: hashPtr(hashForLeaf(bits("11111110"))), isLeaf: true, keybit: 7},
	}
	got = lookup(tr.root, bits("11111111"))
	if got != nil {
		t.Log("lookup nonexistent key on 1-node tree")
		t.Fatalf("got:\n%swant nil", prettyNode(got, 0))
	}

	tr = &Tree{
		root: &node{
			key:    bits("11110000"),
			keybit: 3,
			hash:   hashPtr(hashForNonLeaf(hashForLeaf(bits("11110000")), hashForLeaf(bits("11111111")))),
			children: [2]*node{
				{key: bits("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true, keybit: 7},
				{key: bits("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true, keybit: 7},
			},
		},
	}
	got = lookup(tr.root, bits("11110000"))
	if !testutil.DeepEqual(got, tr.root.children[0]) {
		t.Log("lookup root's first child")
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(tr.root.children[0], 0))
	}

	tr = &Tree{
		root: &node{
			key:    bits("11110000"),
			keybit: 3,
			hash: hashPtr(hashForNonLeaf(
				hashForLeaf(bits("11110000")),
				hashForNonLeaf(hashForLeaf(bits("11111100")), hashForLeaf(bits("11111111"))),
			)),
			children: [2]*node{
				{key: bits("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true, keybit: 7},
				{
					key:    bits("11111100"),
					keybit: 5,
					hash:   hashPtr(hashForNonLeaf(hashForLeaf(bits("11111100")), hashForLeaf(bits("11111111")))),
					children: [2]*node{
						{key: bits("11111100"), hash: hashPtr(hashForLeaf(bits("11111100"))), isLeaf: true, keybit: 7},
						{key: bits("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true, keybit: 7},
					},
				},
			},
		},
	}
	got = lookup(tr.root, bits("11111100"))
	if !testutil.DeepEqual(got, tr.root.children[1].children[0]) {
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(tr.root.children[1].children[0], 0))
	}
}

func TestContains(t *testing.T) {
	tr := new(Tree)

	if v := bits("00000011"); tr.Contains(v) {
		t.Errorf("expected tree to not contain %x, but did", v)
	}

	tr.Insert(bits("00000011"))
	tr.Insert(bits("00000010"))

	if v := bits("00000011"); !tr.Contains(v) {
		t.Errorf("expected tree to contain %x, but did not", v)
	}
	if v := bits("00000000"); tr.Contains(v) {
		t.Errorf("expected tree to not contain %x, but did", v)
	}
	if v := bits("00000010"); !tr.Contains(v) {
		t.Errorf("expected tree to contain %x, but did not", v)
	}

	tr = new(Tree)
	tr.Insert([]byte{1, 0})
	tr.Insert([]byte{1, 255})

	if v := []byte{1}; tr.Contains(v) {
		t.Errorf("expected tree to not contain %x, but did", v)
	}
}

func TestInsert(t *testing.T) {
	tr := new(Tree)

	tr.Insert(bits("11111111"))
	tr.RootHash()
	want := &Tree{
		root: &node{key: bits("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true, keybit: 7},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		log.Printf("want hash? %x", hashForLeaf(bits("11111111")).Bytes())
		t.Log("insert into empty tree")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111111"))
	tr.RootHash()
	want = &Tree{
		root: &node{key: bits("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true, keybit: 7},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Log("inserting the same key does not modify the tree")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11110000"))
	tr.RootHash()
	want = &Tree{
		root: &node{
			key:    bits("11110000"),
			keybit: 3,
			hash:   hashPtr(hashForNonLeaf(hashForLeaf(bits("11110000")), hashForLeaf(bits("11111111")))),
			children: [2]*node{
				{key: bits("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true, keybit: 7},
				{key: bits("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true, keybit: 7},
			},
		},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Log("different key creates a fork")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111100"))
	tr.RootHash()
	want = &Tree{
		root: &node{
			key:    bits("11110000"),
			keybit: 3,
			hash: hashPtr(hashForNonLeaf(
				hashForLeaf(bits("11110000")),
				hashForNonLeaf(hashForLeaf(bits("11111100")), hashForLeaf(bits("11111111"))),
			)),
			children: [2]*node{
				{key: bits("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true, keybit: 7},
				{
					key:    bits("11111100"),
					keybit: 5,
					hash:   hashPtr(hashForNonLeaf(hashForLeaf(bits("11111100")), hashForLeaf(bits("11111111")))),
					children: [2]*node{
						{key: bits("11111100"), hash: hashPtr(hashForLeaf(bits("11111100"))), isLeaf: true, keybit: 7},
						{key: bits("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true, keybit: 7},
					},
				},
			},
		},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111110"))
	tr.RootHash()
	want = &Tree{
		root: &node{
			key:    bits("11110000"),
			keybit: 3,
			hash: hashPtr(hashForNonLeaf(
				hashForLeaf(bits("11110000")),
				hashForNonLeaf(
					hashForLeaf(bits("11111100")),
					hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111"))),
				),
			)),
			children: [2]*node{
				{key: bits("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true, keybit: 7},
				{
					key:    bits("11111100"),
					keybit: 5,
					hash: hashPtr(hashForNonLeaf(
						hashForLeaf(bits("11111100")),
						hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111"))))),
					children: [2]*node{
						{key: bits("11111100"), hash: hashPtr(hashForLeaf(bits("11111100"))), isLeaf: true, keybit: 7},
						{
							key:    bits("11111110"),
							keybit: 6,
							hash:   hashPtr(hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111")))),
							children: [2]*node{
								{key: bits("11111110"), hash: hashPtr(hashForLeaf(bits("11111110"))), isLeaf: true, keybit: 7},
								{key: bits("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true, keybit: 7},
							},
						},
					},
				},
			},
		},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Log("a fork is created for each level of similar key")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Insert(bits("11111011"))
	tr.RootHash()
	want = &Tree{
		root: &node{
			key:    bits("11110000"),
			keybit: 3,
			hash: hashPtr(hashForNonLeaf(
				hashForLeaf(bits("11110000")),
				hashForNonLeaf(
					hashForLeaf(bits("11111011")),
					hashForNonLeaf(
						hashForLeaf(bits("11111100")),
						hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111"))),
					),
				),
			)),
			children: [2]*node{
				{key: bits("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true, keybit: 7},
				{
					key:    bits("11111011"),
					keybit: 4,
					hash: hashPtr(hashForNonLeaf(
						hashForLeaf(bits("11111011")),
						hashForNonLeaf(
							hashForLeaf(bits("11111100")),
							hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111"))),
						))),
					children: [2]*node{
						{key: bits("11111011"), hash: hashPtr(hashForLeaf(bits("11111011"))), isLeaf: true, keybit: 7},
						{
							key:    bits("11111100"),
							keybit: 5,
							hash: hashPtr(hashForNonLeaf(
								hashForLeaf(bits("11111100")),
								hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111"))),
							)),
							children: [2]*node{
								{key: bits("11111100"), hash: hashPtr(hashForLeaf(bits("11111100"))), isLeaf: true, keybit: 7},
								{
									key:    bits("11111110"),
									keybit: 6,
									hash:   hashPtr(hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111")))),
									children: [2]*node{
										{key: bits("11111110"), hash: hashPtr(hashForLeaf(bits("11111110"))), isLeaf: true, keybit: 7},
										{key: bits("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true, keybit: 7},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Log("compressed branch node is split")
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr = new(Tree)
	tr.Insert([]byte{1, 0})
	tr.Insert([]byte{1, 255})
	err := tr.Insert([]byte{1})
	if err == nil {
		t.Error("expected prefix error for insert")
	}

	tr = new(Tree)
	tr.Insert([]byte{1, 0})
	tr.Insert([]byte{1, 1})
	err = tr.Insert([]byte{1})
	if err == nil {
		t.Error("expected prefix error for insert")
	}

	tr = new(Tree)
	tr.Insert([]byte{0, 1})
	tr.Insert([]byte{1, 0})
	tr.Insert([]byte{1, 1})
	err = tr.Insert([]byte{1, 1, 1})
	if err == nil {
		t.Error("expected prefix error for insert")
	}
}

func TestDelete(t *testing.T) {
	tr := new(Tree)
	tr.root = &node{
		key:    bits("11110000"),
		keybit: 3,
		hash: hashPtr(hashForNonLeaf(
			hashForLeaf(bits("11110000")),
			hashForNonLeaf(
				hashForLeaf(bits("11111100")),
				hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111"))),
			),
		)),
		children: [2]*node{
			{key: bits("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true, keybit: 7},
			{
				key:    bits("11111100"),
				keybit: 5,
				hash: hashPtr(hashForNonLeaf(
					hashForLeaf(bits("11111100")),
					hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111"))),
				)),
				children: [2]*node{
					{key: bits("11111100"), hash: hashPtr(hashForLeaf(bits("11111100"))), isLeaf: true, keybit: 7},
					{
						key:    bits("11111110"),
						keybit: 6,
						hash:   hashPtr(hashForNonLeaf(hashForLeaf(bits("11111110")), hashForLeaf(bits("11111111")))),
						children: [2]*node{
							{key: bits("11111110"), hash: hashPtr(hashForLeaf(bits("11111110"))), isLeaf: true, keybit: 7},
							{key: bits("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true, keybit: 7},
						},
					},
				},
			},
		},
	}

	tr.Delete(bits("11111110"))
	tr.RootHash()
	want := &Tree{
		root: &node{
			key:    bits("11111111"),
			keybit: 3,
			hash: hashPtr(hashForNonLeaf(
				hashForLeaf(bits("11110000")),
				hashForNonLeaf(hashForLeaf(bits("11111100")), hashForLeaf(bits("11111111"))),
			)),
			children: [2]*node{
				{key: bits("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true, keybit: 7},
				{
					key:    bits("11111111"),
					keybit: 5,
					hash:   hashPtr(hashForNonLeaf(hashForLeaf(bits("11111100")), hashForLeaf(bits("11111111")))),
					children: [2]*node{
						{key: bits("11111100"), hash: hashPtr(hashForLeaf(bits("11111100"))), isLeaf: true, keybit: 7},
						{key: bits("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true, keybit: 7},
					},
				},
			},
		},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11111100"))
	tr.RootHash()
	want = &Tree{
		root: &node{
			key:    bits("11111111"),
			keybit: 3,
			hash:   hashPtr(hashForNonLeaf(hashForLeaf(bits("11110000")), hashForLeaf(bits("11111111")))),
			children: [2]*node{
				{key: bits("11110000"), hash: hashPtr(hashForLeaf(bits("11110000"))), isLeaf: true, keybit: 7},
				{key: bits("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true, keybit: 7},
			},
		},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11110011")) // nonexistent value
	tr.RootHash()
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11110000"))
	tr.RootHash()
	want = &Tree{
		root: &node{key: bits("11111111"), hash: hashPtr(hashForLeaf(bits("11111111"))), isLeaf: true, keybit: 7},
	}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}

	tr.Delete(bits("11111111"))
	tr.RootHash()
	want = &Tree{}
	if !testutil.DeepEqual(tr.root, want.root) {
		t.Fatalf("got:\n%swant:\n%s", pretty(tr), pretty(want))
	}
}

func TestDeletePrefix(t *testing.T) {
	root := &node{
		key:    []byte{1, 1},
		keybit: 7,
		hash:   hashPtr(hashForNonLeaf(hashForLeaf([]byte{1, 1, 0}), hashForLeaf([]byte{1, 1, 1}))),
		children: [2]*node{
			{key: []byte{1, 1, 0}, hash: hashPtr(hashForLeaf([]byte{1, 1, 0})), isLeaf: true, keybit: 7},
			{key: []byte{1, 1, 1}, hash: hashPtr(hashForLeaf([]byte{1, 1, 1})), isLeaf: true, keybit: 7},
		},
	}

	got := delete(root, []byte{1})
	got.calcHash()
	if !testutil.DeepEqual(got, root) {
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(root, 0))
	}

	got = delete(root, []byte{1, 1})
	got.calcHash()
	if !testutil.DeepEqual(got, root) {
		t.Fatalf("got:\n%swant:\n%s", prettyNode(got, 0), prettyNode(root, 0))
	}

}

func TestWalk(t *testing.T) {
	var found [][]byte
	f := func(item []byte) error {
		found = append(found, item)
		return nil
	}
	tr := new(Tree)
	err := Walk(tr, f)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Error("expected zero found nodes on empty tree")
	}

	tr.Insert([]byte{0})
	tr.Insert([]byte{1})
	tr.Insert([]byte{2})

	err = Walk(tr, f)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 3 {
		t.Errorf("expected 3 leaf nodes, got %d", len(found))
	}

	f = func(item []byte) error {
		return errors.New("x")
	}
	err = Walk(tr, f)
	if err == nil {
		t.Error("expected error")
	}
}

func TestHasPrefix(t *testing.T) {
	cases := []struct {
		s, pref string
		bit     byte
		want    bool
	}{
		{s: "11111111", pref: "10000000", bit: 0, want: true},
		{s: "11111111", pref: "00000000", bit: 0, want: false},
		{s: "11111111", pref: "11110000", bit: 0, want: true},
		{s: "11111111", pref: "11110000", bit: 1, want: true},
		{s: "11111111", pref: "11110000", bit: 2, want: true},
		{s: "11111111", pref: "11110000", bit: 3, want: true},
		{s: "11111111", pref: "11110000", bit: 4, want: false},
		{s: "11111111", pref: "11111111", bit: 7, want: true},
		{s: "11111111", pref: "11111110", bit: 7, want: false},
	}

	for _, c := range cases {
		got := hasPrefix(bits(c.s), bits(c.pref), c.bit)
		if got != c.want {
			t.Errorf("hasPrefix(%s, %s, %d) = %v want %v", c.s, c.pref, c.bit, got, c.want)
		}
	}
}

func TestRootHashes(t *testing.T) {
	tr := new(Tree)
	tr2 := new(Tree)
	for i := 0; i < 1000; i++ {
		h := make([]byte, 32)
		binary.LittleEndian.PutUint64(h, uint64(i))
		err := tr.Insert(h)
		if err != nil {
			t.Fatal(err)
		}

		hash := sha3.Sum256(h)
		err = tr2.Insert(hash[:])
		if err != nil {
			t.Fatal(err)
		}
	}

	want := mustDecodeHash("438d57e753b849f51cdc69d2779715979dda4b4ce75653305a1d24cafd324fda")
	got := tr.RootHash()
	if got != want {
		t.Errorf("RootHash(0-999) = %x want %x", got.Bytes(), want.Bytes())
	}

	want2 := mustDecodeHash("cb8b32a2823d4c35f697caa5698ca95b9ab043be9db238256e7d38408496681b")
	got2 := tr2.RootHash()
	if got2 != want2 {
		t.Errorf("RootHash(SHA3(0-999)) = %x want %x", got2.Bytes(), want2.Bytes())
	}
}

func pretty(t *Tree) string {
	if t.root == nil {
		return ""
	}
	return prettyNode(t.root, 0)
}

func prettyNode(n *node, depth int) string {
	prettyStr := strings.Repeat("  ", depth)
	if n == nil {
		prettyStr += "nil\n"
		return prettyStr
	}
	var b int
	if len(n.key) > 31*8 {
		b = 31 * 8
	}
	prettyStr += fmt.Sprintf("key=%+v", n.key[b:])
	if n.hash != nil {
		prettyStr += fmt.Sprintf(" hash=%+v", n.hash)
	}
	prettyStr += "\n"

	for _, c := range n.children {
		if c != nil {
			prettyStr += prettyNode(c, depth+1)
		}
	}

	return prettyStr
}

func bits(lit string) []byte {
	var b [31]byte
	n, _ := strconv.ParseUint(lit, 2, 8)
	return append(b[:], byte(n))
}

func hashForLeaf(item []byte) bc.Hash {
	return bc.NewHash(sha3.Sum256(append([]byte{0x00}, item...)))
}

func hashForNonLeaf(a, b bc.Hash) bc.Hash {
	d := []byte{0x01}
	d = append(d, a.Bytes()...)
	d = append(d, b.Bytes()...)
	return bc.NewHash(sha3.Sum256(d))
}

func hashPtr(h bc.Hash) *bc.Hash {
	return &h
}

func mustDecodeHash(str string) bc.Hash {
	dec, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	if len(dec) != 32 {
		panic("bad hash length")
	}
	return bc.HashFromBytes(dec)
}
