package immutable

import (
	"fmt"

	"github.com/iotaledger/trie.go/common"
)

// Trie is an updatable trie implemented on top of the unpackedKey/value store. It is virtualized and optimized by caching of the
// trie update operation and keeping consistent trie in the cache
type Trie struct {
	TrieReader
	mutatedRoot *bufferedNode
}

// TrieReader direct read-only access to trie
type TrieReader struct {
	nodeStore      *NodeStore
	persistentRoot common.VCommitment
}

func NewTrieUpdatable(m common.CommitmentModel, store common.KVReader, root common.VCommitment) (*Trie, error) {
	ret := &Trie{
		TrieReader: TrieReader{
			persistentRoot: root,
			nodeStore:      OpenImmutableNodeStore(store, m),
		},
	}
	if err := ret.SetRoot(root); err != nil {
		return nil, err
	}
	return ret, nil
}

func (tr *TrieReader) Root() common.VCommitment {
	return tr.persistentRoot
}

func (tr *TrieReader) Model() common.CommitmentModel {
	return tr.nodeStore.m
}

func (tr *TrieReader) PathArity() common.PathArity {
	return tr.nodeStore.m.PathArity()
}

func (tr *TrieReader) ClearCache() {
	tr.nodeStore.clearCache()
}

func (tr *Trie) SetRoot(c common.VCommitment) error {
	rootNodeData, ok := tr.nodeStore.FetchNodeData(c)
	if !ok {
		return fmt.Errorf("root commitment '%s' does not exist", c)
	}
	tr.persistentRoot = c.Clone()
	tr.mutatedRoot = newBufferedNode(rootNodeData, nil)
	return nil
}

// Commit calculates a new mutatedRoot commitment value from the cache, commits all mutations
// and writes it into the store.
// The nodes and values are written into separate partitions
// The buffered nodes are garbage collected, except the mutated ones
func (tr *Trie) Commit(store common.KVWriter) common.VCommitment {
	triePartition := common.MakeWriterPartition(store, PartitionTrieNodes)
	valuePartition := common.MakeWriterPartition(store, PartitionValues)
	commitNode(triePartition, valuePartition, tr.Model(), tr.mutatedRoot)
	return tr.mutatedRoot.nodeData.Commitment.Clone()
}

// commitNode re-calculates node commitment and, recursively, its children commitments
// Child modification marks in 'uncommittedChildren' are updated
// Return update to the upper commitment. nil mean upper commitment is not updated
// It calls implementation-specific function UpdateNodeCommitment and passes parameter
// calcDelta = true if node's commitment can be updated incrementally. The implementation
// of UpdateNodeCommitment may use this parameter to optimize underlying cryptography
//
// commitNode does not commit to the state index
func commitNode(triePartition, valuePartition common.KVWriter, m common.CommitmentModel, node *bufferedNode) {
	childUpdates := make(map[byte]common.VCommitment)
	for idx, child := range node.uncommittedChildren {
		if child == nil {
			childUpdates[idx] = nil
		} else {
			commitNode(triePartition, valuePartition, m, child)
			childUpdates[idx] = child.nodeData.Commitment
		}
	}
	m.UpdateNodeCommitment(node.nodeData, childUpdates, node.terminal, node.pathFragment, !common.IsNil(node.nodeData.Commitment))
	node.uncommittedChildren = make(map[byte]*bufferedNode)
	common.Assert(node.isCommitted(m), "node.isCommitted(m)")

	node.mustPersist(triePartition, m)
	if node.value != nil {
		valuePartition.Set(common.AsKey(node.terminal), node.value)
	}
}

func (tr *Trie) newTerminalNode(triePath, pathFragment, value []byte) *bufferedNode {
	ret := newBufferedNode(nil, triePath)
	ret.setPathFragment(pathFragment)
	ret.setValue(value, tr.Model())
	return ret
}
