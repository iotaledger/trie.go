package tests

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/iotaledger/trie.go/common"
	"github.com/iotaledger/trie.go/models/trie_blake2b"
	"github.com/stretchr/testify/require"
)

func TestNodeSerialization(t *testing.T) {
	dummyFun := func(_ []byte) ([]byte, error) { return nil, nil }
	runTest := func(arity common.PathArity, hashSize trie_blake2b.HashSize) {
		m := trie_blake2b.New(arity, hashSize)
		t.Run(fmt.Sprintf("1: %s: %s", arity, hashSize), func(t *testing.T) {
			n := common.NewNodeData()
			n.ChildCommitments[0] = m.NewVectorCommitment()
			n.ChildCommitments[byte(arity)] = m.NewVectorCommitment()

			var buf bytes.Buffer
			err := n.Write(&buf, arity, false)
			require.NoError(t, err)
			nBack, err := common.NodeDataFromBytes(m, buf.Bytes(), arity, dummyFun)
			require.NoError(t, err)

			require.True(t, m.EqualCommitments(m.CalcNodeCommitment(n), m.CalcNodeCommitment(nBack)))
		})
		t.Run(fmt.Sprintf("2: %s: %s", arity, hashSize), func(t *testing.T) {
			n := common.NewNodeData()
			n.Terminal = m.CommitToData([]byte("a"))

			var buf bytes.Buffer
			err := n.Write(&buf, arity, false)
			require.NoError(t, err)
			nBack, err := common.NodeDataFromBytes(m, buf.Bytes(), arity, dummyFun)
			require.NoError(t, err)

			require.True(t, m.EqualCommitments(m.CalcNodeCommitment(n), m.CalcNodeCommitment(nBack)))
		})
	}
	runTest(common.PathArity256, trie_blake2b.HashSize256)
	runTest(common.PathArity16, trie_blake2b.HashSize256)
	runTest(common.PathArity2, trie_blake2b.HashSize256)
	runTest(common.PathArity256, trie_blake2b.HashSize160)
	runTest(common.PathArity16, trie_blake2b.HashSize160)
	runTest(common.PathArity2, trie_blake2b.HashSize160)
}
