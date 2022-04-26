package trie_go_tests

import (
	"bytes"
	"fmt"
	trie_go "github.com/iotaledger/trie.go"
	"github.com/iotaledger/trie.go/trie256p"
	"github.com/iotaledger/trie.go/trie_blake2b"
	"github.com/iotaledger/trie.go/trie_kzg_bn256"
	"github.com/stretchr/testify/require"
	"io"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestNode(t *testing.T) {
	runTest := func(t *testing.T, m trie256p.CommitmentModel) {
		t.Run("base normal", func(t *testing.T) {
			n := trie256p.NewNodeData()
			err := n.Write(io.Discard, false)
			require.Error(t, err)

			key := []byte("a")
			value := []byte("b")

			var buf bytes.Buffer
			n = trie256p.NewNodeData()
			n.Terminal = m.CommitToData(value)
			err = n.Write(&buf, false)
			require.NoError(t, err)

			nBack, err := trie256p.NodeDataFromBytes(m, buf.Bytes(), key)
			require.NoError(t, err)
			require.True(t, trie_go.EqualCommitments(n.Terminal, nBack.Terminal))

			h := m.CalcNodeCommitment(n)
			hBack := m.CalcNodeCommitment(nBack)
			require.EqualValues(t, h, hBack)
			t.Logf("commitment = %s", h)
		})
		t.Run("base key commitment", func(t *testing.T) {
			key := []byte("abc")
			pathFragment := []byte("d")
			value := []byte("abcd")

			var buf bytes.Buffer
			n := trie256p.NewNodeData()
			n.PathFragment = pathFragment
			n.Terminal = m.CommitToData(value)
			err := n.Write(&buf, true)
			require.NoError(t, err)

			nBack, err := trie256p.NodeDataFromBytes(m, buf.Bytes(), key)
			require.NoError(t, err)
			require.EqualValues(t, n.PathFragment, nBack.PathFragment)
			require.True(t, trie_go.EqualCommitments(n.Terminal, nBack.Terminal))

			h := m.CalcNodeCommitment(n)
			hBack := m.CalcNodeCommitment(nBack)
			require.True(t, trie_go.EqualCommitments(h, hBack))
			t.Logf("commitment = %s", h)
		})
		t.Run("base short terminal", func(t *testing.T) {
			n := trie256p.NewNodeData()
			n.PathFragment = []byte("kuku")
			n.Terminal = m.CommitToData([]byte("data"))

			var buf bytes.Buffer
			err := n.Write(&buf, false)
			require.NoError(t, err)

			nBack, err := trie256p.NodeDataFromBytes(m, buf.Bytes(), nil)
			require.NoError(t, err)

			h := m.CalcNodeCommitment(n)
			hBack := m.CalcNodeCommitment(nBack)
			require.EqualValues(t, h, hBack)
			t.Logf("commitment = %s", h)
		})
		t.Run("base long terminal", func(t *testing.T) {
			n := trie256p.NewNodeData()
			n.PathFragment = []byte("kuku")
			n.Terminal = m.CommitToData([]byte(strings.Repeat("data", 1000)))

			var buf bytes.Buffer
			err := n.Write(&buf, false)
			require.NoError(t, err)

			nBack, err := trie256p.NodeDataFromBytes(m, buf.Bytes(), nil)
			require.NoError(t, err)

			h := m.CalcNodeCommitment(n)
			hBack := m.CalcNodeCommitment(nBack)
			require.EqualValues(t, h, hBack)
			t.Logf("commitment = %s", h)
		})
	}
	runTest(t, trie_blake2b.New())
	//runTest(t, trie_kzg_bn256.New())
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
	runTest := func(t *testing.T, m trie256p.CommitmentModel) {
		t.Run("base1", func(t *testing.T) {
			store := trie_go.NewInMemoryKVStore()
			tr := trie256p.New(m, store)
			require.EqualValues(t, nil, trie256p.RootCommitment(tr))

			tr.Update([]byte(""), []byte(""))
			tr.Commit()
			require.EqualValues(t, nil, trie256p.RootCommitment(tr))
			t.Logf("root0 = %s", trie256p.RootCommitment(tr))
			_, ok := tr.GetNode(nil)
			require.False(t, ok)

			tr.Update([]byte(""), []byte("0"))
			tr.Commit()
			t.Logf("root0 = %s", trie256p.RootCommitment(tr))
			c := trie256p.RootCommitment(tr)
			rootNode, ok := tr.GetNode(nil)
			require.True(t, ok)
			require.EqualValues(t, c, tr.Model().CalcNodeCommitment(&trie256p.NodeData{
				PathFragment:     rootNode.PathFragment(),
				ChildCommitments: rootNode.ChildCommitments(),
				Terminal:         rootNode.Terminal(),
			}))
		})
		t.Run("base2", func(t *testing.T) {
			data := data1
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
			}
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)

			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("base2-rev", func(t *testing.T) {
			data := data1
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
			}
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			for j := range data {
				i := len(data) - j - 1
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)

			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("base2-1", func(t *testing.T) {
			data := []string{"a", "ab", "abc"}
			t.Logf("%+v", data)
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
			}
			t.Logf("FIRST:\n%s", tr1.DangerouslyDumpCacheToString())

			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			t.Logf("SECOND:\n%s", tr2.DangerouslyDumpCacheToString())
			c2 := trie256p.RootCommitment(tr2)

			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("base2-2", func(t *testing.T) {
			data := data3
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
			}
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)

			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("base3", func(t *testing.T) {
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			data := data2[:5]
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
			}
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("base4", func(t *testing.T) {
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			data := []string{"001", "002", "010"}
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
			}
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			require.True(t, trie_go.EqualCommitments(c1, c2))
		})

		t.Run("reverse short", func(t *testing.T) {
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			tr1.Update([]byte("a"), []byte("k"))
			tr1.Update([]byte("ab"), []byte("l"))
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			tr2.Update([]byte("ab"), []byte("l"))
			tr2.Update([]byte("a"), []byte("k"))
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("reverse full", func(t *testing.T) {
			data := data2
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			for i := len(data) - 1; i >= 0; i-- {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("reverse long", func(t *testing.T) {
			data := genData1()
			require.EqualValues(t, 16*16*16, len(data))

			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			for i := len(data) - 1; i >= 0; i-- {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("deletion edge cases 1", func(t *testing.T) {
			store := trie_go.NewInMemoryKVStore()
			tr := trie256p.New(m, store)

			tr.UpdateStr("ab1", []byte("1"))
			tr.UpdateStr("ab2c", []byte("2"))
			tr.DeleteStr("ab2a")
			tr.UpdateStr("ab4", []byte("4"))
			tr.Commit()
			c1 := trie256p.RootCommitment(tr)

			store = trie_go.NewInMemoryKVStore()
			tr = trie256p.New(m, store)

			tr.UpdateStr("ab1", []byte("1"))
			tr.UpdateStr("ab2c", []byte("2"))
			tr.UpdateStr("ab4", []byte("4"))
			tr.Commit()
			c2 := trie256p.RootCommitment(tr)

			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("deletion edge cases 2", func(t *testing.T) {
			store := trie_go.NewInMemoryKVStore()
			tr := trie256p.New(m, store)

			tr.UpdateStr("abc", []byte("1"))
			tr.UpdateStr("abcd", []byte("2"))
			tr.UpdateStr("abcde", []byte("2"))
			tr.DeleteStr("abcde")
			tr.DeleteStr("abcd")
			tr.DeleteStr("abc")
			tr.Commit()
			c1 := trie256p.RootCommitment(tr)

			store = trie_go.NewInMemoryKVStore()
			tr = trie256p.New(m, store)

			tr.UpdateStr("abc", []byte("1"))
			tr.UpdateStr("abcd", []byte("2"))
			tr.UpdateStr("abcde", []byte("2"))
			tr.DeleteStr("abcde")
			tr.Commit()
			tr.DeleteStr("abcd")
			tr.DeleteStr("abc")
			tr.Commit()
			c2 := trie256p.RootCommitment(tr)

			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("deletion edge cases 3", func(t *testing.T) {
			store := trie_go.NewInMemoryKVStore()
			tr := trie256p.New(m, store)

			tr.UpdateStr("abcd", []byte("1"))
			tr.UpdateStr("ab1234", []byte("2"))
			tr.DeleteStr("ab1234")
			tr.Commit()
			c1 := trie256p.RootCommitment(tr)

			store = trie_go.NewInMemoryKVStore()
			tr = trie256p.New(m, store)

			tr.UpdateStr("abcd", []byte("1"))
			tr.UpdateStr("ab1234", []byte("2"))
			tr.Commit()
			tr.DeleteStr("ab1234")
			tr.Commit()
			c2 := trie256p.RootCommitment(tr)

			require.True(t, trie_go.EqualCommitments(c1, c2))
		})

	}
	runTest(t, trie_blake2b.New())
	runTest(t, trie_kzg_bn256.New())
}

func genRnd3() []string {
	ret := make([]string, 0, len(letters)*len(letters)*len(letters))
	for i := range letters {
		for j := range letters {
			for k := range letters {
				s := string([]byte{letters[i], letters[j], letters[k]})
				s = s + s + s + s
				r1 := rand.Intn(len(s))
				r2 := rand.Intn(len(s))
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
	for i := range letters {
		for j := range letters {
			for k := range letters {
				for l := range letters {
					s := string([]byte{letters[i], letters[j], letters[k], letters[l]})
					s = s + s + s + s
					r1 := rand.Intn(len(s))
					r2 := rand.Intn(len(s))
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
	for i := 0; i < num; i++ {
		ret = append(ret, data[rand.Intn(len(data))])
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
	runTest := func(t *testing.T, m trie256p.CommitmentModel, shortData bool) {
		t.Run("determinism1", func(t *testing.T) {
			data := genData1()
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			permutation := rand.Perm(len(data))
			for _, i := range permutation {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("determinism2", func(t *testing.T) {
			data := genData2()
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			permutation := rand.Perm(len(data))
			for _, i := range permutation {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("determinism3", func(t *testing.T) {
			data := genRnd3()
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			permutation := rand.Perm(len(data))
			for _, i := range permutation {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("determinism4", func(t *testing.T) {
			data := genRnd4()
			if shortData {
				data = data[:1000]
			}
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			permutation := rand.Perm(len(data))
			for _, i := range permutation {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie_go.EqualCommitments(c1, c2))

			tr2.PersistMutations(store2)
			trieSize := trie_go.ByteSize(store2)
			numEntries := trie_go.NumEntries(store2)
			t.Logf("key entries = %d", len(data))
			t.Logf("Trie entries = %d", numEntries)
			t.Logf("Trie bytes = %d KB", trieSize/1024)
			t.Logf("Trie bytes/entry = %d ", trieSize/numEntries)
		})
		t.Run("determinism5", func(t *testing.T) {
			data := genRnd4()
			if shortData {
				data = data[:1000]
			}
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
				tr1.Commit()
			}
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)
			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}

			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
	}
	runTest(t, trie_blake2b.New(), false)
	runTest(t, trie_kzg_bn256.New(), true)
}

func TestTrieRndKeyCommitment(t *testing.T) {
	runTest := func(t *testing.T, m trie256p.CommitmentModel, shortData bool) {
		t.Run("determ key commitment1", func(t *testing.T) {
			data := genData1()
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for _, d := range data {
				if len(d) > 0 {
					tr1.MustInsertKeyCommitment([]byte(d))
				}
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)
			permutation := rand.Perm(len(data))
			for _, i := range permutation {
				if len(data[i]) > 0 {
					tr2.MustInsertKeyCommitment([]byte(data[i]))
				}
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("determ key commitment2", func(t *testing.T) {
			data := genData2()
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				if len(data[i]) > 0 {
					tr1.MustInsertKeyCommitment([]byte(data[i]))
				}
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			permutation := rand.Perm(len(data))
			for _, i := range permutation {
				if len(data[i]) > 0 {
					tr2.MustInsertKeyCommitment([]byte(data[i]))
				}
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("determ key commitment3", func(t *testing.T) {
			data := genRnd3()
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				if len(data[i]) > 0 {
					tr1.MustInsertKeyCommitment([]byte(data[i]))
				}
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			permutation := rand.Perm(len(data))
			for _, i := range permutation {
				if len(data[i]) > 0 {
					tr2.MustInsertKeyCommitment([]byte(data[i]))
				}
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("determ key commitment4", func(t *testing.T) {
			data := genRnd4()
			if shortData {
				data = data[:1000]
			}
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				if len(data[i]) > 0 {
					tr1.MustInsertKeyCommitment([]byte(data[i]))
				}
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)

			permutation := rand.Perm(len(data))
			for _, i := range permutation {
				if len(data[i]) > 0 {
					tr2.MustInsertKeyCommitment([]byte(data[i]))
				}
			}
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie_go.EqualCommitments(c1, c2))

			tr2.PersistMutations(store2)
			trieSize := trie_go.ByteSize(store2)
			numEntries := trie_go.NumEntries(store2)
			t.Logf("key entries = %d", len(data))
			t.Logf("Trie entries = %d", numEntries)
			t.Logf("Trie bytes = %d KB", trieSize/1024)
			t.Logf("Trie bytes/entry = %d ", trieSize/numEntries)
		})
		t.Run("determ key commitment5", func(t *testing.T) {
			data := genRnd4()
			if shortData {
				data = data[:1000]
			}
			store1 := trie_go.NewInMemoryKVStore()
			tr1 := trie256p.New(m, store1)

			for i := range data {
				if len(data[i]) > 0 {
					tr1.MustInsertKeyCommitment([]byte(data[i]))
				}
				tr1.Commit()
			}
			c1 := trie256p.RootCommitment(tr1)

			store2 := trie_go.NewInMemoryKVStore()
			tr2 := trie256p.New(m, store2)
			for i := range data {
				if len(data[i]) > 0 {
					tr2.MustInsertKeyCommitment([]byte(data[i]))
				}
			}

			tr2.Commit()
			c2 := trie256p.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
	}
	runTest(t, trie_blake2b.New(), false)
	runTest(t, trie_kzg_bn256.New(), true)
}

func TestKeyCommitmentOptimization(t *testing.T) {
	data := genRnd4()[:10_000]
	runTest := func(model trie256p.CommitmentModel) {
		store1 := trie_go.NewInMemoryKVStore()
		store2 := trie_go.NewInMemoryKVStore()
		tr1 := trie256p.New(model, store1)
		tr2 := trie256p.New(model, store2)

		for _, d := range data {
			if len(d) > 0 {
				tr1.MustInsertKeyCommitment([]byte(d))
			}
		}
		tr1.Commit()
		tr1.PersistMutations(store1)

		for _, d := range data {
			b := []byte(d)
			if len(d) > 0 {
				b[0] = b[0] + 1 // make different
				tr2.Update([]byte(d), b)
			}
		}
		tr2.Commit()
		tr2.PersistMutations(store2)

		size1 := trie_go.ByteSize(store1)
		size2 := trie_go.ByteSize(store2)
		numEntries := trie_go.NumEntries(store1)
		require.EqualValues(t, numEntries, trie_go.NumEntries(store2))

		t.Logf("num entries: %d", numEntries)
		t.Logf("   with key commitments. Byte size: %d, avg: %f bytes per entry", size1, float32(size1)/float32(numEntries))
		t.Logf("without key commitments. Byte size: %d, avg: %f bytes per entry", size2, float32(size2)/float32(numEntries))
	}
	runTest(trie_blake2b.New())
	runTest(trie_kzg_bn256.New())
}

func TestTrieWithDeletion(t *testing.T) {
	data := []string{"0", "1", "2", "3", "4", "5"}
	var tr1, tr2 *trie256p.Trie
	runTest := func(t *testing.T, m trie256p.CommitmentModel) {
		initTest := func() {
			store1 := trie_go.NewInMemoryKVStore()
			tr1 = trie256p.New(m, store1)
			store2 := trie_go.NewInMemoryKVStore()
			tr2 = trie256p.New(m, store2)
		}
		t.Run("del1", func(t *testing.T) {
			initTest()
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Delete([]byte(data[1]))
			tr2.Update([]byte(data[1]), []byte(data[1]))
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr1)

			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("del2", func(t *testing.T) {
			initTest()
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			tr2.Commit()
			tr2.Delete([]byte(data[1]))
			tr2.Update([]byte(data[1]), []byte(data[1]))
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr1)

			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("del3", func(t *testing.T) {
			initTest()
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
				tr2.Commit()
			}
			tr2.Delete([]byte(data[1]))
			tr2.Update([]byte(data[1]), []byte(data[1]))
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr1)

			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("del4", func(t *testing.T) {
			initTest()
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
				tr2.Commit()
			}
			tr2.Delete([]byte(data[1]))
			tr2.Commit()
			tr2.Update([]byte(data[1]), []byte(data[1]))
			tr2.Commit()
			c2 := trie256p.RootCommitment(tr1)

			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("del5", func(t *testing.T) {
			initTest()
			for i := range data {
				tr1.Update([]byte(data[i]), []byte(data[i]))
			}
			tr1.Commit()
			c1 := trie256p.RootCommitment(tr1)

			for i := range data {
				tr2.Update([]byte(data[i]), []byte(data[i]))
				tr2.Commit()
			}
			c2 := trie256p.RootCommitment(tr1)
			require.True(t, trie_go.EqualCommitments(c1, c2))

			tr2.Delete([]byte(data[1]))
			tr2.Commit()

			c2 = trie256p.RootCommitment(tr2)
			require.False(t, trie_go.EqualCommitments(c1, c2))

			tr2.Update([]byte(data[1]), []byte(data[1]))
			tr2.Commit()
			c2 = trie256p.RootCommitment(tr1)

			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("del all", func(t *testing.T) {
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

			c := trie256p.RootCommitment(tr1)
			require.EqualValues(t, nil, c)
		})
	}
	runTest(t, trie_blake2b.New())
	runTest(t, trie_kzg_bn256.New())
}

func TestTrieWithDeletionDeterm(t *testing.T) {
	data := []string{"0", "1", "2", "3", "4", "5"}
	var tr1, tr2 *trie256p.Trie
	runTest := func(t *testing.T, m trie256p.CommitmentModel, shortData bool) {
		initTest := func() {
			store1 := trie_go.NewInMemoryKVStore()
			tr1 = trie256p.New(m, store1)
			store2 := trie_go.NewInMemoryKVStore()
			tr2 = trie256p.New(m, store2)
		}
		t.Run("del determ 1", func(t *testing.T) {
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
			c1 := trie256p.RootCommitment(tr1)

			permutation := rand.Perm(len(data))
			for _, i := range permutation {
				tr2.Update([]byte(data[i]), []byte(data[i]))
			}
			for i := range dels {
				tr2.Delete([]byte(dels[i]))
			}
			tr2.Commit()

			c2 := trie256p.RootCommitment(tr2)
			t.Logf("root1 = %s", c1)
			t.Logf("root2 = %s", c2)
			require.True(t, trie_go.EqualCommitments(c1, c2))
		})
		t.Run("del determ 2", func(t *testing.T) {
			initTest()
			data = genRnd4()
			if shortData {
				data = data[:1000]
			}
			t.Logf("data len = %d", len(data))

			const rounds = 5
			var c, cPrev trie_go.VCommitment

			rng := rand.New(rand.NewSource(time.Now().UnixNano()))
			for i := 0; i < rounds; i++ {
				t.Logf("-------- round #%d", i)
				perm := rng.Perm(len(data))
				for _, j := range perm {
					tr1.UpdateStr(data[j], []byte(data[j]))
				}
				tr1.Commit()
				if cPrev != nil {
					require.True(t, trie_go.EqualCommitments(c, cPrev))
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
				c = trie256p.RootCommitment(tr1)
				require.True(t, trie_go.EqualCommitments(c, nil))
			}
		})
	}
	runTest(t, trie_blake2b.New(), false)
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
	runTest := func(t *testing.T, m trie256p.CommitmentModel) {
		var c [2]trie_go.VCommitment
		for round := range []int{0, 1} {
			t.Logf("------- run %d", round)
			store := trie_go.NewInMemoryKVStore()
			tr := trie256p.New(m, store)
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
			c[round] = trie256p.RootCommitment(tr)
			t.Logf("c[%d] = %s", round, c[round])
			diff := tr.Reconcile(store)
			require.EqualValues(t, 0, len(diff))
		}
		require.True(t, trie_go.EqualCommitments(c[0], c[1]))
	}
	runTest(t, trie_blake2b.New())
	runTest(t, trie_kzg_bn256.New())
}

func TestGenTrie(t *testing.T) {
	const filename = "$$for testing$$"
	runTest := func(t *testing.T, m trie256p.CommitmentModel) {
		kind := "blake2b"
		if _, ok := m.(*trie_kzg_bn256.CommitmentModel); ok {
			kind = "kzg"
		}
		fname := filename + "_" + kind

		t.Run("gen file "+kind, func(t *testing.T) {
			store := trie_go.NewInMemoryKVStore()
			data := genRnd4()
			for _, s := range data {
				store.Set([]byte(s), []byte("abcdefghijklmnoprstquwxyz"))
			}
			t.Logf("num records = %d", len(data))
			n, err := trie_go.DumpToFile(store, fname+".bin")
			require.NoError(t, err)
			t.Logf("wrote %d bytes to '%s'", n, fname+".bin")
		})
		t.Run("gen trie "+kind, func(t *testing.T) {
			store := trie_go.NewInMemoryKVStore()
			n, err := trie_go.UnDumpFromFile(store, fname+".bin")
			require.NoError(t, err)
			t.Logf("read %d bytes to '%s'", n, fname+".bin")

			storeTrie := trie_go.NewInMemoryKVStore()
			tr := trie256p.New(m, storeTrie)
			tr.UpdateAll(store)
			tr.Commit()
			tr.PersistMutations(store)
			t.Logf("trie len = %d", trie_go.NumEntries(store))
			n, err = trie_go.DumpToFile(store, fname+".trie")
			require.NoError(t, err)
			t.Logf("dumped trie size = %d", n)
		})
	}
	runTest(t, trie_blake2b.New())
	runTest(t, trie_kzg_bn256.New())
}
