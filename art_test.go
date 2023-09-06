package art

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"testing"

	"github.com/armon/go-radix"
	"github.com/arriqaaq/skiplist"
	"github.com/stretchr/testify/assert"
)

var (
	emptyKey = []byte("")
)

func TestNode4AddChild4PreserveSorted(t *testing.T) {
	n := newNode4().innerNode

	for i := 4; i > 0; i-- {
		n.addChild(byte(i), newNode4())
	}

	if n.size < 4 {
		t.Error("incorrect size after adding one child to empty Node4")
	}

	expectedKeys := []byte{1, 2, 3, 4}
	if !bytes.Equal(n.keys, expectedKeys) {
		t.Error("Unexpected key sequence")
	}
}

func TestNode16AddChild16PreserveSorted(t *testing.T) {
	n := newNode16().innerNode
	for i := 16; i > 0; i-- {
		n.addChild(byte(i), newNode4())
	}

	if n.size < 16 {
		t.Error("incorrect size after adding one child to empty Node4")
	}

	for i := 0; i < 16; i++ {
		if n.keys[i] != byte(i+1) {
			t.Error("Unexpected key sequence")
		}
	}
}

func TestGrow(t *testing.T) {
	nodes := []*Node{newNode4(), newNode16(), newNode48()}
	expectedTypes := []int{Node16, Node48, Node256}

	for i, n := range nodes {
		n.innerNode.grow()
		if n.Type() != expectedTypes[i] {
			fmt.Println("type: ", n.Type())
			t.Error("Unexpected node type after growing")
		}
	}
}

func TestShrink(t *testing.T) {
	nodes := []*Node{newNode256(), newNode48(), newNode16(), newNode4()}
	expectedTypes := []int{Node48, Node16, Node4, Leaf}

	for i, n := range nodes {
		in := n.innerNode
		for j := 0; j < in.minSize(); j++ {
			if n.Type() != Node4 {
				in.addChild(byte(i), newNode4())
			} else {
				in.addChild(byte(i), newLeafNode(emptyKey, nil))
			}
		}

		n.shrink()
		if n.Type() != expectedTypes[i] {
			t.Error("Unexpected node type after shrinking")
		}
	}
}

func TestTreeInsert(t *testing.T) {
	tree := NewTree()
	tree.Insert([]byte("hello"), "world")
	if tree.root == nil {
		t.Error("Tree root should not be nil after insterting.")
	}

	if tree.size != 1 {
		t.Error("Unexpected size after inserting.")
	}

	if tree.root.Type() != Leaf {
		t.Error("Unexpected node type for root after a single insert.")
	}
}

func TestTreeInsertAndSearch(t *testing.T) {
	tree := NewTree()

	tree.Insert([]byte("hello"), "world")
	res := tree.Search([]byte("hello"))

	if res != "world" {
		t.Error("Unexpected search result.")
	}
}

func TestTreeInsert2AndSearch(t *testing.T) {
	tree := NewTree()

	tree.Insert([]byte("hello"), "world")
	tree.Insert([]byte("yo"), "earth")
	tree.Insert([]byte("yolo"), "earth")
	tree.Insert([]byte("yol"), "earth")
	tree.Insert([]byte("yoli"), "earth")
	tree.Insert([]byte("yopo"), "earth")

	if res := tree.Search([]byte("yo")); res != "earth" {
		t.Error("unexpected search result")
	}

	if res := tree.Search([]byte("yolo")); res != "earth" {
		t.Error("unexpected search result")
	}

	if res := tree.Search([]byte("yoli")); res != "earth" {
		t.Error("unexpected search result")
	}
}

// An Art Node with a similar prefix should be split into new nodes accordingly
// And should be searchable as intended.
func TestTreeInsert3AndSearchWords(t *testing.T) {
	tree := NewTree()

	searchTerms := []string{"A", "a", "aa"}

	for i := range searchTerms {
		tree.Insert([]byte(searchTerms[i]), searchTerms[i])
	}

	for i := range searchTerms {
		if res := tree.Search([]byte(searchTerms[i])); res != searchTerms[i] {
			t.Error("unexpected search result")
		}
	}
}

