/*
	https://db.in.tum.de/~leis/papers/ART.pdf

	Radix trees have a number of interesting properties that
	distinguish them from comparison-based search trees:

	• The height (and complexity) of radix trees depends on
	the length of the keys but in general not on the number
	of elements in the tree.
	• Radix trees require no rebalancing operations and all
	insertion orders result in the same tree.
	• The keys are stored in lexicographic order.
	• The path to a leaf node represents the key of that
	leaf. Therefore, keys are stored implicitly and can be
	reconstructed from paths.

*/

package art

import (
	"bytes"
	_ "math"
	"math/bits"
	_ "os"
)

type (
	meta struct {
		prefix    []byte
		prefixLen int
		size      int
	}

	leafNode struct {
		key   []byte
		value interface{}
	}

	innerNode struct {
		// meta attributes
		meta
		nodeType int
		keys     []byte
		children []*Node
	}
	Node struct {
		// inner nodes map partial keys to child pointers
		innerNode *innerNode

		// leaf is used to store possible leaf
		leaf *leafNode
	}

	Tree struct {
		root *Node
		size uint64
	}

	level struct {
		node     *Node
		childIdx int
	}

	iterator struct {
		tree     *Tree
		nextNode *Node
		levelIdx int
		levels   []*level
	}

	Iterator interface {
		HasNext() bool
		Next() *Node
	}

	Callback func(node *Node)
)

const (
	Node4 = iota
	Node16
	Node48
	Node256
	Leaf

	Node4Min = 2
	Node4Max = 4

	Node16Min = Node4Max + 1
	Node16Max = 16

	Node48Min = Node16Max + 1
	Node48Max = 48

	Node256Min = Node48Max + 1
	Node256Max = 256

	MaxPrefixLen = 10

	nullIdx = -1
)

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func replace(old, new *Node) {
	*old = *new
}

func replaceRef(oldNode **Node, newNode *Node) {
	*oldNode = newNode
}

func copyBytes(dest []byte, src []byte, numBytes int) {
	for i := 0; i < numBytes && i < len(src) && i < len(dest); i++ {
		dest[i] = src[i]
	}
}

func terminate(key []byte) []byte {
	index := bytes.Index(key, []byte{0})
	if index < 0 {
		key = append(key, byte(0))
	}
	return key
}

func newLeafNode(key []byte, value interface{}) *Node {
	newKey := make([]byte, len(key))
	copy(newKey, key)

	newLeaf := &leafNode{newKey, value}
	return &Node{leaf: newLeaf}
}

func newNode4() *Node {
	in := &innerNode{
		nodeType: Node4,
		keys:     make([]byte, Node4Max),
		children: make([]*Node, Node4Max),
		meta: meta{
			prefix: make([]byte, MaxPrefixLen),
		},
	}
	return &Node{innerNode: in}
}

func newNode16() *Node {
	in := &innerNode{
		nodeType: Node16,
		keys:     make([]byte, Node16Max),
		children: make([]*Node, Node16Max),
		meta: meta{
			prefix: make([]byte, MaxPrefixLen),
		},
	}

	return &Node{innerNode: in}
}

func newNode48() *Node {
	in := &innerNode{
		nodeType: Node48,
		keys:     make([]byte, Node256Max),
		children: make([]*Node, Node48Max),
		meta: meta{
			prefix: make([]byte, MaxPrefixLen),
		},
	}

	return &Node{innerNode: in}
}

func newNode256() *Node {
	in := &innerNode{
		nodeType: Node256,
		children: make([]*Node, Node256Max),
		meta: meta{
			prefix: make([]byte, MaxPrefixLen),
		},
	}

	return &Node{innerNode: in}
}

func (n *leafNode) isMatch(key []byte) bool {
	return bytes.Equal(n.key, key)
}

func (n *Node) isFull() bool { return n.innerNode.size == n.maxSize() }

func (n *Node) isLeaf() bool { return n.leaf != nil }

func (n *Node) nodeType() int {
	if n.innerNode != nil {
		return n.innerNode.nodeType
	}
	if n.leaf != nil {
		return Leaf
	}
	return -1
}

