package tests

import (
	"testing"

	"github.com/iotaledger/trie.go/common"
	"github.com/iotaledger/trie.go/immutable"
	"github.com/iotaledger/trie.go/models/trie_blake2b"
	"github.com/iotaledger/trie.go/models/trie_blake2b/trie_blake2b_verify"
	"github.com/stretchr/testify/require"
)

func TestProofBasic(t *testing.T) {
	identity := "idididididid"
	m := trie_blake2b.New(common.PathArity256, trie_blake2b.HashSize256)
	store := common.NewInMemoryKVStore()
	initRoot := immutable.MustInitRoot(store, m, []byte(identity))
	tr, err := immutable.NewTrieReader(m, store, initRoot)
	require.NoError(t, err)
	p := m.ProofImmutable(nil, tr)
	err = trie_blake2b_verify.Validate(p, initRoot.Bytes())
	require.NoError(t, err)
}
