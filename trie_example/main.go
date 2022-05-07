package main

import (
	"fmt"
	trie_go "github.com/iotaledger/trie.go"
	"github.com/iotaledger/trie.go/trie"
	"github.com/iotaledger/trie.go/trie_blake2b_32"
)

func main() {
	// create store where trie nodes will be stored
	store := trie_go.NewInMemoryKVStore()

	// create blake2b commitment model
	model := trie_blake2b_32.New()

	// create the trie
	tr := trie.New(model, store, trie.PathArity2, false)

	// add some key/value pairs to the trie
	keys := []string{"abc", "klm", "oprs"}
	tr.Update([]byte(keys[0]), []byte("dummy1"))
	tr.Update([]byte(keys[1]), []byte("dummy2"))

	// recalculate commitments in the trie
	tr.Commit()

	// retrieve root commitment (normally it is taken from the 3rd party)
	rootCommitment := trie.RootCommitment(tr)

	// prove that key 'abc' is in the state against the root
	proof := model.Proof([]byte(keys[0]), tr)
	err := proof.Validate(rootCommitment)
	if err == nil {
		fmt.Printf("key '%s' is in the state\n", keys[0])
	}

	// prove that key 'oprs' is not in the state
	proof = model.Proof([]byte(keys[2]), tr)
	err = proof.Validate(rootCommitment)
	if err == nil && proof.IsProofOfAbsence() {
		// proof is valid, however it proves inclusion of something else in the state and effectively
		// proves absence of the target key
		fmt.Printf("key '%s' is NOT in the state\n", keys[2])
	}
}
