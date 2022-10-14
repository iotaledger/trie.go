package immutable

import (
	"fmt"

	"github.com/iotaledger/trie.go/common"
)

// Trie is an updatable trie implemented on top of the unpackedKey/value store. It is virtualized and optimized by caching of the
// trie update operation and keeping consistent trie in the cache
type Trie struct {
	*TrieReader
	mutatedRoot *bufferedNode
}

// TrieReader direct read-only access to trie
type TrieReader struct {
	nodeStore      *NodeStore
	persistentRoot common.VCommitment
}

func NewTrieUpdatable(m common.CommitmentModel, store common.KVReader, root common.VCommitment, clearCacheAtSize ...int) (*Trie, error) {
	ret := &Trie{
		TrieReader: NewTrieReader(m, store, root, clearCacheAtSize...),
	}
	if err := ret.SetRoot(root); err != nil {
		return nil, err
	}
	return ret, nil
}

func NewTrieReader(m common.CommitmentModel, store common.KVReader, root common.VCommitment, clearCacheAtSize ...int) *TrieReader {
	return &TrieReader{
		persistentRoot: root,
		nodeStore:      OpenImmutableNodeStore(store, m, clearCacheAtSize...),
	}
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

// SetRoot fetches and sets new root. By default, it clears cache before fetching the new root
// To override, use notClearCache = true
func (tr *Trie) SetRoot(c common.VCommitment, notClearCache ...bool) error {
	clearCache := true
	if len(notClearCache) > 0 {
		clearCache = !notClearCache[0]
	}
	if clearCache {
		tr.ClearCache()
	}
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
// By default, it sets new root in the end and clears the trie reader cache. To override, use notSetNewRoot = true
func (tr *Trie) Commit(store common.KVWriter, doNotSetNewRoot ...bool) common.VCommitment {
	triePartition := common.MakeWriterPartition(store, PartitionTrieNodes)
	valuePartition := common.MakeWriterPartition(store, PartitionValues)
	commitNode(triePartition, valuePartition, tr.Model(), tr.mutatedRoot)
	ret := tr.mutatedRoot.nodeData.Commitment.Clone()
	setNewRoot := true
	if len(doNotSetNewRoot) > 0 && doNotSetNewRoot[0] {
		setNewRoot = false
	}
	if setNewRoot {
		err := tr.SetRoot(ret)
		common.AssertNoError(err)
	}
	return ret
}

func (tr *Trie) Persist(db common.KVBatchedUpdater, notSetNewRoot ...bool) (common.VCommitment, error) {
	ret := tr.Commit(db, notSetNewRoot...)
	if err := db.Commit(); err != nil {
		return nil, err
	}
	return ret, nil
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
	if len(node.value) > 0 {
		valuePartition.Set(common.AsKey(node.terminal), node.value)
	}
	//fmt.Printf("commited node: trieKey: %+v('%s'): %s'\n",
	//	node.triePath, string(node.triePath), node.nodeData.String())
}

func (tr *Trie) newTerminalNode(triePath, pathFragment, value []byte) *bufferedNode {
	ret := newBufferedNode(nil, triePath)
	ret.setPathFragment(pathFragment)
	ret.setValue(value, tr.Model())
	return ret
}
