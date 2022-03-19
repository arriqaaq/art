art
====
A simple implementation of Adaptive Radix Tree (ART) in Go. Created for use in personal projects like [FlashDB](https://github.com/arriqaaq/flashdb).

About
=====
- An adaptive radix tree (trie) is useful for efficient indexing in main memory. 
- Its lookup performance surpasses highly tuned, read-only search trees, while supporting very efficient insertions and deletions as well. 
- Space efficient and solves the problem of excessive worst-case space consumption, which plagues most radix trees, by adaptively choosing compact and efficient data structures for internal nodes. 
- Maintains the data in sorted order, which enables additional operations like range scan and prefix lookup.


# Usage

```go
package main

import (
	"fmt"

	"github.com/arriqaaq/art"
)

func main() {
	tree := art.NewTree()

	// Insert
	tree.Insert([]byte("hello"), "world")
	value := tree.Search([]byte("hello"))
	fmt.Println("value=", value)

	// Delete
	tree.Insert([]byte("wonderful"), "life")
	tree.Insert([]byte("foo"), "bar")
	deleted := tree.Delete([]byte("foo"))
	fmt.Println("deleted=", deleted)

	// Search
	value = tree.Search([]byte("hello"))
	fmt.Println("value=", value)

	// Traverse (with callback function)
	tree.Each(func(node *art.Node) {
		if node.IsLeaf() {
			fmt.Println("value=", node.Value())
		}
	})

	// Iterator
	for it := tree.Iterator(); it.HasNext(); {
		value := it.Next()
		if value.IsLeaf() {
			fmt.Println("value=", value.Value())
		}
	}

	// Prefix Scan
	tree.Insert([]byte("api"), "bar")
	tree.Insert([]byte("api.com"), "bar")
	tree.Insert([]byte("api.com.xyz"), "bar")
	leafFilter := func(n *art.Node) {
		if n.IsLeaf() {
			fmt.Println("value=", string(n.Key()))
		}
	}
	tree.Scan([]byte("api"), leafFilter)
}
```

# Benchmarks

Benchmarks are run by inserting a dictionary of 235886 words into each tree.

```go
goos: darwin
goarch: amd64
pkg: github.com/arriqaaq/art
cpu: Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz

// ART tree
BenchmarkWordsArtTreeInsert
BenchmarkWordsArtTreeInsert-16   14	  79622476 ns/op  46379858 B/op	 1604123 allocs/op
BenchmarkWordsArtTreeSearch
BenchmarkWordsArtTreeSearch-16   43	  28123512 ns/op         0 B/op	       0 allocs/op
BenchmarkUUIDsArtTreeInsert
BenchmarkUUIDsArtTreeInsert-16   20	  56691374 ns/op  20404504 B/op	  602400 allocs/op
BenchmarkUUIDsArtTreeSearch
BenchmarkUUIDsArtTreeSearch-16   34	  32183846 ns/op         0 B/op	       0 allocs/op

// Radix tree
BenchmarkWordsRadixInsert
BenchmarkWordsRadixInsert-16     12	  96886770 ns/op  50057340 B/op	 1856741 allocs/op
BenchmarkWordsRadixSearch
BenchmarkWordsRadixSearch-16     33	  40109553 ns/op         0 B/op	       0 allocs/op

// Skiplist
BenchmarkWordsSkiplistInsert
BenchmarkWordsSkiplistInsert-16   4	 271771239 ns/op  32366958 B/op	 1494019 allocs/op
BenchmarkWordsSkiplistSearch
BenchmarkWordsSkiplistSearch-16   8	 135836216 ns/op         0 B/op	       0 allocs/op
```

# References

- [The Adaptive Radix Tree: ARTful Indexing for Main-Memory Databases (Specification)](http://www-db.in.tum.de/~leis/papers/ART.pdf)
- [Kelly Dunn implementation of the Adaptive Radix Tree](https://github.com/kellydunn/go-art)
- [Beating hash tables with trees? The ART-ful radix trie](https://www.the-paper-trail.org/post/art-paper-notes/)
- [Pavel Larkin implementation of ART](https://github.com/plar/go-adaptive-radix-tree)
