## trie.go
Go library for implementations of sparse tries (sparse radix trees), state commitments and _proof of inclusion_ 
for large **mutable** data sets with **variable size keys**. 

The _mutable state model_ is assumed as a main use case, but not limited to it. 
By _mutable state_ we assume a large key/value data set which is frequently updated by 
adding, modifying and deleting key/value pairs. 
The application is only interested in the latest (current) state of the mutable state and the commitment to it. 

The `trie.go` package allows efficient and deterministic recalculation of the commitment tree of the 
data set upon each batch of mutations with minimum overhead, in a logarithmic time.

The trie update and proof retrieval operations are highly optimized via caching access 
to the database. It buffers trie updates up until the trie is _committed_ (recalculated).
It saves a lot of DB interactions and a lot of cryptographic operations (hashing or curve arithmetics).

This _mutable commitment tree model_ extends _append only commitment tree model_ often used 
in blockchains, where commitment tree is augmented each time by linearly appending new block
with its Merkle tree to the branch of block headers. 
The former is much less redundant and more efficient in high throughput 
systems when state is mutated with large blocks (batches of state mutations) each few seconds. 

Note that _variable size keys_ implies terminal value can be stored in a key which is 
a prefix of another key. 
This trait makes the state hierarchical, i.e. with sub-states of key/value collection where all 
keys share same prefix. 

`trie.go` implements a generic `256+ trie` for several particular cryptographic commitment schemes with 
rich set of optimization options. 

The library supports both variable and fixed-sized keys as well as a number optimization options:
* 256-ary trie is best for fixed-sized commitment models like `KZG (Kate`  and `verkle` tries
* 16-ary (hexary) trie is similar to Patricia trees. It is close to optimal when it comes to hash-based commitment models
* 2-ary (binary) trie gives the smallest proof size with hash-based commitment. However, much longer proof path and 
bytes-to-bits packing/unpacking overhead is noticeable
* library also supports `key commitments`, a trie optimization when key and value are equal. This makes it optimal for ledger-state commitments, 
because in ledger state commitments the committed values are commitments itself (UTXO IDs or transaction ID) 
and the trie stores committed terminal value only once. This option is ideal for commitment to the ledger state.
* terminal values can be committed to any node of the tree, not only leafs of it, in optimal way.

The trie implementation has minimal dependencies on other projects. 

