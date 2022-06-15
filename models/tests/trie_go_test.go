package tests

import (
	"bytes"
	"fmt"
	"github.com/iotaledger/trie.go/models/trie_blake2b"
	"github.com/iotaledger/trie.go/models/trie_kzg_bn256"
	"github.com/iotaledger/trie.go/trie"
	"github.com/stretchr/testify/require"
	"io"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func tn(m trie.CommitmentModel) string {
	return "-" + m.ShortName()
}

func TestNode(t *testing.T) {
	runTest := func(t *testing.T, m trie.CommitmentModel) {
		t.Run("base normal"+tn(m), func(t *testing.T) {
			n := trie.NewNodeData()
			err := n.Write(io.Discard, m.PathArity(), false, false)
			require.Error(t, err)

			unpackedKey := trie.UnpackBytes([]byte("a"), m.PathArity())
			unpackedValue := trie.UnpackBytes([]byte("b"), m.PathArity())

			var buf bytes.Buffer
			n = trie.NewNodeData()
			n.Terminal = m.CommitToData(unpackedValue)
			err = n.Write(&buf, m.PathArity(), false, false)
			require.NoError(t, err)

			nBack, err := trie.NodeDataFromBytes(m, buf.Bytes(), unpackedKey, m.PathArity(), nil)
			require.NoError(t, err)
			require.True(t, trie.EqualCommitments(n.Terminal, nBack.Terminal))

			h := m.CalcNodeCommitment(n)
			hBack := m.CalcNodeCommitment(nBack)
			require.EqualValues(t, h, hBack)
			t.Logf("commitment = %s", h)
		})
		t.Run("base key commitment"+tn(m), func(t *testing.T) {
			unpackedKey := trie.UnpackBytes([]byte("abc"), m.PathArity())
			unpackedPathFragment := trie.UnpackBytes([]byte("d"), m.PathArity())
			unpackedValue := trie.UnpackBytes([]byte("abcd"), m.PathArity())
			require.EqualValues(t, unpackedValue, trie.Concat(unpackedKey, unpackedPathFragment))

			var buf bytes.Buffer
			n := trie.NewNodeData()
			n.PathFragment = unpackedPathFragment
			n.Terminal = m.CommitToData(unpackedValue)
			err := n.Write(&buf, m.PathArity(), true, false)
			require.NoError(t, err)

			nBack, err := trie.NodeDataFromBytes(m, buf.Bytes(), unpackedKey, m.PathArity(), nil)
			require.NoError(t, err)
			require.EqualValues(t, n.PathFragment, nBack.PathFragment)
			require.True(t, trie.EqualCommitments(n.Terminal, nBack.Terminal))

			h := m.CalcNodeCommitment(n)
			hBack := m.CalcNodeCommitment(nBack)
			require.True(t, trie.EqualCommitments(h, hBack))
			t.Logf("commitment = %s", h)
		})
		t.Run("base short terminal"+tn(m), func(t *testing.T) {
			n := trie.NewNodeData()
			n.PathFragment = trie.UnpackBytes([]byte("kuku"), m.PathArity())
			n.Terminal = m.CommitToData(trie.UnpackBytes([]byte("data"), m.PathArity()))

			var buf bytes.Buffer
			err := n.Write(&buf, m.PathArity(), false, false)
			require.NoError(t, err)

			nBack, err := trie.NodeDataFromBytes(m, buf.Bytes(), nil, m.PathArity(), nil)
			require.NoError(t, err)

			h := m.CalcNodeCommitment(n)
			hBack := m.CalcNodeCommitment(nBack)
			require.EqualValues(t, h, hBack)
			t.Logf("commitment = %s", h)
		})
		t.Run("base long terminal"+tn(m), func(t *testing.T) {
			n := trie.NewNodeData()
			n.PathFragment = trie.UnpackBytes([]byte("kuku"), m.PathArity())
			n.Terminal = m.CommitToData(trie.UnpackBytes([]byte(strings.Repeat("data", 1000)), m.PathArity()))

			var buf bytes.Buffer
			err := n.Write(&buf, m.PathArity(), false, false)
			require.NoError(t, err)

			nBack, err := trie.NodeDataFromBytes(m, buf.Bytes(), nil, m.PathArity(), nil)
			require.NoError(t, err)

			h := m.CalcNodeCommitment(n)
			hBack := m.CalcNodeCommitment(nBack)
			require.EqualValues(t, h, hBack)
			t.Logf("commitment = %s", h)
		})
	}
	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize160))
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize160))
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize160))

	runTest(t, trie_kzg_bn256.New())
}