func TestTreeInsert5AndRootShouldBeNode16(t *testing.T) {
	tree := NewTree()

	for i := 0; i < 5; i++ {
		tree.Insert([]byte{byte(i)}, "data")
	}

	if tree.root.Type() != Node16 {
		t.Error("Unexpected root value after inserting past Node4 Maximum")
	}
}

func TestTreeInsert17AndRootShouldBeNode48(t *testing.T) {
	tree := NewTree()

	for i := 0; i < 17; i++ {
		tree.Insert([]byte{byte(i)}, "data")
	}

	if tree.root.Type() != Node48 {
		t.Error("Unexpected root value after inserting past Node16 Maximum")
	}
}

func TestTreeInsert49AndRootShouldBeNode256(t *testing.T) {
	tree := NewTree()

	for i := 0; i < 49; i++ {
		tree.Insert([]byte{byte(i)}, "data")
	}

	if tree.root.Type() != Node256 {
		t.Error("Unexpected root value after inserting past Node16 Maximum")
	}
}

func TestInsertManyWordsAndEnsureSearchResultAndMinimumMaximum(t *testing.T) {
	tree := NewTree()

	file, err := os.Open("test/words.txt")
	if err != nil {
		t.Error("Couldn't open words.txt")
	}

	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		if line, err := reader.ReadBytes('\n'); err != nil {
			break
		} else {
			tree.Insert([]byte(line), []byte(line))
		}
	}

	file.Seek(0, 0)

	for {
		if line, err := reader.ReadBytes(byte('\n')); err != nil {
			break
		} else {
			res := tree.Search([]byte(line))

			if res == nil {
				t.Error("Unexpected nil value for search result")
			}

			if res == nil {
				t.Error("Expected payload for element in tree")
			}

			if !bytes.Equal(res.([]byte), []byte(line)) {
				t.Errorf("Incorrect value for node %v.", []byte(line))
			}
		}
	}

	// TODO find a better way of testing the words without slurping up the newline character
	minimum := tree.root.minimum()
	if !bytes.Equal(minimum.Value().([]byte), []byte("A\n")) {
		t.Error("Unexpected Minimum node.")
	}

	maximum := tree.root.maximum()
	if !bytes.Equal(maximum.Value().([]byte), []byte("zythum\n")) {
		t.Error("Unexpected Maximum node.")
	}
}

// After inserting many random UUIDs into the tree, we should be able to successfully retreive all of them
// To ensure their presence in the tree.
func TestInsertManyUUIDsAndEnsureSearchAndMinimumMaximum(t *testing.T) {
	tree := NewTree()

	file, err := os.Open("test/uuid.txt")
	if err != nil {
		t.Error("Couldn't open uuid.txt")
	}

	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		if line, err := reader.ReadBytes('\n'); err != nil {
			break
		} else {
			tree.Insert([]byte(line), []byte(line))
		}
	}

	file.Seek(0, 0)

	for {
		if line, err := reader.ReadBytes(byte('\n')); err != nil {
			break
		} else {
			res := tree.Search([]byte(line))

			if res == nil {
				t.Error("Unexpected nil value for search result")
			}

			if res == nil {
				t.Error("Expected payload for element in tree")
			}

			if !bytes.Equal(res.([]byte), []byte(line)) {
				t.Errorf("Incorrect value for node %v.", []byte(line))
			}
		}
	}

	// TODO find a better way of testing the words without slurping up the newline character
	minimum := tree.root.minimum()
	if !bytes.Equal(minimum.Value().([]byte), []byte("00026bda-e0ea-4cda-8245-522764e9f325\n")) {
		t.Error("Unexpected Minimum node.")
	}

	maximum := tree.root.maximum()
	if !bytes.Equal(maximum.Value().([]byte), []byte("ffffcb46-a92e-4822-82af-a7190f9c1ec5\n")) {
		t.Error("Unexpected Maximum node.")
	}
}