func (n *Node) prefixMismatch(key []byte, depth int) int {
	idx := 0
	in := n.innerNode
	p := in.prefix
	for ; idx < in.prefixLen && depth+idx < len(key) && key[depth+idx] == p[idx]; idx++ {
		if idx == MaxPrefixLen-1 {
			min := n.minimum()
			p = min.leaf.key[depth:]
		}
	}
	return idx
}

func (n *Node) index(key byte) int {
	in := n.innerNode

	switch n.nodeType() {
	case Node4:
		for i := 0; i < in.size; i++ {
			if in.keys[i] == key {
				return int(i)
			}
		}
		return -1
	case Node16:
		bitfield := uint(0)
		for i := 0; i < in.size; i++ {
			if in.keys[i] == key {
				bitfield |= (1 << i)
			}
		}
		mask := (1 << in.size) - 1
		bitfield &= uint(mask)
		if bitfield != 0 {
			return bits.TrailingZeros(bitfield)
		}
		return -1
	case Node48:
		index := int(in.keys[key])
		if index > 0 {
			return int(index) - 1
		}

		return -1
	case Node256:
		return int(key)
	}

	return -1
}

func (n *Node) findChild(key byte) **Node {
	if n == nil {
		return nil
	}

	in := n.innerNode
	index := n.index(key)

	switch n.nodeType() {
	case Node4, Node16, Node48:
		if index >= 0 {
			return &in.children[index]
		}

		return nil

	case Node256:
		child := in.children[key]
		if child != nil {
			return &in.children[key]
		}
	}

	return nil
}

func (n *Node) addChild(key byte, node *Node) {
	if n.isFull() {
		n.grow()
		n.addChild(key, node)
		return
	}

	in := n.innerNode

	switch n.nodeType() {
	case Node4:
		idx := 0
		for ; idx < in.size; idx++ {
			if key < in.keys[idx] {
				break
			}
		}

		for i := in.size; i > idx; i-- {
			if in.keys[i-1] > key {
				in.keys[i] = in.keys[i-1]
				in.children[i] = in.children[i-1]
			}
		}

		in.keys[idx] = key
		in.children[idx] = node
		in.size += 1

	case Node16:
		idx := in.size
		bitfield := uint(0)
		for i := 0; i < in.size; i++ {
			if in.keys[i] >= key {
				bitfield |= (1 << i)
			}
		}
		mask := (1 << in.size) - 1
		bitfield &= uint(mask)
		if bitfield != 0 {
			idx = bits.TrailingZeros(bitfield)
		}

		for i := in.size; i > idx; i-- {
			if in.keys[i-1] > key {
				in.keys[i] = in.keys[i-1]
				in.children[i] = in.children[i-1]
			}
		}

		in.keys[idx] = key
		in.children[idx] = node
		in.size += 1

	case Node48:
		idx := 0
		for i := 0; i < len(in.children); i++ {
			if in.children[idx] != nil {
				idx++
			}
		}
		in.children[idx] = node
		in.keys[key] = byte(idx + 1)
		in.size += 1

	case Node256:
		in.children[key] = node
		in.size += 1
	}
}

func (n *Node) deleteChild(key byte) {
	in := n.innerNode

	switch n.nodeType() {
	case Node4, Node16:
		idx := n.index(key)

		in.keys[idx] = 0
		in.children[idx] = nil

		if idx >= 0 {
			for i := idx; i < in.size-1; i++ {
				in.keys[i] = in.keys[i+1]
				in.children[i] = in.children[i+1]
			}

		}

		in.keys[in.size-1] = 0
		in.children[in.size-1] = nil
		in.size -= 1

	case Node48:
		idx := n.index(key)

		if idx >= 0 {
			child := in.children[idx]
			if child != nil {
				in.children[idx] = nil
				in.keys[key] = 0
				in.size -= 1
			}
		}

	case Node256:
		idx := n.index(key)
		child := in.children[idx]
		if child != nil {
			in.children[idx] = nil
			in.size -= 1
		}
	}

	if in.size < n.minSize() {
		n.shrink()
	}
}