const letters = "abcdefghijklmnop"

func genData1() []string {
	ret := make([]string, 0, len(letters)*len(letters)*len(letters))
	for i := range letters {
		for j := range letters {
			for k := range letters {
				ret = append(ret, string([]byte{letters[i], letters[j], letters[k]}))
			}
		}
	}
	return ret
}

func genData2() []string {
	ret := make([]string, 0, len(letters)*len(letters)*len(letters))
	for i := range letters {
		for j := range letters {
			for k := range letters {
				s := string([]byte{letters[i], letters[j], letters[k]})
				ret = append(ret, s+s+s+s)
			}
		}
	}
	return ret
}

func TestTrieBase(t *testing.T) {
	var data1 = []string{"", "1", "2"}
	var data2 = []string{"a", "ab", "ac", "abc", "abd", "ad", "ada", "adb", "adc", "c", "abcd", "abcde", "abcdef", "ab"}
	var data3 = []string{"", "a", "ab", "abc", "abcd", "abcdAAA", "abd", "abe", "abcd"}
	runTest := func(t *testing.T, m trie.CommitmentModel) {
		t.Run("base1"+tn(m), func(t *testing.T) {
			store := trie.NewInMemoryKVStore()
			tr := trie.New(m, store, nil)
			require.EqualValues(t, nil, trie.RootCommitment(tr))

			tr.Update([]byte(""), []byte(""))
			tr.Commit()
			require.EqualValues(t, nil, trie.RootCommitment(tr))
			t.Logf("root0 = %s", trie.RootCommitment(tr))
			_, ok := tr.GetNode(nil)
			require.False(t, ok)

			tr.Update([]byte(""), []byte("0"))
			tr.Commit()
			t.Logf("root0 = %s", trie.RootCommitment(tr))
			c := trie.RootCommitment(tr)
			rootNode, ok := tr.GetNode(nil)
			require.True(t, ok)
			require.EqualValues(t, c, tr.Model().CalcNodeCommitment(&trie.NodeData{
				PathFragment:     rootNode.PathFragment(),
				ChildCommitments: rootNode.ChildCommitments(),
				Terminal:         rootNode.Terminal(),
			}))
		})
		t.Run("base2"+tn(m), func(t *testing.T) {
			data := data1
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
			}
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)

			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("base2-rev"+tn(m), func(t *testing.T) {
			data := data1
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
			}
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			for j := range data {
				i := len(data) - j - 1
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)

			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("base2-1"+tn(m), func(t *testing.T) {
			data := []string{"a", "ab", "abc"}
			t.Logf("%+v", data)
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
			}
			t.Logf("FIRST:\n%s", tr1.DangerouslyDumpCacheToString())

			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			t.Logf("SECOND:\n%s", tr2.DangerouslyDumpCacheToString())
			c2 := trie.RootCommitment(tr2)

			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("base2-2"+tn(m), func(t *testing.T) {
			data := data3
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
			}
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)

			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("base3"+tn(m), func(t *testing.T) {
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			data := data2[:5]
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
			}
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("base4"+tn(m), func(t *testing.T) {
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			data := []string{"001", "002", "010"}
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
			}
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			require.True(t, trie.EqualCommitments(c1, c2))
		})

		t.Run("reverse short"+tn(m), func(t *testing.T) {
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			tr1.Update([]byte("a"), []byte("k"))
			tr1.Update([]byte("ab"), []byte("l"))
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			tr2.Update([]byte("ab"), []byte("l"))
			tr2.Update([]byte("a"), []byte("k"))
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("reverse full"+tn(m), func(t *testing.T) {
			data := data2
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			for i := len(data) - 1; i >= 0; i-- {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("reverse long"+tn(m), func(t *testing.T) {
			data := genData1()
			require.EqualValues(t, 16*16*16, len(data))

			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			for i := len(data) - 1; i >= 0; i-- {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("deletion edge cases 1"+tn(m), func(t *testing.T) {
			store := trie.NewInMemoryKVStore()
			tr := trie.New(m, store, nil)

			tr.UpdateStr("ab1", []byte("1"))
			tr.UpdateStr("ab2c", []byte("2"))
			tr.DeleteStr("ab2a")
			tr.UpdateStr("ab4", []byte("4"))
			tr.Commit()
			c1 := trie.RootCommitment(tr)

			store = trie.NewInMemoryKVStore()
			tr = trie.New(m, store, nil)

			tr.UpdateStr("ab1", []byte("1"))
			tr.UpdateStr("ab2c", []byte("2"))
			tr.UpdateStr("ab4", []byte("4"))
			tr.Commit()
			c2 := trie.RootCommitment(tr)

			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("deletion edge cases 2"+tn(m), func(t *testing.T) {
			store := trie.NewInMemoryKVStore()
			tr := trie.New(m, store, nil)

			tr.UpdateStr("abc", []byte("1"))
			tr.UpdateStr("abcd", []byte("2"))
			tr.UpdateStr("abcde", []byte("2"))
			tr.DeleteStr("abcde")
			tr.DeleteStr("abcd")
			tr.DeleteStr("abc")
			tr.Commit()
			c1 := trie.RootCommitment(tr)

			store = trie.NewInMemoryKVStore()
			tr = trie.New(m, store, nil)

			tr.UpdateStr("abc", []byte("1"))
			tr.UpdateStr("abcd", []byte("2"))
			tr.UpdateStr("abcde", []byte("2"))
			tr.DeleteStr("abcde")
			tr.Commit()
			tr.DeleteStr("abcd")
			tr.DeleteStr("abc")
			tr.Commit()
			c2 := trie.RootCommitment(tr)

			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("deletion edge cases 3"+tn(m), func(t *testing.T) {
			store := trie.NewInMemoryKVStore()
			tr := trie.New(m, store, nil)

			tr.UpdateStr("abcd", []byte("1"))
			tr.UpdateStr("ab1234", []byte("2"))
			tr.DeleteStr("ab1234")
			tr.Commit()
			c1 := trie.RootCommitment(tr)

			store = trie.NewInMemoryKVStore()
			tr = trie.New(m, store, nil)

			tr.UpdateStr("abcd", []byte("1"))
			tr.UpdateStr("ab1234", []byte("2"))
			tr.Commit()
			tr.DeleteStr("ab1234")
			tr.Commit()
			c2 := trie.RootCommitment(tr)

			require.True(t, trie.EqualCommitments(c1, c2))
		})
	}

	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize160))
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize160))
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize160))

	runTest(t, trie_kzg_bn256.New())
}

func genRnd3() []string {
	ret := make([]string, 0, len(letters)*len(letters)*len(letters))
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range letters {
		for j := range letters {
			for k := range letters {
				s := string([]byte{letters[i], letters[j], letters[k]})
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
	return ret
}

func genRnd4() []string {
	ret := make([]string, 0, len(letters)*len(letters)*len(letters))
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range letters {
		for j := range letters {
			for k := range letters {
				for l := range letters {
					s := string([]byte{letters[i], letters[j], letters[k], letters[l]})
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
	if len(ret) > math.MaxUint16 {
		ret = ret[:math.MaxUint16]
	}
	return ret
}

func genDels(data []string, num int) []string {
	ret := make([]string, 0, num)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < num; i++ {
		ret = append(ret, data[rnd.Intn(len(data))])
	}
	return ret
}

func gen2different(n int) ([]string, []string) {
	orig := genRnd4()
	// filter different
	unique := make(map[string]bool)
	for _, s := range orig {
		unique[s] = true
	}
	ret1 := make([]string, 0)
	ret2 := make([]string, 0)
	for s := range unique {
		if rand.Intn(10000) > 1000 {
			ret1 = append(ret1, s)
		} else {
			ret2 = append(ret2, s)
		}
		if len(ret1)+len(ret2) > n {
			break
		}
	}
	return ret1, ret2
}

func TestTrieRnd(t *testing.T) {
	runTest := func(t *testing.T, m trie.CommitmentModel, shortData bool) {
		t.Run("determinism1"+tn(m), func(t *testing.T) {
			data := genData1()
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
			permutation := rnd.Perm(len(data))
			for _, i := range permutation {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("determinism2"+tn(m), func(t *testing.T) {
			data := genData2()
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
			permutation := rnd.Perm(len(data))
			for _, i := range permutation {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("determinism3"+tn(m), func(t *testing.T) {
			data := genRnd3()
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
			permutation := rnd.Perm(len(data))
			for _, i := range permutation {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("determinism4"+tn(m), func(t *testing.T) {
			data := genRnd4()
			if shortData {
				data = data[:1000]
			}
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
			permutation := rnd.Perm(len(data))
			for _, i := range permutation {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie.EqualCommitments(c1, c2))

			tr2.PersistMutations(store2)
			trieSize := trie.ByteSize(store2)
			numEntries := trie.NumEntries(store2)
			t.Logf("key entries = %d", len(data))
			t.Logf("Trie entries = %d", numEntries)
			t.Logf("Trie bytes = %d KB", trieSize/1024)
			t.Logf("Trie bytes/entry = %d ", trieSize/numEntries)
		})
		t.Run("determinism5"+tn(m), func(t *testing.T) {
			data := genRnd4()
			if shortData {
				data = data[:1000]
			}
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
			}
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)
			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}

			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie.EqualCommitments(c1, c2))
		})
	}

	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize256), false)
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize256), false)
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize256), false)
	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize160), false)
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize160), false)
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize160), false)

	runTest(t, trie_kzg_bn256.New(), true)
}

func TestTrieRndKeyCommitment(t *testing.T) {
	runTest := func(t *testing.T, m trie.CommitmentModel, shortData bool) {
		t.Run("determ key commitment1"+tn(m), func(t *testing.T) {
			data := genData1()
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for _, d := range data {
				if len(d) > 0 {
					tr1.InsertKeyCommitment([]byte(d))
				}
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)
			permutation := rand.Perm(len(data))
			for _, i := range permutation {
				if len(data[i]) > 0 {
					tr2.InsertKeyCommitment([]byte(data[i]))
				}
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("determ key commitment2"+tn(m), func(t *testing.T) {
			data := genData2()
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				if len(data[i]) > 0 {
					tr1.InsertKeyCommitment([]byte(data[i]))
				}
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			permutation := rand.Perm(len(data))
			for _, i := range permutation {
				if len(data[i]) > 0 {
					tr2.InsertKeyCommitment([]byte(data[i]))
				}
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("determ key commitment3"+tn(m), func(t *testing.T) {
			data := genRnd3()
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				if len(data[i]) > 0 {
					tr1.InsertKeyCommitment([]byte(data[i]))
				}
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			permutation := rand.Perm(len(data))
			for _, i := range permutation {
				if len(data[i]) > 0 {
					tr2.InsertKeyCommitment([]byte(data[i]))
				}
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("determ key commitment4"+tn(m), func(t *testing.T) {
			data := genRnd4()
			if shortData {
				data = data[:1000]
			}
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				if len(data[i]) > 0 {
					tr1.InsertKeyCommitment([]byte(data[i]))
				}
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)

			permutation := rand.Perm(len(data))
			for _, i := range permutation {
				if len(data[i]) > 0 {
					tr2.InsertKeyCommitment([]byte(data[i]))
				}
			}
			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie.EqualCommitments(c1, c2))

			tr2.PersistMutations(store2)
			trieSize := trie.ByteSize(store2)
			numEntries := trie.NumEntries(store2)
			t.Logf("key entries = %d", len(data))
			t.Logf("Trie entries = %d", numEntries)
			t.Logf("Trie bytes = %d KB", trieSize/1024)
			t.Logf("Trie bytes/entry = %d ", trieSize/numEntries)
		})
		t.Run("determ key commitment5"+tn(m), func(t *testing.T) {
			data := genRnd4()
			if shortData {
				data = data[:1000]
			}
			store1 := trie.NewInMemoryKVStore()
			tr1 := trie.New(m, store1, nil)

			for i := range data {
				if len(data[i]) > 0 {
					tr1.InsertKeyCommitment([]byte(data[i]))
				}
				tr1.Commit()
			}
			c1 := trie.RootCommitment(tr1)

			store2 := trie.NewInMemoryKVStore()
			tr2 := trie.New(m, store2, nil)
			for i := range data {
				if len(data[i]) > 0 {
					tr2.InsertKeyCommitment([]byte(data[i]))
				}
			}

			tr2.Commit()
			c2 := trie.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie.EqualCommitments(c1, c2))
		})
	}

	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize256), false)
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize256), false)
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize256), false)
	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize160), false)
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize160), false)
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize160), false)

	runTest(t, trie_kzg_bn256.New(), true)
}

func TestKeyCommitmentOptimization(t *testing.T) {
	data := genRnd4()[:10_000]
	runTest := func(model trie.CommitmentModel) {
		t.Run(tn(model), func(t *testing.T) {
			store1 := trie.NewInMemoryKVStore()
			store2 := trie.NewInMemoryKVStore()
			tr1 := trie.New(model, store1, nil, true)
			tr2 := trie.New(model, store2, nil, false)

			for _, d := range data {
				if len(d) > 0 {
					tr1.InsertKeyCommitment([]byte(d))
				}
			}
			tr1.Commit()
			tr1.PersistMutations(store1)

			for _, d := range data {
				b := []byte(d)
				if len(d) > 0 {
					b[0] = b[0] + 1 // make different from the key but same length
					tr2.Update([]byte(d), b)
				}
			}
			tr2.Commit()
			tr2.PersistMutations(store2)

			size1 := trie.ByteSize(store1)
			size2 := trie.ByteSize(store2)
			numEntries := trie.NumEntries(store1)
			require.EqualValues(t, numEntries, trie.NumEntries(store2))

			require.True(t, size1 < size2)
			t.Logf("num entries: %d", numEntries)
			t.Logf("   with key commitments. Byte size: %d, avg: %f bytes per entry", size1, float32(size1)/float32(numEntries))
			t.Logf("without key commitments. Byte size: %d, avg: %f bytes per entry", size2, float32(size2)/float32(numEntries))
		})
	}

	runTest(trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize256))
	runTest(trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize256))
	runTest(trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize256))
	runTest(trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize160))
	runTest(trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize160))
	runTest(trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize160))

	runTest(trie_kzg_bn256.New())
}

