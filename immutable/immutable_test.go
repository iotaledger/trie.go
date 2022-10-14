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
		root2 = tr.Commit(store, true) // <<<< override default to set the new root and clear cache

		err = tr.SetRoot(root2) // <<<<<<<<<< we need this if commit does not set the new root
		require.NoError(t, err)

		require.Nil(t, tr.Get([]byte("a")))
	}

	state := NewTrieReader(m, store, root2) // <<<<< We can create TrieReader independently of TrieUpdatable
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
			rootCnext := tr.Commit(store, true)
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
			rootCnext := tr.Commit(store, true)
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

var traceScenarios = false

func runUpdateScenario(trie *Trie, store common.KVWriter, scenario []string) (map[string]string, common.VCommitment) {
	checklist := make(map[string]string)
	uncommitted := false
	var ret common.VCommitment
	for _, cmd := range scenario {
		if len(cmd) == 0 {
			continue
		}
		if cmd == "*" {
			ret = trie.Commit(store)
			_ = trie.SetRoot(ret)
			if traceScenarios {
				fmt.Printf("+++ commit. Root: '%s'\n", ret)
			}
			uncommitted = false
			continue
		}
		var key, value []byte
		before, after, found := strings.Cut(cmd, "/")
		if found {
			if len(before) == 0 {
				continue // key must not be empty
			}
			key = []byte(before)
			if len(after) > 0 {
				value = []byte(after)
			}
		} else {
			key = []byte(cmd)
			value = []byte(cmd)
		}
		trie.Update(key, value)
		checklist[string(key)] = string(value)
		uncommitted = true
		if traceScenarios {
			if len(value) > 0 {
				fmt.Printf("SET '%s' -> '%s'\n", string(key), string(value))
			} else {
				fmt.Printf("DEL '%s'\n", string(key))
			}
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

func checkResult(t *testing.T, trie *Trie, checklist map[string]string) {
	for key, expectedValue := range checklist {
		v := trie.GetStr(key)
		if traceScenarios {
			if len(v) > 0 {
				fmt.Printf("FOUND '%s': '%s' (expected '%s')\n", key, v, expectedValue)
			} else {
				fmt.Printf("NOT FOUND '%s' (expected '%s')\n", key, func() string {
					if len(expectedValue) > 0 {
						return "FOUND"
					} else {
						return "NOT FOUND"
					}
				}())
			}
		}
		require.EqualValues(t, expectedValue, v)
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
			checkResult(t, tr, checklist)
		}
	}
	data1 := []string{"ab", "acd", "-a", "-ab", "abc", "abd", "abcdafgh", "-acd", "aaaaaaaaaaaaaaaa", "klmnt"}

	t.Run("1-1", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"a", "a/"}))
	t.Run("1-2", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"a", "*", "a/"}))
	t.Run("1-3", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"a", "b", "*", "b/", "a/"}))
	t.Run("1-4", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"a", "b", "*", "a/", "b/"}))
	t.Run("1-5", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"a", "b", "*", "a/", "b/bb", "c"}))
	t.Run("1-6", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"a", "b", "*", "a/", "b/bb", "c"}))
	t.Run("1-7", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"a", "b", "*", "a/", "b", "c"}))
	t.Run("1-8", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"acb/", "*", "acb/bca", "acb/123"}))
	t.Run("1-9", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"abc", "a", "abc/", "a/"}))
	t.Run("1-10", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), []string{"abc", "a", "a/", "abc/", "klmn"}))

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
	t.Run(name+"2", tf(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), data4))
	t.Run(name+"3", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), data4))
	t.Run(name+"4", tf(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), data4))
	t.Run(name+"5", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), data4))
	t.Run(name+"6", tf(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), data4))
	t.Run(name+"7", tf(trie_kzg_bn256.New(), data3))

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
			checkResult(t, tr1, checklist1)

			fmt.Printf("--------- scenario2: %v\n", scenario2)
			store2 := common.NewInMemoryKVStore()
			initRoot2 := MustInitRoot(store2, m, []byte(identity))

			tr2, err := NewTrieUpdatable(m, store2, initRoot2)
			require.NoError(t, err)

			checklist2, root2 := runUpdateScenario(tr2, store2, scenario2)
			checkResult(t, tr2, checklist2)

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

	t.Run(name+"kzg", tf(trie_kzg_bn256.New(), s1, s2)) // failing because of KZG commitment model cryptography bug

}

