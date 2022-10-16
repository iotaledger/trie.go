package tests

import (
	"testing"

	"github.com/iotaledger/trie.go/common"
	"github.com/iotaledger/trie.go/immutable"
	"github.com/iotaledger/trie.go/models/trie_blake2b"
	"github.com/iotaledger/trie.go/models/trie_blake2b/trie_blake2b_verify"
	"github.com/stretchr/testify/require"
)

func TestProofIdentityBlake2b(t *testing.T) {
	const identity = "idididididid"
	runTest := func(arity common.PathArity, hashSize trie_blake2b.HashSize) {
		m := trie_blake2b.New(arity, hashSize)
		store := common.NewInMemoryKVStore()
		initRoot := immutable.MustInitRoot(store, m, []byte(identity))
		tr, err := immutable.NewTrieReader(m, store, initRoot)
		require.NoError(t, err)
		p := m.ProofImmutable(nil, tr)
		err = trie_blake2b_verify.Validate(p, initRoot.Bytes())
		require.NoError(t, err)

		cID := m.CommitToData([]byte(identity))
		err = trie_blake2b_verify.ValidateWithTerminal(p, initRoot.Bytes(), cID.Bytes())
		require.NoError(t, err)
	}
	runTest(common.PathArity256, trie_blake2b.HashSize256)
	runTest(common.PathArity256, trie_blake2b.HashSize160)
	runTest(common.PathArity16, trie_blake2b.HashSize256)
	runTest(common.PathArity16, trie_blake2b.HashSize160)
	runTest(common.PathArity2, trie_blake2b.HashSize256)
	runTest(common.PathArity2, trie_blake2b.HashSize160)
}

func TestProofScenariosBlake2b(t *testing.T) {
	const identity = "idididididid"
	runTest := func(arity common.PathArity, hashSize trie_blake2b.HashSize, scenario []string) {
		m := trie_blake2b.New(arity, hashSize)
		store := common.NewInMemoryKVStore()
		initRoot := immutable.MustInitRoot(store, m, []byte(identity))
		tr, err := immutable.NewTrieUpdatable(m, store, initRoot)
		require.NoError(t, err)

		checklist, root := runUpdateScenario(tr, store, scenario)
		trr, err := immutable.NewTrieReader(m, store, root)
		require.NoError(t, err)
		for k, v := range checklist {
			vBin := trr.Get([]byte(k))
			if len(v) == 0 {
				require.EqualValues(t, 0, len(vBin))
			} else {
				require.EqualValues(t, []byte(v), vBin)
			}
			p := m.ProofImmutable([]byte(k), trr)
			err = trie_blake2b_verify.Validate(p, root.Bytes())
			require.NoError(t, err)
			if len(v) > 0 {
				cID := m.CommitToData([]byte(v))
				err = trie_blake2b_verify.ValidateWithTerminal(p, root.Bytes(), cID.Bytes())
				require.NoError(t, err)
			} else {
				require.True(t, trie_blake2b_verify.IsProofOfAbsence(p))
			}
		}
	}
	runScenario := func(scenario []string) {
		runTest(common.PathArity256, trie_blake2b.HashSize256, scenario)
		runTest(common.PathArity256, trie_blake2b.HashSize160, scenario)
		runTest(common.PathArity16, trie_blake2b.HashSize256, scenario)
		runTest(common.PathArity16, trie_blake2b.HashSize160, scenario)
		runTest(common.PathArity2, trie_blake2b.HashSize256, scenario)
		runTest(common.PathArity2, trie_blake2b.HashSize160, scenario)
	}
	runScenario([]string{"a"})
	runScenario([]string{"a", "ab"})
	runScenario([]string{"a", "ab", "a/"})
	runScenario([]string{"a", "ab", "a/", "ab/"})
	runScenario([]string{"a", "ab", "abc", "a/", "ab/"})
	runScenario(genRnd3())
}
