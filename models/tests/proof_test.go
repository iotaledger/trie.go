package tests

import (
	"bytes"
	"github.com/iotaledger/trie.go/models/trie_blake2b"
	"github.com/iotaledger/trie.go/models/trie_kzg_bn256"
	"github.com/iotaledger/trie.go/trie"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTrieProofBlake2b(t *testing.T) {
	runTest20 := func(arity trie.PathArity) {
		model := trie_blake2b.New(arity, trie_blake2b.HashSize160)
		t.Run("proof empty trie"+tn(model), func(t *testing.T) {
			store := trie.NewInMemoryKVStore()
			tr := trie.New(model, store)
			require.EqualValues(t, nil, trie.RootCommitment(tr))

			proof := model.Proof(nil, tr)
			require.EqualValues(t, 0, len(proof.Path))
		})
		t.Run("proof one entry 1"+tn(model), func(t *testing.T) {
			store := trie.NewInMemoryKVStore()
			tr := trie.New(model, store)

			tr.Update(nil, []byte("1"))
			tr.Commit()

			proof := model.Proof(nil, tr)
			require.EqualValues(t, 1, len(proof.Path))

			rootC := trie.RootCommitment(tr)
			err := proof.Validate(rootC)
			require.NoError(t, err)

			// t.Logf("proof presence size = %d bytes", trie_go.MustSize(proof))

			key, term, isHash := proof.MustKeyWithTerminal()
			require.False(t, isHash)
			c := model.CommitToData([]byte("1"))
			c1 := model.CommitToData(term)
			require.EqualValues(t, 0, len(key))
			require.True(t, trie.EqualCommitments(c1, c))

			unpackedKey := trie.UnpackBytes([]byte("a"), arity)
			proof = model.Proof(unpackedKey, tr)
			require.EqualValues(t, 1, len(proof.Path))

			rootC = trie.RootCommitment(tr)
			err = proof.Validate(rootC)
			require.NoError(t, err)
			require.True(t, proof.IsProofOfAbsence())
			t.Logf("proof absence size = %d bytes", trie.MustSize(proof))
		})
		t.Run("proof one entry 2"+tn(model), func(t *testing.T) {
			store := trie.NewInMemoryKVStore()
			tr := trie.New(model, store)

			tr.Update([]byte("1"), []byte("2"))
			tr.Commit()
			proof := model.Proof(nil, tr)
			require.EqualValues(t, 1, len(proof.Path))

			rootC := trie.RootCommitment(tr)
			err := proof.Validate(rootC)
			require.NoError(t, err)
			require.True(t, proof.IsProofOfAbsence())

			proof = model.Proof([]byte("1"), tr)
			require.EqualValues(t, 1, len(proof.Path))

			err = proof.Validate(rootC)
			require.NoError(t, err)
			require.False(t, proof.IsProofOfAbsence())

			_, term, isHash := proof.MustKeyWithTerminal()
			require.False(t, isHash)
			c := model.CommitToData([]byte("2"))
			c1 := model.CommitToData(term)
			require.True(t, trie.EqualCommitments(c, c1))
		})
	}
	runTest32 := func(arity trie.PathArity) {
		model := trie_blake2b.New(arity, trie_blake2b.HashSize256)
		t.Run("proof empty trie"+tn(model), func(t *testing.T) {
			store := trie.NewInMemoryKVStore()
			tr := trie.New(model, store)
			require.EqualValues(t, nil, trie.RootCommitment(tr))

			proof := model.Proof(nil, tr)
			require.EqualValues(t, 0, len(proof.Path))
		})
		t.Run("proof one entry 1"+" "+arity.String(), func(t *testing.T) {
			store := trie.NewInMemoryKVStore()
			tr := trie.New(model, store)

			tr.Update(nil, []byte("1"))
			tr.Commit()

			proof := model.Proof(nil, tr)
			require.EqualValues(t, 1, len(proof.Path))

			rootC := trie.RootCommitment(tr)
			err := proof.Validate(rootC)
			require.NoError(t, err)

			//t.Logf("proof presence size = %d bytes", trie_go.MustSize(proof))

			key, term, isHash := proof.MustKeyWithTerminal()
			require.False(t, isHash)
			c := model.CommitToData([]byte("1"))
			c1 := model.CommitToData(term)
			require.EqualValues(t, 0, len(key))
			require.True(t, trie.EqualCommitments(c1, c))

			proof = model.Proof([]byte("a"), tr)
			require.EqualValues(t, 1, len(proof.Path))

			rootC = trie.RootCommitment(tr)
			err = proof.Validate(rootC)
			require.NoError(t, err)
			require.True(t, proof.IsProofOfAbsence())
			t.Logf("proof absence size = %d bytes", trie.MustSize(proof))
		})
		t.Run("proof one entry 2"+tn(model), func(t *testing.T) {
			store := trie.NewInMemoryKVStore()
			tr := trie.New(model, store)

			tr.Update([]byte("1"), []byte("2"))
			tr.Commit()
			proof := model.Proof(nil, tr)
			require.EqualValues(t, 1, len(proof.Path))

			rootC := trie.RootCommitment(tr)
			err := proof.Validate(rootC)
			require.NoError(t, err)
			require.True(t, proof.IsProofOfAbsence())

			proof = model.Proof([]byte("1"), tr)
			require.EqualValues(t, 1, len(proof.Path))

			err = proof.Validate(rootC)
			require.NoError(t, err)
			require.False(t, proof.IsProofOfAbsence())

			_, term, isHash := proof.MustKeyWithTerminal()
			require.False(t, isHash)
			c := model.CommitToData([]byte("2"))
			c1 := model.CommitToData(term)
			require.True(t, trie.EqualCommitments(c, c1))
		})
	}
	runTest20(trie.PathArity256)
	//runTest20(trie.PathArity16)
	//runTest20(trie.PathArity2)
	runTest32(trie.PathArity256)
	//runTest32(trie.PathArity16)
	//runTest32(trie.PathArity2)
}

func TestTrieProofWithDeletesBlake2b32(t *testing.T) {
	var tr1 *trie.Trie
	var rootC trie.VCommitment
	var model *trie_blake2b.CommitmentModel

	initTrie := func(dataAdd []string, arity trie.PathArity) {
		model = trie_blake2b.New(arity, trie_blake2b.HashSize160)
		store := trie.NewInMemoryKVStore()
		tr1 = trie.New(model, store)
		for _, s := range dataAdd {
			tr1.Update([]byte(s), []byte(s+"++"))
		}
	}
	deleteKeys := func(keysDelete []string) {
		for _, s := range keysDelete {
			tr1.Update([]byte(s), nil)
		}
	}
	commitTrie := func() trie.VCommitment {
		tr1.Commit()
		return trie.RootCommitment(tr1)
	}
	data := []string{"a", "ab", "ac", "abc", "abd", "ad", "ada", "adb", "adc", "c", "ad+dddgsssisd"}
	runTest := func(arity trie.PathArity) {
		t.Run("proof many entries 1"+"-"+arity.String(), func(t *testing.T) {
			initTrie(data, arity)
			rootC = commitTrie()
			for _, s := range data {
				proof := model.Proof([]byte(s), tr1)
				require.False(t, proof.IsProofOfAbsence())
				err := proof.Validate(rootC)
				require.NoError(t, err)
				//t.Logf("key: '%s', proof len: %d", s, len(proof.Path))
				//t.Logf("proof presence size = %d bytes", trie_go.MustSize(proof))
			}
		})
		t.Run("proof many entries 2"+" "+arity.String(), func(t *testing.T) {
			delKeys := []string{"1", "2", "3", "12345", "ab+", "ada+"}
			initTrie(data, arity)
			deleteKeys(delKeys)
			rootC = commitTrie()

			for _, s := range data {
				proof := model.Proof([]byte(s), tr1)
				err := proof.Validate(rootC)
				require.NoError(t, err)
				require.False(t, proof.IsProofOfAbsence())
				//t.Logf("key: '%s', proof presence lenPlus1: %d", s, len(proof.Path))
				//t.Logf("proof presence size = %d bytes", trie_go.MustSize(proof))
			}
			for _, s := range delKeys {
				proof := model.Proof([]byte(s), tr1)
				err := proof.Validate(rootC)
				require.NoError(t, err)
				require.True(t, proof.IsProofOfAbsence())
				//t.Logf("key: '%s', proof absence lenPlus1: %d", s, len(proof.Path))
				//t.Logf("proof absence size = %d bytes", trie_go.MustSize(proof))
			}
		})
		t.Run("proof many entries 3"+" "+arity.String(), func(t *testing.T) {
			delKeys := []string{"1", "2", "3", "12345", "ab+", "ada+"}
			allData := make([]string, 0, len(data)+len(delKeys))
			allData = append(allData, data...)
			allData = append(allData, delKeys...)
			initTrie(allData, arity)
			deleteKeys(delKeys)
			rootC = commitTrie()

			for _, s := range data {
				proof := model.Proof([]byte(s), tr1)
				err := proof.Validate(rootC)
				require.NoError(t, err)
				require.False(t, proof.IsProofOfAbsence())
				//t.Logf("key: '%s', proof presence lenPlus1: %d", s, len(proof.Path))
				sz := trie.MustSize(proof)
				//t.Logf("proof presence size = %d bytes", sz)

				proofBin := proof.Bytes()
				require.EqualValues(t, len(proofBin), sz)
				proofBack, err := trie_blake2b.ProofFromBytes(proofBin)
				require.NoError(t, err)
				err = proofBack.Validate(rootC)
				require.NoError(t, err)
				require.True(t, bytes.Equal(proof.Key, proofBack.Key))
				require.False(t, proofBack.IsProofOfAbsence())
			}
			for _, s := range delKeys {
				proof := model.Proof([]byte(s), tr1)
				err := proof.Validate(rootC)
				require.NoError(t, err)
				require.True(t, proof.IsProofOfAbsence())
				t.Logf("key: '%s', proof absence lenPlus1: %d", s, len(proof.Path))
				sz := trie.MustSize(proof)
				t.Logf("proof absence size = %d bytes", sz)

				proofBin := proof.Bytes()
				require.EqualValues(t, len(proofBin), sz)
				proofBack, err := trie_blake2b.ProofFromBytes(proofBin)
				require.NoError(t, err)
				err = proofBack.Validate(rootC)
				require.NoError(t, err)
				require.True(t, bytes.Equal(proof.Key, proofBack.Key))
				require.True(t, proofBack.IsProofOfAbsence())
			}
		})
		t.Run("proof many entries rnd"+" "+arity.String(), func(t *testing.T) {
			addKeys, delKeys := gen2different(100000)
			t.Logf("lenPlus1 adds: %d, lenPlus1 dels: %d", len(addKeys), len(delKeys))
			allData := make([]string, 0, len(addKeys)+len(delKeys))
			allData = append(allData, addKeys...)
			allData = append(allData, delKeys...)
			initTrie(allData, arity)
			deleteKeys(delKeys)
			rootC = commitTrie()

			lenStats := make(map[int]int)
			size100Stats := make(map[int]int)
			for _, s := range addKeys {
				proof := model.Proof([]byte(s), tr1)
				err := proof.Validate(rootC)
				require.NoError(t, err)
				require.False(t, proof.IsProofOfAbsence())
				lenP := len(proof.Path)
				sizeP100 := trie.MustSize(proof) / 100
				//t.Logf("key: '%s', proof presence lenPlus1: %d", s, )
				//t.Logf("proof presence size = %d bytes", trie_go.MustSize(proof))

				l := lenStats[lenP]
				lenStats[lenP] = l + 1
				sz := size100Stats[sizeP100]
				size100Stats[sizeP100] = sz + 1
			}
			for _, s := range delKeys {
				proof := model.Proof([]byte(s), tr1)
				err := proof.Validate(rootC)
				require.NoError(t, err)
				require.True(t, proof.IsProofOfAbsence())
				//t.Logf("key: '%s', proof absence len: %d", s, len(proof.Path))
				//t.Logf("proof absence size = %d bytes", trie_go.MustSize(proof))
			}
			for i := 0; i < 5000; i++ {
				if i < 10 {
					t.Logf("len[%d] = %d", i, lenStats[i])
				}
				if size100Stats[i] != 0 {
					t.Logf("size[%d] = %d", i*100, size100Stats[i])
				}
			}
		})
		t.Run("reconcile"+" "+arity.String(), func(t *testing.T) {
			data = genRnd4()
			t.Logf("data len = %d", len(data))
			store := trie.NewInMemoryKVStore()
			for _, s := range data {
				store.Set([]byte("1"+s), []byte(s+"2"))
			}
			trieStore := trie.NewInMemoryKVStore()
			tr1 = trie.New(model, trieStore)
			store.Iterate(func(k, v []byte) bool {
				tr1.Update([]byte(k), v)
				return true
			})
			tr1.Commit()
			diff := tr1.Reconcile(store)
			require.EqualValues(t, 0, len(diff))
		})
	}
	runTest(trie.PathArity256)
	runTest(trie.PathArity16)
	runTest(trie.PathArity2)
}

func ar(arity trie.PathArity) string {
	return "-" + arity.String()
}

func TestTrieProofWithDeletesBlake2b20(t *testing.T) {
	var tr1 *trie.Trie
	var rootC trie.VCommitment
	var Model *trie_blake2b.CommitmentModel

	initTrie := func(dataAdd []string, arity trie.PathArity) {
		Model = trie_blake2b.New(arity, trie_blake2b.HashSize160)
		store := trie.NewInMemoryKVStore()
		tr1 = trie.New(Model, store)
		for _, s := range dataAdd {
			tr1.Update([]byte(s), []byte(s+"++"))
		}
	}
	deleteKeys := func(keysDelete []string) {
		for _, s := range keysDelete {
			tr1.Update([]byte(s), nil)
		}
	}
	commitTrie := func() trie.VCommitment {
		tr1.Commit()
		return trie.RootCommitment(tr1)
	}
	data := []string{"a", "ab", "ac", "abc", "abd", "ad", "ada", "adb", "adc", "c", "ad+dddgsssisd"}
	runTest := func(arity trie.PathArity) {
		t.Run("proof many entries 1"+ar(arity), func(t *testing.T) {
			initTrie(data, arity)
			rootC = commitTrie()
			for _, s := range data {
				proof := Model.Proof([]byte(s), tr1)
				require.False(t, proof.IsProofOfAbsence())
				err := proof.Validate(rootC)
				require.NoError(t, err)
				//t.Logf("key: '%s', proof len: %d", s, len(proof.Path))
				//t.Logf("proof presence size = %d bytes", trie_go.MustSize(proof))
			}
		})
		t.Run("proof many entries 2"+ar(arity), func(t *testing.T) {
			delKeys := []string{"1", "2", "3", "12345", "ab+", "ada+"}
			initTrie(data, arity)
			deleteKeys(delKeys)
			rootC = commitTrie()

			for _, s := range data {
				proof := Model.Proof([]byte(s), tr1)
				err := proof.Validate(rootC)
				require.NoError(t, err)
				require.False(t, proof.IsProofOfAbsence())
				//t.Logf("key: '%s', proof presence lenPlus1: %d", s, len(proof.Path))
				//t.Logf("proof presence size = %d bytes", trie_go.MustSize(proof))
			}
			for _, s := range delKeys {
				proof := Model.Proof([]byte(s), tr1)
				err := proof.Validate(rootC)
				require.NoError(t, err)
				require.True(t, proof.IsProofOfAbsence())
				//t.Logf("key: '%s', proof absence lenPlus1: %d", s, len(proof.Path))
				//t.Logf("proof absence size = %d bytes", trie_go.MustSize(proof))
			}
		})
		t.Run("proof many entries 3"+ar(arity), func(t *testing.T) {
			delKeys := []string{"1", "2", "3", "12345", "ab+", "ada+"}
			allData := make([]string, 0, len(data)+len(delKeys))
			allData = append(allData, data...)
			allData = append(allData, delKeys...)
			initTrie(allData, arity)
			deleteKeys(delKeys)
			rootC = commitTrie()

			for _, s := range data {
				proof := Model.Proof([]byte(s), tr1)
				err := proof.Validate(rootC)
				require.NoError(t, err)
				require.False(t, proof.IsProofOfAbsence())
				//t.Logf("key: '%s', proof presence lenPlus1: %d", s, len(proof.Path))
				sz := trie.MustSize(proof)
				//t.Logf("proof presence size = %d bytes", sz)

				proofBin := proof.Bytes()
				require.EqualValues(t, len(proofBin), sz)
				proofBack, err := trie_blake2b.ProofFromBytes(proofBin)
				require.NoError(t, err)
				err = proofBack.Validate(rootC)
				require.NoError(t, err)
				require.True(t, bytes.Equal(proof.Key, proofBack.Key))
				require.False(t, proofBack.IsProofOfAbsence())
			}
			for _, s := range delKeys {
				proof := Model.Proof([]byte(s), tr1)
				err := proof.Validate(rootC)
				require.NoError(t, err)
				require.True(t, proof.IsProofOfAbsence())
				//t.Logf("key: '%s', proof absence lenPlus1: %d", s, len(proof.Path))
				sz := trie.MustSize(proof)
				//t.Logf("proof absence size = %d bytes", sz)

				proofBin := proof.Bytes()
				require.EqualValues(t, len(proofBin), sz)
				proofBack, err := trie_blake2b.ProofFromBytes(proofBin)
				require.NoError(t, err)
				err = proofBack.Validate(rootC)
				require.NoError(t, err)
				require.True(t, bytes.Equal(proof.Key, proofBack.Key))
				require.True(t, proofBack.IsProofOfAbsence())
			}
		})
		t.Run("proof many entries rnd"+ar(arity), func(t *testing.T) {
			addKeys, delKeys := gen2different(100000)
			t.Logf("lenPlus1 adds: %d, lenPlus1 dels: %d", len(addKeys), len(delKeys))
			allData := make([]string, 0, len(addKeys)+len(delKeys))
			allData = append(allData, addKeys...)
			allData = append(allData, delKeys...)
			initTrie(allData, arity)
			deleteKeys(delKeys)
			rootC = commitTrie()

			lenStats := make(map[int]int)
			size100Stats := make(map[int]int)
			for _, s := range addKeys {
				proof := Model.Proof([]byte(s), tr1)
				err := proof.Validate(rootC)
				require.NoError(t, err)
				require.False(t, proof.IsProofOfAbsence())
				lenP := len(proof.Path)
				sizeP100 := trie.MustSize(proof) / 100
				//t.Logf("key: '%s', proof presence lenPlus1: %d", s, )
				//t.Logf("proof presence size = %d bytes", trie_go.MustSize(proof))

				l := lenStats[lenP]
				lenStats[lenP] = l + 1
				sz := size100Stats[sizeP100]
				size100Stats[sizeP100] = sz + 1
			}
			for _, s := range delKeys {
				proof := Model.Proof([]byte(s), tr1)
				err := proof.Validate(rootC)
				require.NoError(t, err)
				require.True(t, proof.IsProofOfAbsence())
				//t.Logf("key: '%s', proof absence len: %d", s, len(proof.Path))
				//t.Logf("proof absence size = %d bytes", trie_go.MustSize(proof))
			}
			for i := 0; i < 5000; i++ {
				if i < 10 {
					t.Logf("len[%d] = %d", i, lenStats[i])
				}
				if size100Stats[i] != 0 {
					t.Logf("size[%d] = %d", i*100, size100Stats[i])
				}
			}
		})
		t.Run("reconcile"+ar(arity), func(t *testing.T) {
			data = genRnd4()
			t.Logf("data len = %d", len(data))
			store := trie.NewInMemoryKVStore()
			for _, s := range data {
				store.Set([]byte("1"+s), []byte(s+"2"))
			}
			trieStore := trie.NewInMemoryKVStore()
			tr1 = trie.New(Model, trieStore)
			store.Iterate(func(k, v []byte) bool {
				tr1.Update(k, v)
				return true
			})
			tr1.Commit()
			diff := tr1.Reconcile(store)
			require.EqualValues(t, 0, len(diff))
		})
	}
	runTest(trie.PathArity256)
	runTest(trie.PathArity16)
	runTest(trie.PathArity2)
}

func TestTrieProofKZG(t *testing.T) {
	Model := trie_kzg_bn256.New()
	t.Run("proof empty trie"+" ", func(t *testing.T) {
		store := trie.NewInMemoryKVStore()
		tr := trie.New(Model, store)
		require.EqualValues(t, nil, trie.RootCommitment(tr))

		proof, ok := Model.ProofOfInclusion(nil, tr)
		require.False(t, ok)
		require.Nil(t, proof)
	})
	t.Run("proof one entry 1", func(t *testing.T) {
		store := trie.NewInMemoryKVStore()
		tr := trie.New(Model, store)

		tr.Update(nil, []byte("1"))
		tr.Commit()

		proof, ok := Model.ProofOfInclusion(nil, tr)
		require.True(t, ok)
		require.EqualValues(t, 1, len(proof.Path))

		rootC := trie.RootCommitment(tr)
		err := proof.Validate(rootC)
		require.NoError(t, err)

		t.Logf("proof size = %d bytes", trie.MustSize(proof))
	})
	t.Run("proof one entry 2", func(t *testing.T) {
		store := trie.NewInMemoryKVStore()
		tr := trie.New(Model, store)

		tr.Update([]byte("100"), []byte("1"))
		tr.Commit()

		proof, ok := Model.ProofOfInclusion([]byte("100"), tr)
		require.True(t, ok)
		require.EqualValues(t, 1, len(proof.Path))

		rootC := trie.RootCommitment(tr)
		err := proof.Validate(rootC)
		require.NoError(t, err)

		t.Logf("proof size = %d bytes", trie.MustSize(proof))
	})
	t.Run("proof some entries", func(t *testing.T) {
		store := trie.NewInMemoryKVStore()
		tr := trie.New(Model, store)

		//data := genRnd4()[:1000]
		data := []string{"a", "ab", "abc", "ac", "acb", "adb", "bcdddd"}

		for _, d := range data {
			tr.Update([]byte(d), []byte("1"+d))
		}
		tr.Commit()

		rootC := trie.RootCommitment(tr)

		for _, d := range data {
			poi, ok := Model.ProofOfInclusion([]byte(d), tr)
			require.True(t, ok)

			err := poi.Validate(rootC)
			require.NoError(t, err)
		}

		tr.DeleteStr("ab")
		_, ok := Model.ProofOfInclusion([]byte("ab"), tr)
		require.False(t, ok)
	})
	t.Run("proof many entries", func(t *testing.T) {
		store := trie.NewInMemoryKVStore()
		tr := trie.New(Model, store)

		data := genRnd4()[:00]

		for _, d := range data {
			tr.Update([]byte(d), []byte("1"+d))
		}
		tr.Commit()

		rootC := trie.RootCommitment(tr)

		for _, d := range data {
			//t.Logf("POI #%d': key = %s'", i, d)
			poi, ok := Model.ProofOfInclusion([]byte(d), tr)
			require.True(t, ok)

			err := poi.Validate(rootC)
			require.NoError(t, err)
		}
	})
}