It is used as a dependency in the [IOTA Smart Contracts (the Wasp node)](https://github.com/iotaledger/wasp) 
as the engine for the state commitment.

The `blake2b`-based trie implementation is ready for the use in other projects. 

### Package `trie` 
Contains: 
- an implementation of the (extended) [radix trie](https://en.wikipedia.org/wiki/Radix_tree).
- data types and interfaces shared between different implementations of trie:
  - interfaces `VCommitment` and `TCommitment` abstracts implementation from serialization details
  - `KVReader`, `KVWriter`, `KVIterator` interfaces abstracts implementation from details of a particular key/value store
  - the `CommitmentModel` interface abstracts trie implementation from particularities of specific commitments schemes
  - various utility functions used in the code and in tests


It essentially follows the formal definition provided in [_256+ trie. Definition_](https://hackmd.io/@Evaldas/H13YFOVGt). 

The implementation is optimized performance and storage-wise. 
It provides `O(log_256(N))` complexity of the trie updates.

The trie itself is stored as a collection of key/value pairs. Updating of the trie is cached in memory. 
It makes trie update a very fast operation.

### Packages `models/trie_blake2b`
Contains implementation of the `CommitmentModel` as a sparse Merkle tree on the `trie` with data commitment via `blake2b` hash function.

The binary (2-ary) trie is essentially the same as well known _Sparse Merkle Tree_. 

The implementation takes particular hash size used in the commitments as a parameter.

The usage of hashing function as a commitment function results in proofs of inclusion up to 5-6 times bigger than with (1-2Kbytes)
polynomial KZG (aka Kate) commitments.

### Package `models/trie_kzg_bn256` 
Contains implementation of the `CommitmentModel` as the **verkle tree** which uses _KZG (Kate) commitments_ 
as a scheme for vectors commitments and `bn256` curve from _Dedis Kyber_ library. 
For related math and other references see the [writeup](https://hackmd.io/@Evaldas/SJ9KHoDJF).

The underlying KZG cryptography in this specific implementation is rather slow and suboptimal. 
The speed of update of the `256+ trie` is some 1-2 orders of magnitude 
slower than implementation with `blake2b` hash function. 
The proofs of inclusion, however, are very short, up to 5-6 times shorter, ~200 bytes only.

The `models/trie_kzg_bn256` implementation is more a _proof of concept_ and verification of the `256+ trie` concept. 
It should not be use in practical project, unless `bn256` is replaced with other, faster curves.

## Package `models/tests`
Contains number of tests of the trie implementation. 
Same tests run for `trie_blak2b` 256 and 160 bit hashing and `trie_kzg_bn256` 
implementations of the `CommitmentModel` and different combinations of other parameters such as arity of the trie.
It also makes sure `trie` implementation is agnostic about the specific commitment model and optimization parameters. 

## Package `hive_adaptor`
Contains useful adaptors to key/value interface of `hive.go`. 
It makes `trie.go` compatible with any key/value storages implemented in the `github.com/iotaledger/hive.go`.

## Package `examples/trie_bench`
Contains `trie_bench` program made for testing and benchmarking of different functions of `trie` with `tre_blake2b` 
commitment model. The `trie_bench` uses `Badger` key/value database via `hive_adaptor`.

In the directory of the package run `go install` and run the program with options and commands. 

Commands:

* `trie_bench [flags] gen <name>` generates a binary file `<name>.bin` of `<size>` random keys and values.
* `trie_bench [flags] mkdbmem <name>` loads file `<name>.bin` into the in-memory k/v database, both values and the trie. Outputs statistics.  
* `trie_bench [flags] mkdbbadger <name>` loads file `<name>.bin` into the `Badger` k/v database on directory `<name>.dbdir`, both values and the trie. 
Outputs statistics.
* `trie_bench [flags] scandbbadger <name>` scans database and outputs statistics. Then it iterates over all keys and value in the database 
and for each key/value pair:
  * retrieves proof of inclusion for the key from the trie
  * runs validation of the proof
  * collects statistics
* `trie_bench mkdbbadgernotrie <name>` just loads key/value pairs to DB

Flags:

* `-n=<num>` number of key value pairs to generate. Default is `1000`.
* `-arity=2|16|32` default is `16`
* `-blake2b=20|32` default is `20`
* `-hashkv` if present, keys and values will be hashed to 32 bytes while generating random file. Defaults to `false`
* `-optkey` if present, `key commitment` optimization will be enabled. Default is `false`

### Benchmark results I
Statistics on the 2.8 GhZ 32 GB RAM SDD laptop. 
`trie_bench` run over the key/value database of 1 mil key/value pairs and `trie_blake2b` commitment model:

* Commitment model `trie_blake2b` 160 bit
* 1 million key/value pairs
* maximum key size: 100 bytes
* maximum value size: 32 bytes
* all terminal values are stored in the trie nodes (no optimization)
* 1 mil key/value pairs in file: 69 MB
* 1 mil key/value pairs in Badger DB (no trie): 162 MB

_Note: retrieval speed very much depends on caching parameters. Benchmark results (caching turned off) would be much faster
with caching turned on._

#### Benchmark for 256-ary trie
| Parameter                                                              | blake2b 160 bit<br/>model | 
|------------------------------------------------------------------------|---------------------------|
| Load 1 mil records into the DB <br> with trie generation (cached trie) | 30000 kv pairs/sec        |
| Badger DB size                                                         | 408 MB                    |
| Retrieve proof + validation (not-cached trie )                         | 3340 proofs/sec           |
| Average length of the proof path                                       | 4.04                      |
| Average size of serialized proof                                       | 10.6 kB                   |

#### Benchmark for 16-ary trie hexary)
| Parameter                                                              | blake2b 160 bit<br/>model |
|------------------------------------------------------------------------|---------------------------|
| Load 1 mil records into the DB <br> with trie generation (cached trie) | 32000 kv pairs/sec        |
| Badger DB size                                                         | 424 MB                    |
| Retrieve proof + validation (not-cached trie)                          | 14400 proofs/sec          |
| Average length of the proof path                                       | 6.6                       |
| Average size of serialized proof                                       | 1.75 kB                   |

#### Benchmarks for 2-ary trie (binary)
| Parameter                                                              | blake2b 160 bit<br/>model |
|------------------------------------------------------------------------|---------------------------|
| Load 1 mil records into the DB <br> with trie generation (cached trie) | 13800 kv pairs/sec        |
| Badger DB size                                                         | 540 MB                    |
| Retrieve proof + validation (not-cached trie)                          | 7580 proofs/sec           |
| Average length of the proof path                                       | 21.1                      |
| Average size of serialized proof                                       | 1.3 kB                    |

We can see that optimal choice with `blake2b 160-bit` model is between hexary and binary trie. 
Note that binary trie is using more runtime memory for key packing/unpacking.

The `KZG (Kate)` model would give the shortest proofs (~200 bytes) with 1-2 orders of magnitude slower trie update.

### Benchmark results II: compare storage optimization
Statistics on the 2.8 GhZ 32 GB RAM SDD laptop.

* * Commitment model `trie_blake2b` 160 bit
* 1 million key/value pairs
* maximum key size: 100 bytes
* maximum value size: 40 Kbytes
* 1 mil key/value pairs in file: 18.6 GB

#### Benchmark for 16-ary (hexary) trie, terminal commitments NOT STORED in the trie
| Parameter                                                              | blake2b 160 bit<br/>model | 
|------------------------------------------------------------------------|---------------------------|
| Load 1 mil records into the DB <br> with trie generation (cached trie) | 5100 kv pairs/sec         |
| Badger DB size                                                         | 18.8 GB                   |
| Retrieve proof + validation (not-cached trie )                         | 4700 proofs/sec           |
| Average length of the proof path                                       | 6.62                      |
| Average size of serialized proof                                       | 1.77 kB                   |

#### Benchmark for 16-ary (hexary) trie, approx 25% of terminal commitments are STORED in the trie
(terminal commitments for values longer than 10000 bytes are stored in the trie)

| Parameter                                                              | blake2b 160 bit<br/>model | 
|------------------------------------------------------------------------|---------------------------|
| Load 1 mil records into the DB <br> with trie generation (cached trie) | 7000 kv pairs/sec         |
| Badger DB size                                                         | 18.8 GB                   |
| Retrieve proof + validation (not-cached trie )                         | 8600 proofs/sec           |
| Average length of the proof path                                       | 6.62                      |
| Average size of serialized proof                                       | 1.77 kB                   |

We can see that big average values does not inflate trie at some expense of proof performance speed. 
This is expected because big value must be fetched from the DB and hashed every time when terminal
commitment is needed. 

However, with terminal value threshold parameter performance can be optimized without noticeable
increase in the DB size.

## Package `examples/trie_example`  
Contains a simple example with the in memory key/value store. Run `go install` and the run the program `trie_example`.

```go
package main

import (
  "fmt"
  "github.com/iotaledger/trie.go/models/trie_blake2b"
  "github.com/iotaledger/trie.go/models/trie_blake2b/trie_blake2b_verify"
  "github.com/iotaledger/trie.go/trie"
)

var data = []string{"a", "abc", "abcd", "b", "abd", "klmn", "oprst", "ab", "bcd"}

func main() {
  // create store where trie nodes will be stored
  store := trie.NewInMemoryKVStore()

  // create blake2b 20 bytes (160 bit) commitment model
  model := trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize160)

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
      s, len(proof.Path), trie.MustSize(proof))
    // validate proof
    err := trie_blake2b_verify.Validate(proof, rootCommitment.Bytes())
    errstr := "OK"
    if err != nil {
      errstr = err.Error()
    }
    if err != nil {
      fmt.Printf("validating PoI for '%s': %s\n", s, errstr)
      continue
    }
    if trie_blake2b_verify.IsProofOfAbsence(proof) {
      fmt.Printf("key '%s' is NOT IN THE STATE\n", s)
    } else {
      fmt.Printf("key '%s' is IN THE STATE\n", s)
    }
  }
}

```