func (n *Node) grow() {
	in := n.innerNode

	switch n.nodeType() {
	case Node4:
		n16 := newNode16()
		n16.copyMeta(n)
		n16in := n16.innerNode
		for i := 0; i < in.size; i++ {
			n16in.keys[i] = in.keys[i]
			n16in.children[i] = in.children[i]
		}
		replace(n, n16)

	case Node16:
		n48 := newNode48()
		n48.copyMeta(n)
		n48in := n48.innerNode
		index := 0
		for i := 0; i < in.size; i++ {
			child := in.children[i]
			if child != nil {
				n48in.keys[in.keys[i]] = byte(index + 1)
				n48in.children[index] = child
				index++
			}
		}

		replace(n, n48)

	case Node48:
		n256 := newNode256()
		n256.copyMeta(n)
		n256in := n256.innerNode
		for i := 0; i < len(in.keys); i++ {
			child := (n.findChild(byte(i)))
			if child != nil {
				n256in.children[byte(i)] = *child
			}
		}

		replace(n, n256)

	case Node256:
	}
}

func (n *Node) shrink() {
	in := n.innerNode

	switch n.nodeType() {
	case Node4:
		c := in.children[0]

		if !c.isLeaf() {
			child := c.innerNode
			currentPrefixLen := in.prefixLen

			if currentPrefixLen < MaxPrefixLen {
				in.prefix[currentPrefixLen] = in.keys[0]
				currentPrefixLen++
			}

			if currentPrefixLen < MaxPrefixLen {
				childPrefixLen := min(child.prefixLen, MaxPrefixLen-currentPrefixLen)
				copyBytes(in.prefix[currentPrefixLen:], child.prefix, childPrefixLen)
				currentPrefixLen += childPrefixLen
			}

			copyBytes(child.prefix, in.prefix, min(currentPrefixLen, MaxPrefixLen))
			child.prefixLen += in.prefixLen + 1
		}

		replace(n, c)

	case Node16:
		n4 := newNode4()
		n4.copyMeta(n)
		n4in := n4.innerNode
		n4in.size = 0

		for i := 0; i < len(n4in.keys); i++ {
			n4in.keys[i] = in.keys[i]
			n4in.children[i] = in.children[i]
			n4in.size++
		}

		replace(n, n4)

	case Node48:
		n16 := newNode16()
		n16.copyMeta(n)
		n16in := n16.innerNode
		n16in.size = 0

		for i := 0; i < len(in.keys); i++ {
			idx := in.keys[byte(i)]
			if idx > 0 {
				child := in.children[idx-1]
				if child != nil {
					n16in.children[n16in.size] = child
					n16in.keys[n16in.size] = byte(i)
					n16in.size++
				}
			}
		}

		replace(n, n16)

	case Node256:
		n48 := newNode48()
		n48.copyMeta(n)
		n48in := n48.innerNode
		n48in.size = 0

		for i := 0; i < len(in.children); i++ {
			child := in.children[byte(i)]
			if child != nil {
				n48in.children[n48in.size] = child
				n48in.keys[byte(i)] = byte(n48in.size + 1)
				n48in.size++
			}
		}

		replace(n, n48)
	}
}

func (n *leafNode) longestCommonPrefix(leaf *leafNode, depth int) int {
	limit := min(len(n.key), len(leaf.key)) - depth

	i := 0
	for ; i < limit; i++ {
		if n.key[depth+i] != leaf.key[depth+i] {
			return i
		}
	}
	return i
}

func (n *Node) minSize() int {
	switch n.nodeType() {
	case Node4:
		return Node4Min
	case Node16:
		return Node16Min
	case Node48:
		return Node48Min
	case Node256:
		return Node256Min
	default:
	}
	return 0
}

func (n *Node) maxSize() int {
	switch n.nodeType() {
	case Node4:
		return Node4Max
	case Node16:
		return Node16Max
	case Node48:
		return Node48Max
	case Node256:
		return Node256Max
	default:
	}
	return 0
}

