# Package `trie_blake2b`

Package contains implementation of commitment model for the `256+ trie` based on `MIMC` 32 byte hashing.

## Structure of the proof

The proof of inclusion or absence is the proof of inclusion of key/value pair (K,V) into the key/value store.
If your key value store is used to only keep keys as leafs, the key/value pair will be (K,K).

The proof is a sequence of hashes `H[0], ..., H[N-1]` where:
* `H[N-1]` includes value V as part of it
* `H[0]` is a root hash. It must be equal to the root hash value provided from outside by the verifier
* must be `H[i]` = `Hash(H[i+1])` along the path

The proof in the `trie-mimc.CommitmentModel` implementation is represented by two data structure

```go
type Proof struct {
    PathArity trie.PathArity
    Key       []byte
    Path      []*ProofElement
}
```
and
```go
type ProofElement struct {
    PathFragment []byte
    Children     map[byte][]byte
    Terminal     []byte
    ChildIndex   int
}
```

The proof as a data has no dependencies to any data structures. The only functionality needed in order to check the validity of the proof
is ability of MIMC-hashing of the binary representation of hashes.

You will retrieve the proof by calling `proof := model.Proof(key, trie)` where `model` was created 
with ` model := trie_mimc1.New(trie.PathArity2))` and the `trie := tr := trie.New(m, store, nil)`.

This way it is guaranteed that the proof is correct. But to check it against specific root hash `root` 
you call `trie_mimc1_verify.Validate(proof, root)`. 

The `Proof` structure has header information:
* `PathArity` will be `1` for binary tries
* `Key` will be equal the key you want to check in the key/value store i.e. what you provided in the call `model.Proof(key, trie)` 

The `Path` is a list of `ProofElement`'s. Each proof element in the list is used to compute the hash `H[i]` above.

Let's say the `Path` contains `N` elements. 
* Only the last element in the proof path at index `N-1` contains all information needed to compute `H[N-1]`:
* each element at other index `i` will need to value of `H[i+1]` to compute `H[i]`
* So, we start at last one at `H[N-1]` and then compute one by one up to the `H[0]`

For binary tree, the last element `proof.Path[N-1]` will always contain:
* value `V` corresponding to the key `K` in the key/value store as
`proof.Path[N-1].Terminal`. It cannot be `nil` for proof of existence (if the (K,V) exists in the key/value store). 
It will be MIMC hash of the raw commitment to data (which may by up to 33 byte long)
* 0, 1 or 2 values in the map `proof.Path[N-1].Children`
* `proof.Path[N-1].ChildIndex` is not important for the last element in the path

The hash `H[N-1]` is computed by concatenating `proof.Path[N-1].Children[0], proof.Path[N-1].Children[1], proof.Path[N-1].Terminal, proof.Path[N-1].PathFragment` 
If value is `nil` (it cannot be for terminal in the last element), the all-0 value `[32]byte{}` is used.

So this was the last element in the path.
Now we go to the next one at index `N-2`, and so on to `N-3` down to inde `0`.
Each element `proof.Path[i].ChildIndex` where `i<N-1` will contain for binary tree:
* exactly 1 element in the `proof.Path[i].Children` either at index `0` or index `1`. The missing (`nil`) element in the map `proof.Path[i].Children` 
will be at index `proof.Path[i].ChildIndex`. It can be only `0` or `1` for the binary tree. That would be the hash from below (next proof element).
The other will be the sibling hash already present int the tree. 

So, for the `proof.Path[i]` the `H[i]` is calculated by concatenating and hashing the following values:
`C0[i], C1[i], proof.Path[i].Terminal, proof.Path[i].PathFragment` (`nil` mean all-0).
Here: 
If `proof.Path[i].ChildIndex == 0` then:  
* `C[0] = H[i+1]`
* `C[1] = proof.Path[i].Children[1]`

Otherwise, if `proof.Path[i].ChildIndex == 1` then:
* `C[0] = proof.Path[i].Children[0]`
* `C[1] = H[i+1]`

(other than `0` or `1` value are not possible in the binary tree)

The whole logic of forming the vector for hashing is in the functions `hashIt`, `makeHashVector` and `HashTheVector` 

## Summery: how to generate circuit
1. Have key K
2. Call `proof := model.Proof(key, trie)`
3. Call `trie_mimc1_verify.Validate(proof, root)`. It checks syntactic validity of the proof and is it the valid proof against you `root`. 
4. The proof will prove the inclusion of key K with the value committed in the `Terminal` field in the last element in the `Path`.
5. You generate circuit going from the last element in the path `proof.Path[N-1]` to the element `0`. 
6. In each step you generate a circuit corresponding to 4 values (not 2): 1 sibling, 1 child, terminal, and the path fragment. 
7. Terminal will be not `nil` in the last element, exactly 1 child will be in the elements which are not the last one
8. Which one is child and which one is sibling is contained in the `ChildIndex` field which is `0` or `1`

