package trie_blake2b

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/iotaledger/trie.go/trie"
	"github.com/stretchr/testify/require"
)

func TestNodeSerialization(t *testing.T) {
	runTest := func(arity trie.PathArity, hashSize HashSize) {
		model := New(arity, hashSize)
		t.Run(fmt.Sprintf("1: %s: %s", arity, hashSize), func(t *testing.T) {
			n := trie.NewNodeData()
			n.ChildCommitments[0] = model.NewVectorCommitment()
			n.ChildCommitments[byte(arity)] = model.NewVectorCommitment()

			var buf bytes.Buffer
			key := []byte("abc")
			err := n.Write(&buf, arity, false, false)
			require.NoError(t, err)
			nBack, err := trie.NodeDataFromBytes(model, buf.Bytes(), key, arity, nil)
			require.NoError(t, err)

			require.True(t, trie.EqualCommitments(model.CalcNodeCommitment(n), model.CalcNodeCommitment(nBack)))
		})
		t.Run(fmt.Sprintf("2: %s: %s", arity, hashSize), func(t *testing.T) {
			n := trie.NewNodeData()
			n.Terminal = model.NewTerminalCommitment()

			var buf bytes.Buffer
			key := []byte("abc")
			err := n.Write(&buf, arity, false, false)
			require.NoError(t, err)
			nBack, err := trie.NodeDataFromBytes(model, buf.Bytes(), key, arity, nil)
			require.NoError(t, err)

			require.True(t, trie.EqualCommitments(model.CalcNodeCommitment(n), model.CalcNodeCommitment(nBack)))
		})
	}
	runTest(trie.PathArity256, HashSize256)
	runTest(trie.PathArity16, HashSize256)
	runTest(trie.PathArity2, HashSize256)
	runTest(trie.PathArity256, HashSize160)
	runTest(trie.PathArity16, HashSize160)
	runTest(trie.PathArity2, HashSize160)
}
