package trie_blake2b

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/iotaledger/trie.go/common"
	"github.com/stretchr/testify/require"
)

func TestNodeSerialization(t *testing.T) {
	runTest := func(arity common.PathArity, hashSize HashSize) {
		m := New(arity, hashSize)
		t.Run(fmt.Sprintf("1: %s: %s", arity, hashSize), func(t *testing.T) {
			n := common.NewNodeData()
			n.ChildCommitments[0] = m.NewVectorCommitment()
			n.ChildCommitments[byte(arity)] = m.NewVectorCommitment()

			var buf bytes.Buffer
			key := []byte("abc")
			err := n.Write(&buf, arity, false, false)
			require.NoError(t, err)
			nBack, err := common.NodeDataFromBytes(m, buf.Bytes(), key, arity, nil)
			require.NoError(t, err)

			require.True(t, m.EqualCommitments(m.CalcNodeCommitment(n), m.CalcNodeCommitment(nBack)))
		})
		t.Run(fmt.Sprintf("2: %s: %s", arity, hashSize), func(t *testing.T) {
			n := common.NewNodeData()
			n.Terminal = m.CommitToData([]byte("a"))

			var buf bytes.Buffer
			key := []byte("abc")
			err := n.Write(&buf, arity, false, false)
			require.NoError(t, err)
			nBack, err := common.NodeDataFromBytes(m, buf.Bytes(), key, arity, nil)
			require.NoError(t, err)

			require.True(t, m.EqualCommitments(m.CalcNodeCommitment(n), m.CalcNodeCommitment(nBack)))
		})
	}
	runTest(common.PathArity256, HashSize256)
	runTest(common.PathArity16, HashSize256)
	runTest(common.PathArity2, HashSize256)
	runTest(common.PathArity256, HashSize160)
	runTest(common.PathArity16, HashSize160)
	runTest(common.PathArity2, HashSize160)
}
