package main

import (
	"fmt"
	trie_go "github.com/iotaledger/trie.go"
	"github.com/iotaledger/trie.go/trie"
	"github.com/iotaledger/trie.go/trie_blake2b_20"
)

var data = []string{"a", "abc", "abcd", "b", "abd", "klmn", "oprst", "ab", "bcd"}

func main() {
	// create store where trie nodes will be stored
	store := trie_go.NewInMemoryKVStore()

	// create blake2b 20 bytes (160 bit) commitment model
	model := trie_blake2b_20.New()

	// create the trie with binary keys
	tr := trie.New(model, store, trie.PathArity2, false)
	fmt.Printf("\nExample of trie.\n%s\n", tr.Info())

	// add data key/value pairs to the trie
	for _, s := range data {
		fmt.Printf("add key '%s' into the trie\n", s)
		tr.UpdateStr(s, s+"$")
	}
	// recalculate commitments in the trie
	tr.Commit()
	rootCommitment := trie.RootCommitment(tr)
	fmt.Printf("root commitment: %s\n", rootCommitment)
	// remove some keys from the trie
	for i := range []int{1, 5, 6} {
		fmt.Printf("remove key '%s' from the trie\n", data[i])
		tr.DeleteStr(data[i])
	}
	// recalc trie again
	tr.Commit()
	rootCommitment = trie.RootCommitment(tr)
	fmt.Printf("root commitment: %s\n", rootCommitment)

	// check PoI for all data
	for _, s := range data {
		// retrieve proof
		proof := model.Proof([]byte(s), tr)
		fmt.Printf("PoI of the key '%s': length %d, serialized size %d bytes\n",
			s, len(proof.Path), trie_go.MustSize(proof))
		// validate proof
		err := proof.Validate(rootCommitment)
		errstr := "OK"
		if err != nil {
			errstr = err.Error()
		}
		if err != nil {
			fmt.Printf("validating PoI for '%s': %s\n", s, errstr)
			continue
		}
		if proof.IsProofOfAbsence() {
			fmt.Printf("key '%s' is NOT IN THE STATE\n", s)
		} else {
			fmt.Printf("key '%s' is IN THE STATE\n", s)
		}
	}
}