func TestKeyCommitmentOptimizationOptions(t *testing.T) {
	data := genRnd4()[:10_000]
	runTest := func(model trie.CommitmentModel, optKeys bool) int {
		store1 := trie.NewInMemoryKVStore()
		tr1 := trie.New(model, store1, nil, optKeys)

		for _, d := range data {
			if len(d) > 0 {
				tr1.InsertKeyCommitment([]byte(d))
			}
		}
		tr1.Commit()
		tr1.PersistMutations(store1)

		return trie.ByteSize(store1)
	}
	arity := trie.PathArity256
	size1 := runTest(trie_blake2b.New(arity, trie_blake2b.HashSize256), true)
	size2 := runTest(trie_blake2b.New(arity, trie_blake2b.HashSize256), false)
	t.Logf("   with key commitment optimization. Arity: %s, Byte size: %d", arity, size1)
	t.Logf("without key commitment optimization. Arity: %s, Byte size: %d", arity, size2)
	require.True(t, size1 < size2)

	arity = trie.PathArity16
	size1 = runTest(trie_blake2b.New(arity, trie_blake2b.HashSize256), true)
	size2 = runTest(trie_blake2b.New(arity, trie_blake2b.HashSize256), false)
	t.Logf("   with key commitment optimization. Arity: %s, Byte size: %d", arity, size1)
	t.Logf("without key commitment optimization. Arity: %s, Byte size: %d", arity, size2)
	require.True(t, size1 < size2)

	arity = trie.PathArity2
	size1 = runTest(trie_blake2b.New(arity, trie_blake2b.HashSize256), true)
	size2 = runTest(trie_blake2b.New(arity, trie_blake2b.HashSize256), false)
	t.Logf("   with key commitment optimization. Arity: %s, Byte size: %d", arity, size1)
	t.Logf("without key commitment optimization. Arity: %s, Byte size: %d", arity, size2)
	require.True(t, size1 < size2)
}

