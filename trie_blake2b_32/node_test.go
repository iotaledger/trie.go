package trie_blake2b_32

import (
	"bytes"
	"fmt"
	trie_go "github.com/iotaledger/trie.go"
	"testing"

	"github.com/iotaledger/trie.go/trie256p"
	"github.com/stretchr/testify/require"
)

func TestNodeSerialization(t *testing.T) {
	model := New()
	runTest := func(arity trie256p.PathArity) {
		t.Run(fmt.Sprintf("1: arity: %s", arity), func(t *testing.T) {
			n := trie256p.NewNodeData()
			n.ChildCommitments[1] = model.NewVectorCommitment()
			n.ChildCommitments[6] = model.NewVectorCommitment()
			n.ChildCommitments[255] = model.NewVectorCommitment()

			var buf bytes.Buffer
			key := []byte("abc")
			err := n.Write(&buf, arity, false)
			require.NoError(t, err)
			nBack, err := trie256p.NodeDataFromBytes(model, buf.Bytes(), key, arity)
			require.NoError(t, err)

			require.True(t, trie_go.EqualCommitments(model.CalcNodeCommitment(n), model.CalcNodeCommitment(nBack)))
		})
		t.Run(fmt.Sprintf("2: arity: %s", arity), func(t *testing.T) {
			n := trie256p.NewNodeData()
			n.Terminal = model.NewTerminalCommitment()

			var buf bytes.Buffer
			key := []byte("abc")
			err := n.Write(&buf, arity, false)
			require.NoError(t, err)
			nBack, err := trie256p.NodeDataFromBytes(model, buf.Bytes(), key, arity)
			require.NoError(t, err)

			require.True(t, trie_go.EqualCommitments(model.CalcNodeCommitment(n), model.CalcNodeCommitment(nBack)))
		})
	}
	runTest(trie256p.PathArity256)
	runTest(trie256p.PathArity16)
	runTest(trie256p.PathArity2)
}
