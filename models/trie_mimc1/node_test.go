package trie_mimc1

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/iotaledger/trie.go/trie"
	"github.com/stretchr/testify/require"
)

func TestNodeSerialization(t *testing.T) {
	runTest := func(arity trie.PathArity) {
		model := New(arity)
		t.Run(fmt.Sprintf("1: %s", arity), func(t *testing.T) {
			n := trie.NewNodeData()
			n.ChildCommitments[0] = model.NewVectorCommitment()
			n.ChildCommitments[byte(arity)] = model.NewVectorCommitment()

			var buf bytes.Buffer
			key := []byte("abc")
			err := n.Write(&buf, arity, false, false)
			require.NoError(t, err)
			nBack, err := trie.NodeDataFromBytes(model, buf.Bytes(), key, arity, nil)
			require.NoError(t, err)

			require.True(t, model.EqualCommitments(model.CalcNodeCommitment(n), model.CalcNodeCommitment(nBack)))
		})
		t.Run(fmt.Sprintf("2: %s", arity), func(t *testing.T) {
			n := trie.NewNodeData()
			n.Terminal = model.CommitToData([]byte("a"))

			var buf bytes.Buffer
			key := []byte("abc")
			err := n.Write(&buf, arity, false, false)
			require.NoError(t, err)
			nBack, err := trie.NodeDataFromBytes(model, buf.Bytes(), key, arity, nil)
			require.NoError(t, err)

			require.True(t, model.EqualCommitments(model.CalcNodeCommitment(n), model.CalcNodeCommitment(nBack)))
		})
	}
	runTest(trie.PathArity256)
	runTest(trie.PathArity16)
	runTest(trie.PathArity2)
}