// Inserting a single value into the tree and removing it should result in a nil tree root.
func TestInsertAndRemove1(t *testing.T) {
	tree := NewTree()

	tree.Insert([]byte("test"), []byte("data"))

	tree.Delete([]byte("test"))

	if tree.size != 0 {
		t.Error("Unexpected tree size after inserting and removing")
	}

	if tree.root != nil {
		t.Error("Unexpected root node after inserting and removing")
	}
}

// Inserting Two values into the tree and removing one of them
// should result in a tree root of type LEAF
func TestInsert2AndRemove1AndRootShouldBeLeafNode(t *testing.T) {
	tree := NewTree()

	tree.Insert([]byte("test"), []byte("data"))
	tree.Insert([]byte("test2"), []byte("data"))

	tree.Delete([]byte("test"))

	if tree.size != 1 {
		t.Error("Unexpected tree size after inserting and removing")
	}

	if tree.root == nil || tree.root.Type() != Leaf {
		t.Error("Unexpected root node after inserting and removing")
	}
}

// Inserting Two values into a tree and deleting them both
// should result in a nil tree root
// This tests the expansion of the root into a NODE4 and
// successfully collapsing into a LEAF and then nil upon successive removals
func TestInsert2AndRemove2AndRootShouldBeNil(t *testing.T) {
	tree := NewTree()

	tree.Insert([]byte("test"), []byte("data"))
	tree.Insert([]byte("test2"), []byte("data"))

	tree.Delete([]byte("test"))
	tree.Delete([]byte("test2"))

	if tree.size != 0 {
		t.Error("Unexpected tree size after inserting and removing")
	}

	if tree.root != nil {
		t.Error("Unexpected root node after inserting and removing")
	}
}

// Inserting Five values into a tree and deleting one of them
// should result in a tree root of type NODE4
// This tests the expansion of the root into a NODE16 and
// successfully collapsing into a NODE4 upon successive removals
func TestInsert5AndRemove1AndRootShouldBeNode4(t *testing.T) {
	tree := NewTree()

	for i := 0; i < 5; i++ {
		tree.Insert([]byte{byte(i)}, []byte{byte(i)})
	}

	tree.Delete([]byte{1})
	res := (tree.root.innerNode.findChild(byte(1)))
	if res != nil {
		t.Error("Did not expect to find child after removal")
	}

	if tree.size != 4 {
		t.Error("Unexpected tree size after inserting and removing")
	}

	if tree.root == nil || tree.root.Type() != Node4 {
		t.Error("Unexpected root node after inserting and removing")
	}
}

// Inserting Five values into a tree and deleting all of them
// should result in a tree root of type nil
// This tests the expansion of the root into a NODE16 and
// successfully collapsing into a NODE4, Leaf, then nil
func TestInsert5AndRemove5AndRootShouldBeNil(t *testing.T) {
	tree := NewTree()

	for i := 0; i < 5; i++ {
		tree.Insert([]byte{byte(i)}, []byte{byte(i)})
	}

	for i := 0; i < 5; i++ {
		tree.Delete([]byte{byte(i)})
	}

	if tree.root != nil {
		t.Error("Unexpected root node after inserting and removing")
	}
}

// Inserting 17 values into a tree and deleting one of them should
// result in a tree root of type NODE16
// This tests the expansion of the root into a NODE48, and
// successfully collapsing into a NODE16
func TestInsert17AndRemove1AndRootShouldBeNode16(t *testing.T) {
	tree := NewTree()

	for i := 0; i < 17; i++ {
		tree.Insert([]byte{byte(i)}, []byte{byte(i)})
	}

	tree.Delete([]byte{2})
	res := (tree.root.innerNode.findChild(byte(2)))
	if res != nil {
		t.Error("Did not expect to find child after removal")
	}

	if tree.size != 16 {
		t.Error("Unexpected tree size after inserting and removing")
	}

	if tree.root == nil || tree.root.Type() != Node16 {
		t.Error("Unexpected root node after inserting and removing")
	}
}

