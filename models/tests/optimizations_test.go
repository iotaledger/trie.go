package tests

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/iotaledger/trie.go/common"
	"github.com/iotaledger/trie.go/models/trie_blake2b"
	"github.com/iotaledger/trie.go/models/trie_blake2b/trie_blake2b_verify"
	"github.com/iotaledger/trie.go/mutable"
	"github.com/stretchr/testify/require"
)

const letters1 = "abcdefghijklmnop"

func genRndOpt() []string {
	ret := make([]string, 0, len(letters1)*len(letters1)*len(letters1))
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range letters1 {
		for j := range letters1 {
			for k := range letters1 {
				for l := range letters1 {
					s := string([]byte{letters1[i], letters1[j], letters1[k], letters1[l]})
					s = s + s + s + s
					r1 := rnd.Intn(len(s))
					r2 := rnd.Intn(len(s))
					if r2 < r1 {
						r1, r2 = r2, r1
					}
					ret = append(ret, s[r1:r2])
				}
			}
		}
	}
	return ret
}

func TestTerminalOptimizationOptions(t *testing.T) {
	data := genRndOpt()[:60_000]
	runOptions := func(arity common.PathArity, sz trie_blake2b.HashSize, thr int) (int, int) {
		var ret1, ret2 int
		tname := fmt.Sprintf("%s-%s-thr=%d", arity, sz, thr)
		t.Run(tname, func(t *testing.T) {
			trieStore1 := common.NewInMemoryKVStore()
			trieStore2 := common.NewInMemoryKVStore()
			valueStore := common.NewInMemoryKVStore()

			m1 := trie_blake2b.New(arity, sz)
			tr1 := mutable.NewTrie(m1, trieStore1, nil)

			m2 := trie_blake2b.New(arity, sz, thr)
			tr2 := mutable.NewTrie(m2, trieStore2, valueStore)

			for _, d := range data {
				if len(d) > 0 {
					k := []byte(d)
					v := []byte(strings.Repeat(d, 10))
					tr1.Update(k, v)
					tr2.Update(k, v)
					valueStore.Set(k, v)
				}
			}
			tr1.Commit()
			tr1.PersistMutations(trieStore1)
			tr1.ClearCache()

			tr2.Commit()
			tr2.PersistMutations(trieStore2)
			tr2.ClearCache()

			ret1 = common.ByteSize(trieStore1)
			ret2 = common.ByteSize(trieStore2)
			num := common.NumEntries(valueStore)
			t.Logf("valueStore size = %d, num entries = %d",
				common.ByteSize(valueStore), num)
			t.Logf("trieStore1 size = %d, %d bytes/entry", ret1, ret1/num)
			t.Logf("trieStore2 size = %d, %d bytes/entry", ret2, ret2/num)
			t.Logf("difference = %d bytes, %d%%", ret1-ret2, ((ret1 - ret2) * 100 / ret1))
		})
		return ret1, ret2
	}
	runAllOptions := func(fun func(arity common.PathArity, sz trie_blake2b.HashSize)) {
		for _, a := range common.AllPathArity {
			for _, sz := range trie_blake2b.AllHashSize {
				fun(a, sz)
			}
		}
	}
	runAllOptions(func(arity common.PathArity, sz trie_blake2b.HashSize) {
		size1, size2 := runOptions(arity, sz, 0)
		require.EqualValues(t, size1, size2)
	})
	runAllOptions(func(arity common.PathArity, sz trie_blake2b.HashSize) {
		size1, size2 := runOptions(arity, sz, 10)
		require.True(t, size2 < size1)
	})
	runAllOptions(func(arity common.PathArity, sz trie_blake2b.HashSize) {
		size1, size2 := runOptions(arity, sz, 10000)
		require.True(t, size2 < size1)
	})
}