func (n *Node) minimum() *Node {
	in := n.innerNode

	switch n.nodeType() {
	case Node4, Node16:
		return in.children[0].minimum()

	case Node48:
		i := 0
		for in.keys[i] == 0 {
			i++
		}

		child := in.children[in.keys[i]-1]

		return child.minimum()

	case Node256:
		i := 0
		for in.children[i] == nil {
			i++
		}
		return in.children[i].minimum()

	case Leaf:
		return n
	}

	return n
}

func (n *Node) maximum() *Node {
	in := n.innerNode

	switch n.nodeType() {
	case Leaf:
		return n

	case Node4, Node16:
		return in.children[in.size-1].maximum()

	case Node48:
		i := len(in.keys) - 1
		for in.keys[i] == 0 {
			i--
		}

		child := in.children[in.keys[i]-1]
		return child.maximum()

	case Node256:
		i := len(in.children) - 1
		for i > 0 && in.children[byte(i)] == nil {
			i--
		}

		return in.children[i].maximum()
	}

	return n
}

func (n *Node) copyMeta(src *Node) {
	n.innerNode.meta = src.innerNode.meta
}

func (n *Node) Key() []byte {
	if n.nodeType() != Leaf {
		return nil
	}
	return n.leaf.key
}

func (n *Node) Value() interface{} {
	if n.nodeType() != Leaf {
		return nil
	}
	return n.leaf.value
}

func NewTree() *Tree {
	return &Tree{root: nil, size: 0}
}

func (t *Tree) Size() uint64 {
	return t.size
}

func (t *Tree) Search(key []byte) interface{} {
	key = terminate(key)
	return t.search(t.root, key, 0)
}

func (t *Tree) search(current *Node, key []byte, depth int) interface{} {
	for current != nil {
		if current.isLeaf() {
			if current.leaf.isMatch(key) {
				return current.leaf.value
			}
			return nil
		}

		in := current.innerNode
		if current.prefixMismatch(key, depth) != in.prefixLen {
			return nil
		} else {
			depth += in.prefixLen
		}

		v := current.findChild(key[depth])
		if v == nil {
			return nil
		}
		current = *(v)
		depth++
	}

	return nil
}

func (t *Tree) Insert(key []byte, value interface{}) bool {
	key = terminate(key)
	updated := t.insert(&t.root, key, value, 0)
	if !updated {
		t.size++
	}
	return updated
}

func (t *Tree) insert(currentRef **Node, key []byte, value interface{}, depth int) bool {
	current := *currentRef
	if current == nil {
		replaceRef(currentRef, newLeafNode(key, value))
		return false
	}

	if current.isLeaf() {
		if current.leaf.isMatch(key) {
			current.leaf.value = value
			return true
		}

		currentLeaf := current.leaf
		newLeaf := newLeafNode(key, value)
		limit := currentLeaf.longestCommonPrefix(newLeaf.leaf, depth)

		n4 := newNode4()
		n4in := n4.innerNode
		n4in.prefixLen = limit

		copyBytes(n4in.prefix, key[depth:], min(n4in.prefixLen, MaxPrefixLen))

		depth += n4in.prefixLen

		n4.addChild(currentLeaf.key[depth], current)
		n4.addChild(key[depth], newLeaf)
		replaceRef(currentRef, n4)

		return false
	}

	in := current.innerNode
	if in.prefixLen != 0 {
		mismatch := current.prefixMismatch(key, depth)

		if mismatch != in.prefixLen {
			newNode := newNode4()
			replaceRef(currentRef, newNode)
			newNode.innerNode.prefixLen = mismatch

			copyBytes(newNode.innerNode.prefix, in.prefix, mismatch)

			if in.prefixLen < MaxPrefixLen {
				newNode.addChild(in.prefix[mismatch], current)
				in.prefixLen -= (mismatch + 1)
				copyBytes(in.prefix, in.prefix[mismatch+1:], min(in.prefixLen, MaxPrefixLen))
			} else {
				in.prefixLen -= (mismatch + 1)
				minKey := current.minimum().leaf.key
				newNode.addChild(minKey[depth+mismatch], current)
				copyBytes(in.prefix, minKey[depth+mismatch+1:], min(in.prefixLen, MaxPrefixLen))
			}

			newLeafNode := newLeafNode(key, value)
			newNode.addChild(key[depth+mismatch], newLeafNode)

			return false
		}
		depth += in.prefixLen
	}

	next := current.findChild(key[depth])
	if next != nil {
		return t.insert(next, key, value, depth+1)
	}

	current.addChild(key[depth], newLeafNode(key, value))
	return false
}