func TestKeyCommitmentOptimizationOptions20(t *testing.T) {
	data := genRnd4()[:10_000]
	runTest := func(model trie.CommitmentModel, optKeys bool) int {
		store1 := trie.NewInMemoryKVStore()
		tr1 := trie.New(model, store1, nil, optKeys)

		for _, d := range data {
			if len(d) > 0 {
				tr1.InsertKeyCommitment([]byte(d))
			}
		}
		tr1.Commit()
		tr1.PersistMutations(store1)

		return trie.ByteSize(store1)
	}
	arity := trie.PathArity256
	size1 := runTest(trie_blake2b.New(arity, trie_blake2b.HashSize160), true)
	size2 := runTest(trie_blake2b.New(arity, trie_blake2b.HashSize160), false)
	t.Logf("   with key commitment optimization. Arity: %s, Byte size: %d", arity, size1)
	t.Logf("without key commitment optimization. Arity: %s, Byte size: %d", arity, size2)
	require.True(t, size1 < size2)

	arity = trie.PathArity16
	size1 = runTest(trie_blake2b.New(arity, trie_blake2b.HashSize160), true)
	size2 = runTest(trie_blake2b.New(arity, trie_blake2b.HashSize160), false)
	t.Logf("   with key commitment optimization. Arity: %s, Byte size: %d", arity, size1)
	t.Logf("without key commitment optimization. Arity: %s, Byte size: %d", arity, size2)
	require.True(t, size1 < size2)

	arity = trie.PathArity2
	size1 = runTest(trie_blake2b.New(arity, trie_blake2b.HashSize160), true)
	size2 = runTest(trie_blake2b.New(arity, trie_blake2b.HashSize160), false)
	t.Logf("   with key commitment optimization. Arity: %s, Byte size: %d", arity, size1)
	t.Logf("without key commitment optimization. Arity: %s, Byte size: %d", arity, size2)
	require.True(t, size1 < size2)
}

