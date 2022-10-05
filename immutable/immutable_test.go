package immutable

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/iotaledger/trie.go/common"
	"github.com/iotaledger/trie.go/models/trie_blake2b"
	"github.com/iotaledger/trie.go/models/trie_kzg_bn256"
	"github.com/stretchr/testify/require"
)

func TestDeletedKey(t *testing.T) {
	store := common.NewInMemoryKVStore()
	m := trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160)

	var root0 common.VCommitment
	{
		root0 = MustInitRoot(store, m, []byte("identity"))
	}

	var root1 common.VCommitment
	{
		tr, err := NewTrieUpdatable(m, store, root0)
		require.NoError(t, err)
		tr.Update([]byte("a"), []byte("a"))
		tr.Update([]byte("b"), []byte("b"))
		root1 = tr.Commit(store)
	}

	var root2 common.VCommitment
	{
		tr, err := NewTrieUpdatable(m, store, root1)
		require.NoError(t, err)
		tr.Update([]byte("a"), nil)
		tr.Update([]byte("b"), []byte("bb"))
		tr.Update([]byte("c"), []byte("c"))
		root2 = tr.Commit(store)
	}

	tr, err := NewTrieUpdatable(m, store, root2)
	require.NoError(t, err)

	state := tr.TrieReader
	require.Nil(t, state.Get([]byte("a")))
}

func TestCreateTrie(t *testing.T) {
	runTest := func(m common.CommitmentModel) {
		t.Run("not init-"+m.ShortName(), func(t *testing.T) {
			_, err := NewTrieUpdatable(m, common.NewInMemoryKVStore(), nil)
			common.RequireErrorWith(t, err, "does not exist")
		})
		t.Run("wrong init-"+m.ShortName(), func(t *testing.T) {
			store := common.NewInMemoryKVStore()
			common.RequirePanicOrErrorWith(t, func() error {
				MustInitRoot(store, m, nil)
				return nil
			}, "identity of the root cannot be empty")
		})
		t.Run("ok init-"+m.ShortName(), func(t *testing.T) {
			store := common.NewInMemoryKVStore()
			const identity1 = "abc"
			const identity2 = "abcabc"

			rootC1 := MustInitRoot(store, m, []byte(identity1))
			require.NotNil(t, rootC1)
			t.Logf("initial root commitment with id '%s': %s", identity1, rootC1)

			rootC2 := MustInitRoot(store, m, []byte(identity2))
			require.NotNil(t, rootC2)
			t.Logf("initial root commitment with id '%s': %s", identity2, rootC2)

			require.False(t, m.EqualCommitments(rootC1, rootC2))
		})
		t.Run("ok init long id-"+m.ShortName(), func(t *testing.T) {
			store := common.NewInMemoryKVStore()
			identity := strings.Repeat("abc", 50)

			rootC1 := MustInitRoot(store, m, []byte(identity))
			require.NotNil(t, rootC1)
			t.Logf("initial root commitment with id '%s': %s", identity, rootC1)
		})
		t.Run("update 1"+m.ShortName(), func(t *testing.T) {
			store := common.NewInMemoryKVStore()
			const (
				identity = "idIDidIDidID"
				key      = "key"
				value    = "value"
			)

			rootInitial := MustInitRoot(store, m, []byte(identity))
			require.NotNil(t, rootInitial)
			t.Logf("initial root commitment with id '%s': %s", identity, rootInitial)

			tr, err := NewTrieUpdatable(m, store, rootInitial)
			require.NoError(t, err)

			v := tr.GetStr("")
			require.EqualValues(t, identity, v)

			tr.UpdateStr(key, value)
			rootCnext := tr.Commit(store)
			t.Logf("initial root commitment: %s", rootInitial)
			t.Logf("next root commitment: %s", rootCnext)

			v = tr.GetStr("")
			require.EqualValues(t, identity, v)

			require.False(t, tr.HasStr(key))

			err = tr.SetRoot(rootCnext)
			require.NoError(t, err)

			v = tr.GetStr("")
			require.EqualValues(t, identity, v)

			v = tr.GetStr(key)
			require.EqualValues(t, value, v)

			require.True(t, tr.HasStr(key))
		})
		t.Run("update 2 long value"+m.ShortName(), func(t *testing.T) {
			store := common.NewInMemoryKVStore()
			const (
				identity = "idIDidIDidID"
				key      = "key"
				value    = "value"
			)

			rootInitial := MustInitRoot(store, m, []byte(identity))
			require.NotNil(t, rootInitial)
			t.Logf("initial root commitment with id '%s': %s", identity, rootInitial)

			tr, err := NewTrieUpdatable(m, store, rootInitial)
			require.NoError(t, err)

			v := tr.GetStr("")
			require.EqualValues(t, identity, v)

			tr.UpdateStr(key, strings.Repeat(value, 500))
			rootCnext := tr.Commit(store)
			t.Logf("initial root commitment: %s", rootInitial)
			t.Logf("next root commitment: %s", rootCnext)

			v = tr.GetStr("")
			require.EqualValues(t, identity, v)

			require.False(t, tr.HasStr(key))

			err = tr.SetRoot(rootCnext)
			require.NoError(t, err)
			require.True(t, m.EqualCommitments(rootCnext, tr.Root()))

			v = tr.GetStr("")
			require.EqualValues(t, identity, v)

			v = tr.GetStr(key)
			require.EqualValues(t, strings.Repeat(value, 500), v)

			require.True(t, tr.HasStr(key))
		})
	}
	runTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256))
	runTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160))
	runTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256))
	runTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160))
	runTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256))
	runTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160))
	runTest(trie_kzg_bn256.New())
}

