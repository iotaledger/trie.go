package main

import (
	"fmt"

	"github.com/iotaledger/trie.go/models/trie_blake2b"
	"github.com/iotaledger/trie.go/trie"
)

var data = []string{"", "a", "abc", "abcd", "b", "abd", "klmn", "oprst", "ab", "bcd"}

func main() {
	// create empty store where trie nodes will be stored
	store := trie.NewInMemoryKVStore()

	// create blake2b 20 bytes (160 bit) commitment model for binary trie
	model := trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize160)

	// create the trie with binary keys
	tr := trie.New(model, store, nil)
	fmt.Printf("\nExample of trie.\n%s\n", tr.Info())

	// add data key/value pairs to the trie
	for _, s := range data {
		fmt.Printf("add key '%s' into the trie\n", s)
		tr.Update([]byte(s), []byte(s+"$"))
	}
	// recalculate commitments in the trie
	tr.Commit()
	rootCommitment := trie.RootCommitment(tr)
	fmt.Printf("root commitment 1: %s\n", rootCommitment)

	// currently, the trie is partially cached
	// Persist all cached mutations to the store
	tr.PersistMutations(store)

	// Clear the cache in the trie
	tr.ClearCache()

	// create another trie on the same store
	tr2 := trie.New(model, store, nil)

	// the root must be the same
	rootCommitment2 := trie.RootCommitment(tr2)
	fmt.Printf("root commitment 2: %s\n", rootCommitment2)

	fmt.Printf("roo1 == root2: %v\n", model.EqualCommitments(rootCommitment, rootCommitment2))
}
