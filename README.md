## trie.go
Go library for implementations of tries (radix trees), state commitments and _proof of inclusion_ for large data sets.

It implements a generic `256+ trie` for several particular commitment schemes. 

The trie implementation has minimal dependencies and no dependencies on other projects in IOTA. 

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
Contains implementation of the `CommitmentModel` as a Merkle tree on the `256+ trie` with data commitment via `blake2b` hash function. 

The implementation is fast and optimized. It can be used in various project. It is used in the `Wasp` node.

The usage of hashing function as a commitment function results in proofs of inclusion 5-6 times bigger than with
polynomial KZG (aka Kate) commitments (size of PoI usually is up to 1-2K bytes).

### Package `trie_kzg_bn256` 
Contains implementation of the `CommitmentModel` as the _verkle_ tree which uses _KZG (Kate) commitments_ 
as a scheme for vectors commitments and `bn256` curve from _Dedis Kyber_ library. 
For related math see the [writeup](https://hackmd.io/@Evaldas/SJ9KHoDJF).

The underlying KZG cryptography is slow and suboptimal. The speed of update of the `256+ trie` is some 1-2 orders of magnitude 
slower than implementation with `blake2b` hash function. The proofs of inclusion, however, are very short, up to 5-6
times shorter.

This makes `trie_kzg_bn256` implementation more a _proof of concept_ and verification of the `256+ trie` concept. 
It should not be use in practical project, unless `bn256` is replaced with other, faster curves.

## Package `trie_go_tests`
Contains number of tests of the trie implementation. Most of the tests run with both `trie_blak2b` and `trie_kzg_bn256` 
implementations of the `CommitmentModel`. The tests check different edge conditions and determinism of the `trie`.
It also makes sure `trie256p` implementation is agnostic about the specific commitment model. 

## Package `hive_adaptor`
Contains useful adaptors to key/value interface of `hive.go`. It makes `trie.go` compatible with any key/value database
implemented in the `hive.go`.

## Package `trie_bench`
Contains `trie_bench` program made for testing and benchmarking of different functions of `trie` with `tre_blake2b` 
commitment model. The `trie_bench` uses `Badger` key/value database via `hive_adaptor`.

In the directory of the package run `go install`. Then you can run the program one of the following ways:

* `trie_bench -gen <size> <name>` generates a binary file `<name>.bin` of `<size>` random keys and values. Key and value are of variable length.
* `trie_bench -genhash <size> <name>` generates a binary file `<name>.bin` of `<size>` random keys and values. Keys and values have fixed length of 32 bytes.
* `trie_bench -mkdbmem <name>` loads file `<name>.bin` into the in-memory k/v database, both values and the trie. Outputs statistics.  
* `trie_bench -mkdbbadger <name>` loads file `<name>.bin` into the `Badger` k/v database on directory `<name>.dbdir`, both values and the trie. 
Outputs statistics.
* `trie_bench -scandbbadger <name>` scans database and outputs statistics. Then it iterates over all keys and value in the database 
and for each key/value pair:
  * retrieves proof of inclusion for the key from the trie
  * runs validation of the proof
  * collects statistics

Statistics on the 2.8 GhZ 32 GB RAM SDD laptop. 
`trie_bench` run over the key/value database of 1 mil key/value pairs and `trie_blake2b` commitment model:

| Parameter                                                              | Value               |
|------------------------------------------------------------------------|---------------------|
| Load 1 mil records into the DB <br> with trie generation (cached trie) | 31400 key pairs/sec |
| Retrieve proof + validation (not-cached trie)                          | 3400 proofs/sec     |
| Average length of the proof path                                       | 4.04                |
| Average size of serialized proof                                       | 17 kB               |


## Package `trie_example`  
Contains example with the in memory key/value store. Run `go install` and the run the program `trie_example`.
