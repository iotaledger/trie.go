package main

import (
	"fmt"

	"github.com/iotaledger/trie.go/models/trie_mimc"
	"github.com/iotaledger/trie.go/models/trie_mimc/trie_mimc_verify"
	"github.com/iotaledger/trie.go/trie"
)

var data = []string{"a", "abc", "abcd", "b", "abd", "klmn", "oprst", "ab", "bcd"}

func main() {
	// create store where trie nodes will be stored
	store := trie.NewInMemoryKVStore()

	// create mimc 32 bytes (256 bit) commitment model for binary trie
	model := trie_mimc.New(trie.PathArity2, trie_mimc.HashSize256)

	// create the trie with binary keys
	tr := trie.New(model, store, nil)
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
	for _, i := range []int{1, 5, 6} {
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
			s, len(proof.Path), trie.MustSize(proof))
		// validate proof
		err := trie_mimc_verify.Validate(proof, rootCommitment.Bytes())
		errstr := "OK"
		if err != nil {
			errstr = err.Error()
		}
		if err != nil {
			fmt.Printf("validating PoI for '%s': %s\n", s, errstr)
			continue
		}
		if trie_mimc_verify.IsProofOfAbsence(proof) {
			fmt.Printf("key '%s' is NOT IN THE STATE\n", s)
		} else {
			fmt.Printf("key '%s' is IN THE STATE\n", s)
		}
	}
}