// Inserting 17 values into a tree and removing them all should
// result in a tree of root type nil
// This tests the expansion of the root into a NODE48, and
// successfully collapsing into a NODE16, NODE4, Leaf, and then nil
func TestInsert17AndRemove17AndRootShouldBeNil(t *testing.T) {
	tree := NewTree()

	for i := 0; i < 17; i++ {
		tree.Insert([]byte{byte(i)}, []byte{byte(i)})
	}

	for i := 0; i < 17; i++ {
		tree.Delete([]byte{byte(i)})
	}

	if tree.root != nil {
		t.Error("Unexpected root node after inserting and removing")
	}
}

// Inserting 49 values into a tree and removing one of them should
// result in a tree root of type NODE48
// This tests the expansion of the root into a NODE256, and
// successfully collapasing into a NODE48
func TestInsert49AndRemove1AndRootShouldBeNode48(t *testing.T) {
	tree := NewTree()

	for i := 0; i < 49; i++ {
		tree.Insert([]byte{byte(i)}, []byte{byte(i)})
	}

	tree.Delete([]byte{2})
	res := (tree.root.innerNode.findChild(byte(2)))
	if res != nil {
		t.Error("Did not expect to find child after removal")
	}

	if tree.size != 48 {
		t.Error("Unexpected tree size after inserting and removing")
	}

	if tree.root == nil || tree.root.Type() != Node48 {
		t.Error("Unexpected root node after inserting and removing")
	}
}

// Inserting 49 values into a tree and removing all of them should
// result in a nil tree root
// This tests the expansion of the root into a NODE256, and
// successfully collapsing into a Node48, Node16, Node4, Leaf, and finally nil
func TestInsert49AndRemove49AndRootShouldBeNil(t *testing.T) {
	tree := NewTree()

	for i := 0; i < 49; i++ {
		tree.Insert([]byte{byte(i)}, []byte{byte(i)})
	}

	for i := 0; i < 49; i++ {
		tree.Delete([]byte{byte(i)})
	}

	if tree.root != nil {
		t.Error("Unexpected root node after inserting and removing")
	}
}

// A traversal of the tree should be in preorder
func TestEachPreOrderness(t *testing.T) {
	tree := NewTree()
	tree.Insert([]byte("1"), []byte("1"))
	tree.Insert([]byte("2"), []byte("2"))

	traversal := []*Node{}

	tree.Each(func(node *Node) {
		traversal = append(traversal, node)
	})

	// Order should be Node4, 1, 2
	if traversal[0] != tree.root || traversal[0].Type() != Node4 {
		t.Error("Unexpected node at begining of traversal")
	}

	if !bytes.Equal(traversal[1].leaf.key, append([]byte("1"), 0)) || traversal[1].Type() != Leaf {
		t.Error("Unexpected node at second element of traversal")
	}

	if !bytes.Equal(traversal[2].leaf.key, append([]byte("2"), 0)) || traversal[2].Type() != Leaf {
		t.Error("Unexpected node at third element of traversal")
	}
}

// A traversal of a Node48 node should preserve order
// And traverse in the same way for all other nodes.
// Node48s do not store their children in order, and require different logic to traverse them
// so we must test that logic seperately.
func TestEachNode48(t *testing.T) {
	tree := NewTree()

	for i := 48; i > 0; i-- {
		tree.Insert([]byte{byte(i)}, []byte{byte(i)})
	}

	traversal := []*Node{}

	tree.Each(func(node *Node) {
		traversal = append(traversal, node)
	})

	// Order should be Node48, then the rest of the keys in sorted order
	if traversal[0] != tree.root || traversal[0].Type() != Node48 {
		t.Error("Unexpected node at begining of traversal")
	}

	for i := 1; i < 48; i++ {
		if !bytes.Equal(traversal[i].leaf.key, append([]byte{byte(i)}, 0)) || traversal[i].Type() != Leaf {
			t.Error("Unexpected node at second element of traversal")
		}
	}
}