func TestTrieProofWithDeletesBlake2b20AndTerminalOpt(t *testing.T) {
	var tr1 *mutable.Trie
	var rootC common.VCommitment
	var m *trie_blake2b.CommitmentModel
	var storeTrie, storeValue common.KVStore

	initRun := func(dataAdd []string, arity common.PathArity, thr int) {
		if thr < 0 {
			m = trie_blake2b.New(arity, trie_blake2b.HashSize160)
		} else {
			m = trie_blake2b.New(arity, trie_blake2b.HashSize160, thr)
		}
		storeTrie = common.NewInMemoryKVStore()
		storeValue = common.NewInMemoryKVStore()
		tr1 = mutable.NewTrie(m, storeTrie, storeValue)
		for _, s := range dataAdd {
			k := []byte(s)
			v := []byte(strings.Repeat(s, 10))
			tr1.Update(k, v)
			storeValue.Set(k, v)
		}
	}
	deleteKeys := func(keysDelete []string) {
		for _, s := range keysDelete {
			tr1.Update([]byte(s), nil)
			storeValue.Set([]byte(s), nil)
		}
	}
	commitTrie := func() common.VCommitment {
		tr1.Commit()
		tr1.PersistMutations(storeTrie)
		tr1.ClearCache()
		return mutable.RootCommitment(tr1)
	}
	data := []string{"a", "ab", "ac", "abc", "abd", "ad", "ada", "adb", "adc", "c", "ad+dddgsssisd"}
	runOptions := func(arity common.PathArity, thr int) {
		tname := fmt.Sprintf("-%s-thr=%d", arity, thr)
		t.Run("proof 1"+tname, func(t *testing.T) {
			initRun(data, arity, thr)
			rootC = commitTrie()
			for _, s := range data {
				proof := m.Proof([]byte(s), tr1)
				require.False(t, trie_blake2b_verify.IsProofOfAbsence(proof))
				err := trie_blake2b_verify.Validate(proof, rootC.Bytes())
				require.NoError(t, err)
				//t.Logf("key: '%s', proof len: %d", s, len(proof.Path))
				//t.Logf("proof presence size = %d bytes", trie_go.MustSize(proof))
			}
		})
		t.Run("proof 2"+tname, func(t *testing.T) {
			delKeys := []string{"1", "2", "3", "12345", "ab+", "ada+"}
			initRun(data, arity, thr)
			deleteKeys(delKeys)
			rootC = commitTrie()

			for _, s := range data {
				proof := m.Proof([]byte(s), tr1)
				err := trie_blake2b_verify.Validate(proof, rootC.Bytes())
				require.NoError(t, err)
				require.False(t, trie_blake2b_verify.IsProofOfAbsence(proof))
				//t.Logf("key: '%s', proof presence lenPlus1: %d", s, len(proof.Path))
				//t.Logf("proof presence size = %d bytes", trie_go.MustSize(proof))
			}
			for _, s := range delKeys {
				proof := m.Proof([]byte(s), tr1)
				err := trie_blake2b_verify.Validate(proof, rootC.Bytes())
				require.NoError(t, err)
				require.True(t, trie_blake2b_verify.IsProofOfAbsence(proof))
				//t.Logf("key: '%s', proof absence lenPlus1: %d", s, len(proof.Path))
				//t.Logf("proof absence size = %d bytes", trie_go.MustSize(proof))
			}
		})
		t.Run("proof 3"+tname, func(t *testing.T) {
			delKeys := []string{"1", "2", "3", "12345", "ab+", "ada+"}
			allData := make([]string, 0, len(data)+len(delKeys))
			allData = append(allData, data...)
			allData = append(allData, delKeys...)
			initRun(allData, arity, thr)
			deleteKeys(delKeys)
			rootC = commitTrie()

			for _, s := range data {
				proof := m.Proof([]byte(s), tr1)
				err := trie_blake2b_verify.Validate(proof, rootC.Bytes())
				require.NoError(t, err)
				require.False(t, trie_blake2b_verify.IsProofOfAbsence(proof))
				//t.Logf("key: '%s', proof presence lenPlus1: %d", s, len(proof.Path))
				sz := common.MustSize(proof)
				//t.Logf("proof presence size = %d bytes", sz)

				proofBin := proof.Bytes()
				require.EqualValues(t, len(proofBin), sz)
				proofBack, err := trie_blake2b.ProofFromBytes(proofBin)
				require.NoError(t, err)
				err = trie_blake2b_verify.Validate(proofBack, rootC.Bytes())
				require.NoError(t, err)
				require.True(t, bytes.Equal(proof.Key, proofBack.Key))
				require.False(t, trie_blake2b_verify.IsProofOfAbsence(proofBack))
			}
			for _, s := range delKeys {
				proof := m.Proof([]byte(s), tr1)
				err := trie_blake2b_verify.Validate(proof, rootC.Bytes())
				require.NoError(t, err)
				require.True(t, trie_blake2b_verify.IsProofOfAbsence(proof))
				//t.Logf("key: '%s', proof absence lenPlus1: %d", s, len(proof.Path))
				sz := common.MustSize(proof)
				//t.Logf("proof absence size = %d bytes", sz)

				proofBin := proof.Bytes()
				require.EqualValues(t, len(proofBin), sz)
				proofBack, err := trie_blake2b.ProofFromBytes(proofBin)
				require.NoError(t, err)
				err = trie_blake2b_verify.Validate(proofBack, rootC.Bytes())
				require.NoError(t, err)
				require.True(t, bytes.Equal(proof.Key, proofBack.Key))
				require.True(t, trie_blake2b_verify.IsProofOfAbsence(proofBack))
			}
		})
		t.Run("proof rnd"+tname, func(t *testing.T) {
			addKeys, delKeys := gen2different(100000)
			t.Logf("lenPlus1 adds: %d, lenPlus1 dels: %d", len(addKeys), len(delKeys))
			allData := make([]string, 0, len(addKeys)+len(delKeys))
			allData = append(allData, addKeys...)
			allData = append(allData, delKeys...)
			initRun(allData, arity, thr)
			deleteKeys(delKeys)
			rootC = commitTrie()

			lenStats := make(map[int]int)
			size100Stats := make(map[int]int)
			for _, s := range addKeys {
				proof := m.Proof([]byte(s), tr1)
				err := trie_blake2b_verify.Validate(proof, rootC.Bytes())
				require.NoError(t, err)
				require.False(t, trie_blake2b_verify.IsProofOfAbsence(proof))
				lenP := len(proof.Path)
				sizeP100 := common.MustSize(proof) / 100
				//t.Logf("key: '%s', proof presence lenPlus1: %d", s, )
				//t.Logf("proof presence size = %d bytes", trie_go.MustSize(proof))

				l := lenStats[lenP]
				lenStats[lenP] = l + 1
				sz := size100Stats[sizeP100]
				size100Stats[sizeP100] = sz + 1
			}
			for _, s := range delKeys {
				proof := m.Proof([]byte(s), tr1)
				err := trie_blake2b_verify.Validate(proof, rootC.Bytes())
				require.NoError(t, err)
				require.True(t, trie_blake2b_verify.IsProofOfAbsence(proof))
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
	}
	//runOptions(trie.PathArity256, 10000)
	for _, a := range common.AllPathArity {
		runOptions(a, -1)
		runOptions(a, 10)
		runOptions(a, 10000)
	}
}