func Test20Vs32(t *testing.T) {
	data := genRnd4()[:10_000]
	runTest := func(model trie.CommitmentModel) int {
		store := trie.NewInMemoryKVStore()
		tr1 := trie.New(model, store, nil)

		for _, d := range data {
			if len(d) > 0 {
				tr1.InsertKeyCommitment([]byte(d))
			}
		}
		tr1.Commit()
		tr1.PersistMutations(store)

		return trie.ByteSize(store)
	}
	arity := trie.PathArity256
	size1 := runTest(trie_blake2b.New(arity, trie_blake2b.HashSize256))
	size2 := runTest(trie_blake2b.New(arity, trie_blake2b.HashSize160))
	t.Logf("with blake2b 32 byte. Byte size: %d, Arity: %s", size1, arity)
	t.Logf("with blake2b 20. Byte size: %d, Arity: %s", size2, arity)
	require.True(t, size2 < size1)

	arity = trie.PathArity16
	size1 = runTest(trie_blake2b.New(arity, trie_blake2b.HashSize256))
	size2 = runTest(trie_blake2b.New(arity, trie_blake2b.HashSize160))
	t.Logf("with blake2b 32 byte. Byte size: %d, Arity: %s", size1, arity)
	t.Logf("with blake2b 20. Byte size: %d, Arity: %s", size2, arity)
	require.True(t, size2 < size1)

	arity = trie.PathArity2
	size1 = runTest(trie_blake2b.New(arity, trie_blake2b.HashSize256))
	size2 = runTest(trie_blake2b.New(arity, trie_blake2b.HashSize160))
	t.Logf("with blake2b 32 byte. Byte size: %d, Arity: %s", size1, arity)
	t.Logf("with blake2b 20. Byte size: %d, Arity: %s", size2, arity)
	require.True(t, size2 < size1)
}