func TestBaseUpdate(t *testing.T) {
	const identity = "idIDidIDidID"

	runTest := func(m common.CommitmentModel, data []string) {
		t.Run("update many", func(t *testing.T) {
			store := common.NewInMemoryKVStore()
			rootInitial := MustInitRoot(store, m, []byte(identity))
			require.NotNil(t, rootInitial)
			t.Logf("initial root commitment with id '%s': %s", identity, rootInitial)

			tr, err := NewTrieUpdatable(m, store, rootInitial)
			require.NoError(t, err)

			//data = data[:2]
			for _, key := range data {
				value := strings.Repeat(key, 5)
				fmt.Printf("+++ update key='%s', value='%s'\n", key, value)
				tr.UpdateStr(key, value)
			}
			rootNext := tr.Commit(store)
			t.Logf("after commit: %s", rootNext)

			err = tr.SetRoot(rootNext)
			require.NoError(t, err)

			for _, key := range data {
				v := tr.GetStr(key)
				require.EqualValues(t, strings.Repeat(key, 5), v)
			}
		})
	}
	data := []string{"ab", "acd", "a", "dba", "abc", "abd", "abcdafgh", "aaaaaaaaaaaaaaaa", "klmnt"}

	runTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"a", "ab"})
	runTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"ab", "acb"})
	runTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"abc", "a"})
	runTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), data)
	runTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), data)
	runTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), data)
	runTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), data)
	runTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), data)
	runTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), data)
	runTest(trie_kzg_bn256.New(), data)
}

func runUpdateScenario(trie *Trie, store common.KVWriter, scenario []string) (map[string]string, common.VCommitment) {
	checklist := make(map[string]string)
	uncommitted := false
	var ret common.VCommitment
	for _, key := range scenario {
		if len(key) == 0 {
			continue
		}
		switch key[0] {
		case '-':
			trie.DeleteStr(key[1:])
			//fmt.Printf("+++ delete key: '%s'\n", key[1:])
			delete(checklist, key[1:])
			uncommitted = true
		case '*':
			ret = trie.Commit(store)
			_ = trie.SetRoot(ret)
			//fmt.Printf("+++ commit. Root: '%s'\n", ret)
			uncommitted = false
		default:
			value := strings.Repeat(key, 5)
			//fmt.Printf("+++ update key: '%s', value: '%s'\n", key, value)
			trie.UpdateStr(key, value)
			checklist[key] = value
			uncommitted = true
		}
	}
	if uncommitted {
		ret = trie.Commit(store)
		fmt.Printf("+++ commit. Root: '%s'\n", ret)
		_ = trie.SetRoot(ret)
	}
	fmt.Printf("+++ return root: '%s'\n", ret)
	return checklist, ret
}

func checkUpdateScenario(t *testing.T, trie *Trie, scenario []string, checklist map[string]string) {
	var k string
	for _, key := range scenario {
		if len(key) == 0 {
			continue
		}
		switch key[0] {
		case '*':
			continue
		case '-':
			k = key[1:]
		default:
			k = key
		}
		v := trie.GetStr(k)
		_, check := checklist[key]
		if check {
			require.EqualValues(t, strings.Repeat(key, 5), v)
		} else {
			require.EqualValues(t, "", v)
		}
	}
}