func (t *Tree) Delete(key []byte) bool {
	key = terminate(key)
	deleted := t.delete(&t.root, key, 0)
	if deleted {
		t.size--
		return true
	}
	return false
}

func (t *Tree) delete(currentRef **Node, key []byte, depth int) bool {
	current := *currentRef
	if current.isLeaf() {
		if current.leaf.isMatch(key) {
			replaceRef(currentRef, nil)
			return true
		}
	} else {
		in := current.innerNode
		if in.prefixLen != 0 {
			mismatch := current.prefixMismatch(key, depth)
			if mismatch != in.prefixLen {
				return false
			}
			depth += in.prefixLen
		}

		next := current.findChild(key[depth])
		if *next != nil {
			if (*next).isLeaf() {
				leaf := (*next).leaf
				if leaf.isMatch(key) {
					current.deleteChild(key[depth])
					return true
				}
			}
		}

		return t.delete(next, key, depth+1)
	}
	return false
}

func (t *Tree) Each(callback Callback) {
	t.each(t.root, callback)
}

func (t *Tree) each(current *Node, callback Callback) {

	callback(current)

	in := current.innerNode

	switch current.nodeType() {
	case Node4, Node16, Node256:
		for i := 0; i < len(in.children); i++ {
			next := in.children[i]
			if next != nil {
				t.each(next, callback)
			}
		}

	case Node48:
		for i := 0; i < len(in.keys); i++ {
			index := in.keys[byte(i)]
			if index > 0 {
				next := in.children[index-1]
				if next != nil {
					t.each(next, callback)
				}
			}
		}

	}
}

// Iterator pattern
func (t *Tree) Iterator() Iterator {
	return &iterator{
		tree:     t,
		nextNode: t.root,
		levelIdx: 0,
		levels:   []*level{{t.root, nullIdx}},
	}
}

func (ti *iterator) HasNext() bool {
	return ti != nil && ti.nextNode != nil
}

func (ti *iterator) Next() *Node {
	if !ti.HasNext() {
		return nil
	}

	cur := ti.nextNode
	ti.next()

	return cur
}

func (ti *iterator) addLevel() {
	newlevel := make([]*level, ti.levelIdx+10)
	copy(newlevel, ti.levels)
	ti.levels = newlevel
}

func nextChild(children []*Node, idx int) (int, *Node) {
	if idx == nullIdx {
		idx = 0
	}
	for i := idx; i < len(children); i++ {
		child := children[i]
		if child != nil {
			return i + 1, child
		}
	}

	return 0, nil
}

func (ti *iterator) next() {
	for {
		var nextNode *Node
		nextIdx := nullIdx

		curNode := ti.levels[ti.levelIdx].node
		curChildIdx := ti.levels[ti.levelIdx].childIdx

		switch curNode.nodeType() {
		case Node4:
			nextIdx, nextNode = nextChild(curNode.innerNode.children, curChildIdx)
		case Node16:
			nextIdx, nextNode = nextChild(curNode.innerNode.children, curChildIdx)
		case Node48:
			node := curNode
			for i := curChildIdx; i < len(node.innerNode.keys); i++ {
				child := node.innerNode.children[node.innerNode.keys[i]]
				if child != nil {
					nextIdx = i + 1
					nextNode = child
					break
				}
			}
		case Node256:
			nextIdx, nextNode = nextChild(curNode.innerNode.children, curChildIdx)
		}

		if nextNode == nil {
			if ti.levelIdx > 0 {
				ti.levelIdx--
			} else {
				ti.nextNode = nil
				return
			}
		} else {
			ti.levels[ti.levelIdx].childIdx = nextIdx
			ti.nextNode = nextNode

			if ti.levelIdx+1 >= cap(ti.levels) {
				ti.addLevel()
			}

			ti.levelIdx++
			ti.levels[ti.levelIdx] = &level{nextNode, nullIdx}
			return
		}
	}
}
