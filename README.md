## trie.go  (WIP)
Go library for implementations of tries (radix trees), state commitments and _proof of inclusion_ in large data sets.

It implements a generic `256+ trie` for several particular commitment schemes. 

The repository has minimal dependencies and no dependencies on other projects in IOTA. 

It is used as a dependency in the [IOTA Smart Contracts (the Wasp node)](https://github.com/iotaledger/wasp) 
as the engine for the state commitment.

The `blake2b`-based trie implementation is ready for the use in other project with any implementations of key/value store. 

### Root package of `trie.go` 
Contains data types and interfaces shared between different implementations of trie:
- interfaces `VCommitment` and `TCommitment` abstracts implementation from serialization details
- `KVReader`, `KVWriter`, `KVIterator` abstracts implementation from details of a particular key/value store
- various utility functions used in the code and in tests

### Package `trie256p` 
Contains implementation of the extended [radix trie](https://en.wikipedia.org/wiki/Radix_tree).

It essentially follows the formal definition provided in [_256+ trie. Definition_](https://hackmd.io/@Evaldas/H13YFOVGt). 

The implementation is optimized performance and storage-wise. It provides `O(log_256(N))` complexity of the trie updates.

The trie itself is stored as a collection of key/value pairs. Updating of the trie is cached in memory. 
It makes trie update a very fast operation.

The `trie256p` is abstracted from both the particular commitment scheme through `CommitmentModel` interface 
and from details of key/value store implementation via `KVStore` interface. 

The generic implementation of `256+ trie` can be used to implement different commitment models by implementing 
`CommitmentModel` interface.  

### Package `trie_blake2b`
Contains implementation of the `CommitmentModel` as the Merkle tree on the `256+ trie` with data commitment via `blake2b` hash function. 

The implementation is fast and optimized. It can be used in various project. It is used in the `Wasp` node.

The usage of hashing function as a commitment function results in proofs of inclusion 5-6 times bigger than with
polynomial KZK (Kate) commitments (size of PoI usually is up to 1-2K bytes).

### Package `trie_kzg_bn256` 
Contains implementation of the `CommitmentModel` as the _verkle_ tree which uses _KZG (Kate) commitments_ 
as a scheme for vectors commitments and `bn256` curve from _Dedis Kyber_ library. 
For related math see the [writeup](https://hackmd.io/@Evaldas/SJ9KHoDJF).

The underlying KZG cryptography is slow and suboptimal. The speed of update of the `256+ trie` is some 1-2 orders of magnitude 
slower than implementation with `blake2b` hash function. The proofs of inclusion, however, are very short, up to 5-6
times shorter.

This makes `trie_kzg_bn256` implementation more a _proof of concept_ and verification of the `256+ trie` concept. 
It should not be use in practical project, unless `bn256` is replaced with other, faster curves.

## Example 

```go
package main

import (
	"fmt"
	trie_go "github.com/iotaledger/trie.go"
	"github.com/iotaledger/trie.go/trie256p"
	"github.com/iotaledger/trie.go/trie_blake2b"
)

func main() {
	// create store where trie nodes will be stored
	store := trie_go.NewInMemoryKVStore()

	// create blake2b commitment model
	model := trie_blake2b.New()

	// create the trie
	tr := trie256p.New(model, store)

	// add some key/value pairs to the trie
	keys := []string{"abc", "klm", "oprs"}
	tr.Update([]byte(keys[0]), []byte("dummy1"))
	tr.Update([]byte(keys[1]), []byte("dummy2"))

	// recalculate commitments in the trie
	tr.Commit()

	// retrieve root commitment (normally it is taken from the 3rd party)
	rootCommitment := trie256p.RootCommitment(tr)

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
```