func TestIterate(t *testing.T) {
	iterTest := func(m common.CommitmentModel, scenario []string) func(t *testing.T) {
		return func(t *testing.T) {
			store := common.NewInMemoryKVStore()
			rootInitial := MustInitRoot(store, m, []byte("identity"))
			require.NotNil(t, rootInitial)
			t.Logf("initial root commitment with id '%s': %s", "identity", rootInitial)

			tr, err := NewTrieUpdatable(m, store, rootInitial)
			require.NoError(t, err)

			checklist, root := runUpdateScenario(tr, store, scenario)
			checkResult(t, tr, checklist)

			trr := NewTrieReader(m, store, root, 0)
			trr.Iterate(func(k []byte, v []byte) bool {
				if traceScenarios {
					fmt.Printf("---- iter --- '%s': '%s'\n", string(k), string(v))
				}
				if len(k) != 0 {
					vCheck := checklist[string(k)]
					require.True(t, len(v) > 0)
					require.EqualValues(t, []byte(vCheck), v)
				} else {
					require.EqualValues(t, []byte("identity"), v)
				}
				return true
			})
		}
	}
	{
		name := "iterate-one-"
		scenario := []string{"a"}
		t.Run(name+"1", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), scenario))
		t.Run(name+"2", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), scenario))
		t.Run(name+"3", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), scenario))
		t.Run(name+"4", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), scenario))
		t.Run(name+"5", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), scenario))
		t.Run(name+"6", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), scenario))
		t.Run(name+"7", iterTest(trie_kzg_bn256.New(), scenario))
	}
	{
		name := "iterate-"
		scenario := []string{"a", "b", "c", "*", "a/"}
		t.Run(name+"1", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), scenario))
		t.Run(name+"2", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), scenario))
		t.Run(name+"3", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), scenario))
		t.Run(name+"4", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), scenario))
		t.Run(name+"5", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), scenario))
		t.Run(name+"6", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), scenario))
		t.Run(name+"7", iterTest(trie_kzg_bn256.New(), scenario))
	}
	{
		name := "iterate-big-"
		scenario := genRnd3()
		t.Run(name+"1", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), scenario))
		t.Run(name+"2", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), scenario))
		t.Run(name+"3", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), scenario))
		t.Run(name+"4", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), scenario))
		t.Run(name+"5", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), scenario))
		t.Run(name+"6", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), scenario))
		t.Run(name+"7", iterTest(trie_kzg_bn256.New(), scenario))
	}
}

func TestIteratePrefix(t *testing.T) {
	iterTest := func(m common.CommitmentModel, scenario []string, prefix string) func(t *testing.T) {
		return func(t *testing.T) {
			store := common.NewInMemoryKVStore()
			rootInitial := MustInitRoot(store, m, []byte("identity"))
			require.NotNil(t, rootInitial)
			t.Logf("initial root commitment with id '%s': %s", "identity", rootInitial)

			tr, err := NewTrieUpdatable(m, store, rootInitial)
			require.NoError(t, err)

			_, root := runUpdateScenario(tr, store, scenario)

			trr := NewTrieReader(m, store, root, 0)

			countIter := 0
			trr.Iterator([]byte(prefix)).Iterate(func(k []byte, v []byte) bool {
				if traceScenarios {
					fmt.Printf("---- iter --- '%s': '%s'\n", string(k), string(v))
				}
				if string(v) != "identity" {
					countIter++
				}
				require.True(t, strings.HasPrefix(string(k), prefix))
				return true
			})
			countOrig := 0
			for _, s := range scenario {
				if strings.HasPrefix(s, prefix) {
					countOrig++
				}
			}
			require.EqualValues(t, countOrig, countIter)
		}
	}
	{
		name := "iterate-ab"
		scenario := []string{"a", "ab", "c", "cd", "abcd", "klmn", "aaa", "abra", "111"}
		prefix := "ab"
		t.Run(name+"1", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"2", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"3", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"4", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"5", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"6", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"7", iterTest(trie_kzg_bn256.New(), scenario, prefix))
	}
	{
		name := "iterate-a"
		scenario := []string{"a", "ab", "c", "cd", "abcd", "klmn", "aaa", "abra", "111", "baba", "ababa"}
		prefix := "a"
		t.Run(name+"1", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"2", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"3", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"4", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"5", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"6", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"7", iterTest(trie_kzg_bn256.New(), scenario, prefix))
	}
	{
		name := "iterate-empty"
		scenario := []string{"a", "ab", "c", "cd", "abcd", "klmn", "aaa", "abra", "111", "baba", "ababa"}
		prefix := ""
		t.Run(name+"1", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"2", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"3", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"4", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"5", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"6", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"7", iterTest(trie_kzg_bn256.New(), scenario, prefix))
	}
	{
		name := "iterate-none"
		scenario := []string{"a", "ab", "c", "cd", "abcd", "klmn", "aaa", "abra", "111", "baba", "ababa"}
		prefix := "---"
		t.Run(name+"1", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"2", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"3", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"4", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"5", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"6", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"7", iterTest(trie_kzg_bn256.New(), scenario, prefix))
	}
}

