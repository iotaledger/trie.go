package immutable

import (
	"strings"
	"testing"

	"github.com/iotaledger/trie.go/common"
	"github.com/iotaledger/trie.go/models/trie_blake2b"
	"github.com/iotaledger/trie.go/models/trie_kzg_bn256"
	"github.com/stretchr/testify/require"
)

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