// After inserting many values into the tree, we should be able to iterate through all of them
// And get the expected number of nodes.
func TestEachFullIterationExpectCountOfAllTypes(t *testing.T) {
	tree := NewTree()

	file, err := os.Open("test/words.txt")
	if err != nil {
		t.Error("Couldn't open words.txt")
	}

	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		if line, err := reader.ReadBytes('\n'); err != nil {
			break
		} else {
			tree.Insert([]byte(line), []byte(line))
		}
	}

	var leafCount int = 0
	var node4Count int = 0
	var node16Count int = 0
	var node48Count int = 0
	var node256Count int = 0

	tree.Each(func(node *Node) {
		switch node.Type() {
		case Node4:
			node4Count++
		case Node16:
			node16Count++
		case Node48:
			node48Count++
		case Node256:
			node256Count++
		case Leaf:
			leafCount++
		default:
		}
	})

	if leafCount != 235886 {
		t.Error("Did not correctly count all leaf nodes during traversal")
	}

	if node4Count != 111616 {
		t.Error("Did not correctly count all node4 nodes during traversal")
	}

	if node16Count != 12181 {
		t.Error("Did not correctly count all node16 nodes during traversal")
	}

	if node48Count != 458 {
		t.Error("Did not correctly count all node48 nodes during traversal")
	}

	if node256Count != 1 {
		t.Error("Did not correctly count all node256 nodes during traversal")
	}
}

// After Inserting many values into the tree, we should be able to remove them all
// And expect nothing to exist in the tree.
func TestInsertManyWordsAndRemoveThemAll(t *testing.T) {
	tree := NewTree()

	file, err := os.Open("test/words.txt")
	if err != nil {
		t.Error("Couldn't open words.txt")
	}

	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		if line, err := reader.ReadBytes('\n'); err != nil {
			break
		} else {
			tree.Insert([]byte(line), []byte(line))
		}
	}

	file.Seek(0, 0)

	for {
		if line, err := reader.ReadBytes('\n'); err != nil {
			break
		} else {
			tree.Delete([]byte(line))
		}
	}

	if tree.size != 0 {
		t.Error("Tree is not empty after adding and removing many words")
	}

	if tree.root != nil {
		t.Error("Tree is expected to be nil after removing many words")
	}
}

// After Inserting many values into the tree, we should be able to remove them all
// And expect nothing to exist in the tree.
func TestInsertManyUUIDsAndRemoveThemAll(t *testing.T) {
	tree := NewTree()

	file, err := os.Open("test/uuid.txt")
	if err != nil {
		t.Error("Couldn't open uuid.txt")
	}

	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		if line, err := reader.ReadBytes('\n'); err != nil {
			break
		} else {
			tree.Insert([]byte(line), []byte(line))
		}
	}

	file.Seek(0, 0)

	for {
		if line, err := reader.ReadBytes('\n'); err != nil {
			break
		} else {
			tree.Delete([]byte(line))
		}
	}

	if tree.size != 0 {
		t.Error("Tree is not empty after adding and removing many uuids")
	}

	if tree.root != nil {
		t.Error("Tree is expected to be nil after removing many uuids")
	}
}

func TestInsertWithSameByteSliceAddress(t *testing.T) {
	rand.Seed(42)
	key := make([]byte, 8)
	tree := NewTree()

	// Keep track of what we inserted
	keys := make(map[string]bool)

	for i := 0; i < 135; i++ {
		binary.BigEndian.PutUint64(key, uint64(rand.Int63()))
		tree.Insert(key, key)

		// Ensure that we can search these records later
		keys[string(key)] = true
	}

	if tree.size != uint64(len(keys)) {
		t.Errorf("Mismatched size of tree and expected values.  Expected: %d.  Actual: %d\n", len(keys), tree.size)
	}

	for k, _ := range keys {
		n := tree.Search([]byte(k))
		if n == nil {
			t.Errorf("Did not find entry for key: %v\n", []byte(k))
		}
	}
}