func TestDeletePrefix(t *testing.T) {
	iterTest := func(m common.CommitmentModel, scenario []string, prefix string) func(t *testing.T) {
		return func(t *testing.T) {
			store := common.NewInMemoryKVStore()
			rootInitial := MustInitRoot(store, m, []byte("identity"))
			require.NotNil(t, rootInitial)
			t.Logf("initial root commitment with id '%s': %s", "identity", rootInitial)

			tr, err := NewTrieUpdatable(m, store, rootInitial)
			require.NoError(t, err)

			_, root := runUpdateScenario(tr, store, scenario)

			tr, err = NewTrieUpdatable(m, store, root, 0)
			require.NoError(t, err)

			deleted := tr.DeletePrefix([]byte(prefix))
			tr.Commit(store)

			tr.Iterator([]byte(prefix)).Iterate(func(k []byte, v []byte) bool {
				if traceScenarios {
					fmt.Printf("---- iter --- '%s': '%s'\n", string(k), string(v))
				}
				if len(k) == 0 {
					require.EqualValues(t, "identity", string(v))
					return true
				}
				if deleted && len(prefix) != 0 {
					require.False(t, strings.HasPrefix(string(k), prefix))
				}
				return true
			})
		}
	}
	{
		name := "delete-ab"
		scenario := []string{"a", "ab", "c", "cd", "abcd", "klmn", "aaa", "abra", "111"}
		prefix := "ab"
		t.Run(name+"1", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"2", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"3", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"4", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"5", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"6", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"7", iterTest(trie_kzg_bn256.New(), scenario, prefix))
	}
	{
		name := "delete-a"
		scenario := []string{"a", "ab", "c", "cd", "abcd", "klmn", "aaa", "abra", "111", "baba", "ababa"}
		prefix := "a"
		t.Run(name+"1", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"2", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"3", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"4", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"5", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"6", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"7", iterTest(trie_kzg_bn256.New(), scenario, prefix))
	}
	{
		name := "delete-root"
		scenario := []string{"a", "ab", "c", "cd", "abcd", "klmn", "aaa", "abra", "111", "baba", "ababa"}
		prefix := ""
		t.Run(name+"1", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"2", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"3", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"4", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"5", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"6", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"7", iterTest(trie_kzg_bn256.New(), scenario, prefix))
	}
	{
		name := "delete-none"
		scenario := []string{"a", "ab", "c", "cd", "abcd", "klmn", "aaa", "abra", "111", "baba", "ababa"}
		prefix := "---"
		t.Run(name+"1", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"2", iterTest(trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"3", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"4", iterTest(trie_blake2b.New(common.PathArity16, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"5", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize256), scenario, prefix))
		t.Run(name+"6", iterTest(trie_blake2b.New(common.PathArity2, trie_blake2b.HashSize160), scenario, prefix))
		t.Run(name+"7", iterTest(trie_kzg_bn256.New(), scenario, prefix))
	}
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
