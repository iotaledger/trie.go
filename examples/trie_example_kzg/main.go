package main

import (
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/iotaledger/trie.go/models/trie_kzg_bn256"
	"github.com/iotaledger/trie.go/mutable"
)

var data = []string{"a", "abc", "abcd", "b", "abd", "klmn", "oprst", "ab", "bcd"}

func main() {
	// create store where trie nodes will be stored
	store := common.NewInMemoryKVStore()

	// create kzg commitment common for 256-ary trie
	m := trie_kzg_bn256.New()

	// create the trie with binary keys
	tr := mutable.New(m, store, nil)
	fmt.Printf("\nExample of trie.\n%s\n", tr.Info())

	// add data key/value pairs to the trie
	for _, s := range data {
		fmt.Printf("add key '%s' into the trie\n", s)
		tr.UpdateStr(s, s+"$")
	}
	// recalculate commitments in the trie
	tr.Commit()
	rootCommitment := mutable.RootCommitment(tr)
	fmt.Printf("root commitment: %s\n", rootCommitment)
	// remove some keys from the trie
	for _, i := range []int{1, 5, 6} {
		fmt.Printf("remove key '%s' from the trie\n", data[i])
		tr.DeleteStr(data[i])
	}
	// recalc trie again
	tr.Commit()
	rootCommitment = mutable.RootCommitment(tr)
	fmt.Printf("root commitment: %s\n", rootCommitment)

	// check PoI for all data
	for _, s := range data {
		// retrieve proof
		proof, exists := m.ProofOfInclusion([]byte(s), tr)
		if !exists {
			fmt.Printf("key not found: '%s'\n", s)
			continue
		}
		fmt.Printf("PoI of the key '%s': length %d, serialized size %d bytes\n",
			s, len(proof.Path), common.MustSize(proof))
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
	}
}