func TestTrieWithDeletion(t *testing.T) {
	data := []string{"0", "1", "2", "3", "4", "5"}
	var tr1, tr2 *trie.Trie
	runTest := func(t *testing.T, m trie.CommitmentModel) {
		initTest := func() {
			store1 := trie.NewInMemoryKVStore()
			tr1 = trie.New(m, store1, nil)
			store2 := trie.NewInMemoryKVStore()
			tr2 = trie.New(m, store2, nil)
		}
		t.Run("del1"+tn(m), func(t *testing.T) {
			initTest()
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Delete([]byte(data[1]))
			tr2.Update([]byte(data[1]), []byte(data[1]))
			tr2.Commit()
			c2 := trie.RootCommitment(tr1)

			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("del2"+tn(m), func(t *testing.T) {
			initTest()
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			tr2.Delete([]byte(data[1]))
			tr2.Update([]byte(data[1]), []byte(data[1]))
			tr2.Commit()
			c2 := trie.RootCommitment(tr1)

			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("del3"+tn(m), func(t *testing.T) {
			initTest()
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
				tr2.Commit()
			}
			tr2.Delete([]byte(data[1]))
			tr2.Update([]byte(data[1]), []byte(data[1]))
			tr2.Commit()
			c2 := trie.RootCommitment(tr1)

			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("del4"+tn(m), func(t *testing.T) {
			initTest()
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
				tr2.Commit()
			}
			tr2.Delete([]byte(data[1]))
			tr2.Commit()
			tr2.Update([]byte(data[1]), []byte(data[1]))
			tr2.Commit()
			c2 := trie.RootCommitment(tr1)

			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("del5"+tn(m), func(t *testing.T) {
			initTest()
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
				tr2.Commit()
			}
			c2 := trie.RootCommitment(tr1)
			require.True(t, trie.EqualCommitments(c1, c2))

			tr2.Delete([]byte(data[1]))
			tr2.Commit()

			c2 = trie.RootCommitment(tr2)
			require.False(t, trie.EqualCommitments(c1, c2))

			tr2.Update([]byte(data[1]), []byte(data[1]))
			tr2.Commit()
			c2 = trie.RootCommitment(tr1)

			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("del all"+tn(m), func(t *testing.T) {
			initTest()
			data = genData1()
			//data = data[:18]
			//data = []string{"001", "010", "012"}
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()

			for i := range data {
				tr1.Delete([]byte(data[i]))
			}
			tr1.Commit()

			c := trie.RootCommitment(tr1)
			require.EqualValues(t, nil, c)
		})
	}

	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize160))
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize160))
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize160))

	runTest(t, trie_kzg_bn256.New())
}

func TestTrieWithDeletionDeterm(t *testing.T) {
	data := []string{"0", "1", "2", "3", "4", "5"}
	var tr1, tr2 *trie.Trie
	runTest := func(t *testing.T, m trie.CommitmentModel, shortData bool) {
		initTest := func() {
			store1 := trie.NewInMemoryKVStore()
			tr1 = trie.New(m, store1, nil)
			store2 := trie.NewInMemoryKVStore()
			tr2 = trie.New(m, store2, nil)
		}
		t.Run("del determ 1"+tn(m), func(t *testing.T) {
			initTest()
			data = genRnd4()

			if shortData {
				data = data[:1000]
			}
			dels := genDels(data, len(data)/2)

			posDel := 0
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
				if i%3 == 0 {
					tr1.Delete([]byte(dels[posDel]))
					posDel = (posDel + 1) % len(dels)
				}
			}
			tr1.Commit()
			for i := range dels {
				tr1.Delete([]byte(dels[i]))
			}
			tr1.Commit()
			c1 := trie.RootCommitment(tr1)

			permutation := rand.Perm(len(data))
			for _, i := range permutation {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			for i := range dels {
				tr2.Delete([]byte(dels[i]))
			}
			tr2.Commit()

			c2 := trie.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie.EqualCommitments(c1, c2))
		})
		t.Run("del determ 2"+tn(m), func(t *testing.T) {
			initTest()
			data = genRnd4()
			if shortData {
				data = data[:1000]
			}
			t.Logf("data len = %d", len(data))

			const rounds = 5
			var c, cPrev trie.VCommitment

			rng := rand.New(rand.NewSource(time.Now().UnixNano()))
			for i := 0; i < rounds; i++ {
				t.Logf("-------- round #%d", i)
				perm := rng.Perm(len(data))
				for _, j := range perm {
					tr1.UpdateStr(data[j], []byte(data[j]))
				}
				tr1.Commit()
				if cPrev != nil {
					require.True(t, trie.EqualCommitments(c, cPrev))
				}
				perm = rng.Perm(len(data))
				for _, j := range perm {
					tr1.DeleteStr(data[j])
					if rng.Intn(1000) < 100 {
						tr1.Commit()
					}
				}
				tr1.Commit()
				cPrev = c
				c = trie.RootCommitment(tr1)
				require.True(t, trie.EqualCommitments(c, nil))
			}
		})
	}

	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize256), false)
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize256), false)
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize256), false)
	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize160), false)
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize160), false)
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize160), false)

	runTest(t, trie_kzg_bn256.New(), true)
}

