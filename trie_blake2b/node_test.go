package trie_blake2b

import (
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

		bin := n.Bytes()
		nBack, err := trie256p.NodeDataFromBytes(model, bin)
		require.NoError(t, err)
		require.EqualValues(t, n.Bytes(), nBack.Bytes())
	})
	t.Run("2", func(t *testing.T) {
		n := trie256p.NewNodeData()
		n.Terminal = model.NewTerminalCommitment()

		bin := n.Bytes()
		nBack, err := trie256p.NodeDataFromBytes(model, bin)
		require.NoError(t, err)
		require.EqualValues(t, n.Bytes(), nBack.Bytes())
	})
}