func TestBaseScenarios(t *testing.T) {
	const identity = "idIDidIDidID"

	tf := func(m common.CommitmentModel, data []string) func(t *testing.T) {
		return func(t *testing.T) {
			store := common.NewInMemoryKVStore()
			rootInitial := MustInitRoot(store, m, []byte(identity))
			require.NotNil(t, rootInitial)
			t.Logf("initial root commitment with id '%s': %s", identity, rootInitial)

			tr, err := NewTrieUpdatable(m, store, rootInitial)
			require.NoError(t, err)

			checklist, _ := runUpdateScenario(tr, store, data)
			checkUpdateScenario(t, tr, data, checklist)
		}
	}
	data1 := []string{"ab", "acd", "-a", "-ab", "abc", "abd", "abcdafgh", "-acd", "aaaaaaaaaaaaaaaa", "klmnt"}

	t.Run("1", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"a", "-a"}))
	t.Run("2", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"-acb"}))
	t.Run("3", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"abc", "a", "-abc", "-a"}))
	t.Run("4", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"abc", "a", "-a", "-abc", "klmn"}))
	t.Run("5", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), data1))
	t.Run("6", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), data1))
	t.Run("7", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), data1))
	t.Run("8", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), data1))
	t.Run("9", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), data1))
	t.Run("10", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), data1))
	t.Run("11", tf(trie_kzg_bn256.New(), data1))

	t.Run("12", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"a", "ab", "-a"}))

	data2 := []string{"a", "ab", "abc", "abcd", "abcde", "-abd", "-a"}
	t.Run("14", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), data2))
	t.Run("15", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), data2))
	t.Run("16", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), data2))
	t.Run("17", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), data2))
	t.Run("18", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), data2))
	t.Run("19", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), data2))
	t.Run("20", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), data2))
	t.Run("21", tf(trie_kzg_bn256.New(), data2))

	data3 := []string{"a", "ab", "abc", "abcd", "abcde", "-abcde", "-abcd", "-abc", "-ab", "-a"}
	t.Run("14", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), data3))
	t.Run("15", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), data3))
	t.Run("16", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), data3))
	t.Run("17", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), data3))
	t.Run("18", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), data3))
	t.Run("19", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), data3))
	t.Run("20", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), data3))
	t.Run("21", tf(trie_kzg_bn256.New(), data3))

	data4 := genRnd3()
	name := "update-many-"
	t.Run(name+"1", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), data4))
	t.Run(name+"2", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), data4))
	t.Run(name+"3", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), data4))
	t.Run(name+"4", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), data4))
	t.Run(name+"5", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), data4))
	t.Run(name+"6", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), data4))
	t.Run(name+"7", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), data4))
	t.Run(name+"8", tf(trie_kzg_bn256.New(), data3))

}

func TestDeterminism(t *testing.T) {
	const identity = "idIDidIDidID"

	tf := func(m common.CommitmentModel, scenario1, scenario2 []string) func(t *testing.T) {
		return func(t *testing.T) {
			fmt.Printf("--------- scenario1: %v\n", scenario1)
			store1 := common.NewInMemoryKVStore()
			initRoot1 := MustInitRoot(store1, m, []byte(identity))

			tr1, err := NewTrieUpdatable(m, store1, initRoot1)
			require.NoError(t, err)

			checklist1, root1 := runUpdateScenario(tr1, store1, scenario1)
			checkUpdateScenario(t, tr1, scenario1, checklist1)

			fmt.Printf("--------- scenario2: %v\n", scenario2)
			store2 := common.NewInMemoryKVStore()
			initRoot2 := MustInitRoot(store2, m, []byte(identity))

			tr2, err := NewTrieUpdatable(m, store2, initRoot2)
			require.NoError(t, err)

			checklist2, root2 := runUpdateScenario(tr2, store2, scenario2)
			checkUpdateScenario(t, tr2, scenario2, checklist2)

			require.True(t, m.EqualCommitments(root1, root2))
		}
	}

	s1 := []string{"a", "ab"}
	s2 := []string{"ab", "a"}
	name := "order-simple-"
	t.Run(name+"1", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), s1, s2))
	t.Run(name+"2", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), s1, s2))
	t.Run(name+"3", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), s1, s2))
	t.Run(name+"4", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), s1, s2))
	t.Run(name+"5", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), s1, s2))
	t.Run(name+"6", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), s1, s2))
	t.Run(name+"7", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), s1, s2))
	t.Run(name+"8", tf(trie_kzg_bn256.New(), s1, s2))

	s1 = genRnd3()
	s2 = reverse(s1)
	name = "order-reverse-many-"
	t.Run(name+"1", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), s1, s2))
	t.Run(name+"2", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), s1, s2))
	t.Run(name+"3", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), s1, s2))
	t.Run(name+"4", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), s1, s2))
	t.Run(name+"5", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), s1, s2))
	t.Run(name+"6", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), s1, s2))
	t.Run(name+"7", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), s1, s2))
	t.Run(name+"8", tf(trie_kzg_bn256.New(), s1, s2))

	s1 = []string{"a", "ab"}
	s2 = []string{"a", "*", "ab"}
	name = "commit-simple-"
	t.Run(name+"1", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), s1, s2))
	t.Run(name+"2", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), s1, s2))
	t.Run(name+"3", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), s1, s2))
	t.Run(name+"4", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), s1, s2))
	t.Run(name+"5", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), s1, s2))
	t.Run(name+"6", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), s1, s2))
	t.Run(name+"7", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), s1, s2))

	t.Run(name+"8", tf(trie_kzg_bn256.New(), s1, s2)) // failing because of KZG commitment model cryptography bug

}

const letters = "abcdefghijklmnop"

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

func reverse(orig []string) []string {
	ret := make([]string, 0, len(orig))
	for i := len(orig) - 1; i >= 0; i-- {
		ret = append(ret, orig[i])
	}
	return ret
}