type act struct {
	key    string
	del    bool
	commit [2]bool
}

var flow = []act{
	{"ab", false, [2]bool{false, false}},
	{"abc", false, [2]bool{false, false}},
	{"abc", true, [2]bool{false, true}},
	{"abcd1", false, [2]bool{false, false}},
}

func TestDeleteCommit(t *testing.T) {
	runTest := func(t *testing.T, m trie.CommitmentModel) {
		var c [2]trie.VCommitment
		for round := range []int{0, 1} {
			t.Logf("------- run %d", round)
			store := trie.NewInMemoryKVStore()
			tr := trie.New(m, store, nil)
			for i, a := range flow {
				if a.del {
					t.Logf("round %d: DEL '%s'", round, a.key)
					tr.DeleteStr(a.key)
				} else {
					t.Logf("round %d: SET '%s'", round, a.key)
					tr.UpdateStr(a.key, fmt.Sprintf("%s-%d", a.key, i))
				}
				if a.commit[round] {
					t.Logf("round %d: COMMIT ++++++", round)
					tr.Commit()
				}
			}
			tr.Commit()
			c[round] = trie.RootCommitment(tr)
			t.Logf("c[%d] = %s", round, c[round])
			diff := tr.Reconcile(store)
			require.EqualValues(t, 0, len(diff))
		}
		require.True(t, trie.EqualCommitments(c[0], c[1]))
	}

	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize160))
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize160))
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize160))

	runTest(t, trie_kzg_bn256.New())
}