func TestTreeIterator(t *testing.T) {
	tree := NewTree()
	tree.Insert([]byte("2"), []byte{2})
	tree.Insert([]byte("1"), []byte{1})

	it := tree.Iterator()
	assert.NotNil(t, it)

	n := it.Next()
	assert.Equal(t, Node4, n.Type())

	n = it.Next()
	assert.Equal(t, Leaf, n.Type())
	assert.Equal(t, n.leaf.value, []byte{1})

	n = it.Next()
	assert.Equal(t, Leaf, n.Type())
	assert.Equal(t, n.leaf.value, []byte{2})

	assert.False(t, it.HasNext())
	n = it.Next()
	assert.Nil(t, n)
}

// https://github.com/plar/go-adaptive-radix-tree/blob/e4cdd437992f3811b732416fd668ba6209db61a8/tree_test.go#L601
func TestTreeTraversalPrefix(t *testing.T) {
	dataSet := []struct {
		keyPrefix string
		keys      []string
		expected  []string
	}{
		{
			"",
			[]string{},
			[]string{},
		},
		{
			"api",
			[]string{"api.foo.bar", "api.foo.baz", "api.foe.fum", "abc.123.456", "api.foo", "api"},
			[]string{"api.foo.bar", "api.foo.baz", "api.foe.fum", "api.foo", "api"},
		},
		{
			"a",
			[]string{"api.foo.bar", "api.foo.baz", "api.foe.fum", "abc.123.456", "api.foo", "api"},
			[]string{"api.foo.bar", "api.foo.baz", "api.foe.fum", "abc.123.456", "api.foo", "api"},
		},
		{
			"b",
			[]string{"api.foo.bar", "api.foo.baz", "api.foe.fum", "abc.123.456", "api.foo", "api"},
			[]string{},
		},
		{
			"api.",
			[]string{"api.foo.bar", "api.foo.baz", "api.foe.fum", "abc.123.456", "api.foo", "api"},
			[]string{"api.foo.bar", "api.foo.baz", "api.foe.fum", "api.foo"},
		},
		{
			"api.foo.bar",
			[]string{"api.foo.bar", "api.foo.baz", "api.foe.fum", "abc.123.456", "api.foo", "api"},
			[]string{"api.foo.bar"},
		},
		{
			"api.end",
			[]string{"api.foo.bar", "api.foo.baz", "api.foe.fum", "abc.123.456", "api.foo", "api"},
			[]string{},
		},
		{
			"",
			[]string{"api.foo.bar", "api.foo.baz", "api.foe.fum", "abc.123.456", "api.foo", "api"},
			[]string{"api.foo.bar", "api.foo.baz", "api.foe.fum", "abc.123.456", "api.foo", "api"},
		},
		{
			"this:key:has",
			[]string{
				"this:key:has:a:long:prefix:3",
				"this:key:has:a:long:common:prefix:2",
				"this:key:has:a:long:common:prefix:1",
			},
			[]string{
				"this:key:has:a:long:prefix:3",
				"this:key:has:a:long:common:prefix:2",
				"this:key:has:a:long:common:prefix:1",
			},
		},
		{
			"ele",
			[]string{"elector", "electibles", "elect", "electible"},
			[]string{"elector", "electibles", "elect", "electible"},
		},
		{
			"long.api.url.v1",
			[]string{"long.api.url.v1.foo", "long.api.url.v1.bar", "long.api.url.v2.foo"},
			[]string{"long.api.url.v1.foo", "long.api.url.v1.bar"},
		},
	}

	for _, d := range dataSet {
		tree := NewTree()
		for _, k := range d.keys {
			tree.Insert([]byte(k), string(k))
		}

		actual := []string{}
		leafFilter := func(n *Node) {
			if n.IsLeaf() {
				actual = append(actual, string(n.Key()))
			}
		}
		tree.Scan([]byte(d.keyPrefix), leafFilter)

		sort.Strings(d.expected)
		sort.Strings(actual)
		assert.Equal(t, len(d.expected), len(actual))
		assert.Equal(t, d.expected, actual, d.keyPrefix)
	}
}

