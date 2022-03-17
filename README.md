art
====
A simple implementation of Adaptive Radix Tree (ART) in Go. Created for use in personal projects.

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
    value := tree.Search(art.Key("Hi, I'm Key"))
    fmt.Println("value=", value)

    // Delete
    tree.Insert([]byte("wonderful"), "life")
    tree.Insert([]byte("foo"), "bar")
    deleted := tree.Delete([]byte("foo"))
    fmt.Println("deleted=", deleted)

    // Search
    value := tree.Search([]byte("hello"))
    fmt.Println("value=", value)

    // Traverse (with callback function)
    tree.Each(func(node art.Node) {
        fmt.Println("value=", node.Value())
    })

    // Iterator
    for it := tree.Iterator(); it.HasNext(); {
        value := it.Next()
        fmt.Println("value=", value.Value())
    }
}
```

# Benchmarks

```go
goos: darwin
goarch: amd64
pkg: github.com/arriqaaq/art
cpu: Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz

BenchmarkWordsTreeInsert
BenchmarkWordsTreeInsert-16    	      13	  86533062 ns/op	47637452 B/op	 1617223 allocs/op
BenchmarkWordsTreeSearch
BenchmarkWordsTreeSearch-16    	      38	  33476213 ns/op	       0 B/op	       0 allocs/op
BenchmarkUUIDsTreeInsert
BenchmarkUUIDsTreeInsert-16    	      19	  56799013 ns/op	20896024 B/op	  607520 allocs/op
BenchmarkUUIDsTreeSearch
BenchmarkUUIDsTreeSearch-16    	      34	  34188859 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/arriqaaq/art	6.333s
```

# References

- [The Adaptive Radix Tree: ARTful Indexing for Main-Memory Databases (Specification)](http://www-db.in.tum.de/~leis/papers/ART.pdf)
- [Kelly Dunn implementation of the Adaptive Radix Tree](https://github.com/kellydunn/go-art)
- [Beating hash tables with trees? The ART-ful radix trie](https://www.the-paper-trail.org/post/art-paper-notes/)