func TestGenTrie(t *testing.T) {
	const filename = "$$for testing$$"
	runTest := func(t *testing.T, m trie.CommitmentModel) {
		kind := "blake2b"
		if _, ok := m.(*trie_kzg_bn256.CommitmentModel); ok {
			kind = "kzg"
		}
		fname := filename + "_" + kind

		t.Run("gen file "+kind, func(t *testing.T) {
			store := trie.NewInMemoryKVStore()
			data := genRnd4()
			for _, s := range data {
				store.Set([]byte(s), []byte("abcdefghijklmnoprstquwxyz"))
			}
			t.Logf("num records = %d", len(data))
			n, err := trie.DumpToFile(store, fname+".bin")
			require.NoError(t, err)
			t.Logf("wrote %d bytes to '%s'", n, fname+".bin")
		})
		t.Run("gen trie "+kind, func(t *testing.T) {
			store := trie.NewInMemoryKVStore()
			n, err := trie.UnDumpFromFile(store, fname+".bin")
			require.NoError(t, err)
			t.Logf("read %d bytes to '%s'", n, fname+".bin")

			storeTrie := trie.NewInMemoryKVStore()
			tr := trie.New(m, storeTrie, nil)
			tr.UpdateAll(store)
			tr.Commit()
			tr.PersistMutations(store)
			t.Logf("trie len = %d", trie.NumEntries(store))
			n, err = trie.DumpToFile(store, fname+".trie")
			require.NoError(t, err)
			t.Logf("dumped trie size = %d", n)
		})
	}

	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize256))
	runTest(t, trie_blake2b.New(trie.PathArity256, trie_blake2b.HashSize160))
	runTest(t, trie_blake2b.New(trie.PathArity16, trie_blake2b.HashSize160))
	runTest(t, trie_blake2b.New(trie.PathArity2, trie_blake2b.HashSize160))

	runTest(t, trie_kzg_bn256.New())
}