func TestWordsWithPrefix(t *testing.T) {
	tree := NewTree()

	file, err := os.Open("test/words.txt")
	if err != nil {
		t.Error("Couldn't open words.txt")
	}

	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		if line, err := reader.ReadBytes('\n'); err != nil {
			break
		} else {
			tree.Insert([]byte(line), []byte(line))
		}
	}

	actual := []string{}
	leafFilter := func(n *Node) {
		if n.IsLeaf() {
			actual = append(actual, string(n.Key()))
		}
	}
	tree.Scan([]byte("aa"), leafFilter)

	if len(actual) != 6 {
		fmt.Println(len(actual))
		t.Error("Tree is not empty after adding and removing many words")
	}
}

func TestWordsWithPrefixOutOfRange(t *testing.T) {
	tree := NewTree()

	file, err := os.Open("test/words.txt")
	if err != nil {
		t.Error("Couldn't open words.txt")
	}

	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		if line, err := reader.ReadBytes('\n'); err != nil {
			break
		} else {
			tree.Insert([]byte(line), []byte(line))
		}
	}

	actual := []string{}
	leafFilter := func(n *Node) {
		if n.IsLeaf() {
			actual = append(actual, string(n.Key()))
		}
	}
	tree.Scan([]byte("ZyzzogetonOUTOFRANGE"), leafFilter)

	if len(actual) != 0 {
		fmt.Println(len(actual))
		t.Error("Unexpected item found")
	}
}

func TestEmptyTree(t *testing.T) {
	tree := NewTree()
	testKey := []byte("foo")
	tree.Each(func(n *Node) {})
	tree.Scan([]byte("foo"), func(n *Node) {})
	tree.Search(testKey)
	tree.Delete(testKey)
	assert.False(t, tree.Iterator().HasNext())
}

//
// Benchmarks
//

func loadTestFile(path string) [][]byte {
	file, err := os.Open(path)
	if err != nil {
		panic("Couldn't open " + path)
	}
	defer file.Close()

	var words [][]byte
	reader := bufio.NewReader(file)
	for {
		if line, err := reader.ReadBytes(byte('\n')); err != nil {
			break
		} else {
			if len(line) > 0 {
				words = append(words, line[:len(line)-1])
			}
		}
	}
	return words
}
func BenchmarkWordsArtTreeInsert(b *testing.B) {
	words := loadTestFile("test/words.txt")
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		tree := NewTree()
		for _, w := range words {
			tree.Insert(w, w)
		}
	}
}

func BenchmarkWordsArtTreeSearch(b *testing.B) {
	words := loadTestFile("test/words.txt")
	tree := NewTree()
	for _, w := range words {
		tree.Insert(w, w)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, w := range words {
			tree.Search(w)
		}
	}
}

func BenchmarkUUIDsArtTreeInsert(b *testing.B) {
	words := loadTestFile("test/uuid.txt")
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		tree := NewTree()
		for _, w := range words {
			tree.Insert(w, w)
		}
	}
}

func BenchmarkUUIDsArtTreeSearch(b *testing.B) {
	words := loadTestFile("test/uuid.txt")
	tree := NewTree()
	for _, w := range words {
		tree.Insert(w, w)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, w := range words {
			tree.Search(w)
		}
	}
}

func BenchmarkWordsRadixInsert(b *testing.B) {
	words := loadTestFile("test/words.txt")
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		tree := radix.New()
		for _, w := range words {
			tree.Insert(string(w), string(w))
		}
	}
}

func BenchmarkWordsRadixSearch(b *testing.B) {
	words := loadTestFile("test/words.txt")
	tree := radix.New()
	for _, w := range words {
		tree.Insert(string(w), string(w))
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, w := range words {
			tree.Get(string(w))
		}
	}
}

func BenchmarkWordsSkiplistInsert(b *testing.B) {
	words := loadTestFile("test/words.txt")
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		tree := skiplist.New()
		for _, w := range words {
			tree.Set(string(w), string(w))
		}
	}
}

func BenchmarkWordsSkiplistSearch(b *testing.B) {
	words := loadTestFile("test/words.txt")
	tree := skiplist.New()
	for _, w := range words {
		tree.Set(string(w), string(w))
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, w := range words {
			tree.Get(string(w))
		}
	}
}
