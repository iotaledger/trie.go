package trie_blake2b_32

import (
	"bytes"
	trie_go "github.com/iotaledger/trie.go"
	"testing"

	"github.com/iotaledger/trie.go/trie256p"
	"github.com/stretchr/testify/require"
)

func TestNodeSerialization(t *testing.T) {
	model := New()
	t.Run("1", func(t *testing.T) {
		n := trie256p.NewNodeData()
		n.ChildCommitments[1] = model.NewVectorCommitment()
		n.ChildCommitments[6] = model.NewVectorCommitment()
		n.ChildCommitments[255] = model.NewVectorCommitment()

		var buf bytes.Buffer
		key := []byte("abc")
		err := n.Write(&buf, false)
		require.NoError(t, err)
		nBack, err := trie256p.NodeDataFromBytes(model, buf.Bytes(), key)
		require.NoError(t, err)

		require.True(t, trie_go.EqualCommitments(model.CalcNodeCommitment(n), model.CalcNodeCommitment(nBack)))
	})
	t.Run("2", func(t *testing.T) {
		n := trie256p.NewNodeData()
		n.Terminal = model.NewTerminalCommitment()

		var buf bytes.Buffer
		key := []byte("abc")
		err := n.Write(&buf, false)
		require.NoError(t, err)
		nBack, err := trie256p.NodeDataFromBytes(model, buf.Bytes(), key)
		require.NoError(t, err)

		require.True(t, trie_go.EqualCommitments(model.CalcNodeCommitment(n), model.CalcNodeCommitment(nBack)))
	})
}
