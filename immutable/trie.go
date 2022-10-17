package immutable

import (
	"fmt"

	"github.com/iotaledger/trie.go/common"
)

// TrieUpdatable is an updatable trie implemented on top of the unpackedKey/value store. It is virtualized and optimized by caching of the
// trie update operation and keeping consistent trie in the cache
type TrieUpdatable struct {
	*TrieReader
	mutatedRoot *bufferedNode
}

// TrieReader direct read-only access to trie
type TrieReader struct {
	nodeStore      *NodeStore
	persistentRoot common.VCommitment
}

func NewTrieUpdatable(m common.CommitmentModel, store common.KVReader, root common.VCommitment, clearCacheAtSize ...int) (*TrieUpdatable, error) {
	trieReader, err := NewTrieReader(m, store, root, clearCacheAtSize...)
	if err != nil {
		return nil, err
	}
	ret := &TrieUpdatable{
		TrieReader: trieReader,
	}
	if err = ret.SetRoot(root); err != nil {
		return nil, err
	}
	return ret, nil
}

func NewTrieReader(m common.CommitmentModel, store common.KVReader, root common.VCommitment, clearCacheAtSize ...int) (*TrieReader, error) {
	ret := &TrieReader{
		nodeStore: openImmutableNodeStore(store, m, clearCacheAtSize...),
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

func (tr *TrieReader) SetRoot(c common.VCommitment) error {
	_, err := tr.setRoot(c)
	return err
}

// SetRoot fetches and sets new root. It clears cache before fetching the new root
func (tr *TrieReader) setRoot(c common.VCommitment) (*common.NodeData, error) {
	tr.ClearCache()
	rootNodeData, ok := tr.nodeStore.FetchNodeData(c)
	if !ok {
		return nil, fmt.Errorf("root commitment '%s' does not exist", c)
	}
	tr.persistentRoot = c.Clone()
	return rootNodeData, nil
}

// SetRoot overloaded for updatable trie
func (tr *TrieUpdatable) SetRoot(c common.VCommitment) error {
	rootNodeData, err := tr.TrieReader.setRoot(c)
	if err != nil {
		return err
	}
	tr.mutatedRoot = newBufferedNode(rootNodeData, nil) // the previous mutated tree will be GC-ed
	return nil
}

// Commit calculates a new mutatedRoot commitment value from the cache, commits all mutations
// and writes it into the store.
// The nodes and values are written into separate partitions
// The buffered nodes are garbage collected, except the mutated ones
// By default, it sets new root in the end and clears the trie reader cache. To override, use notSetNewRoot = true
func (tr *TrieUpdatable) Commit(store common.KVWriter) common.VCommitment {
	triePartition := common.MakeWriterPartition(store, PartitionTrieNodes)
	valuePartition := common.MakeWriterPartition(store, PartitionValues)

	tr.mutatedRoot.commitNode(triePartition, valuePartition, tr.Model())
	// set uncommitted children in the root to empty -> the GC will collect the whole tree of buffered nodes
	tr.mutatedRoot.uncommittedChildren = make(map[byte]*bufferedNode)

	ret := tr.mutatedRoot.nodeData.Commitment.Clone()
	err := tr.SetRoot(ret) // always clear cache because NodeData-s are mutated and not valid anymore
	common.AssertNoError(err)
	return ret
}

func (tr *TrieUpdatable) Persist(db common.KVBatchedWriter) (common.VCommitment, error) {
	ret := tr.Commit(db)
	if err := db.Commit(); err != nil {
		return nil, err
	}
	return ret, nil
}

func (tr *TrieUpdatable) newTerminalNode(triePath, pathFragment, value []byte) *bufferedNode {
	ret := newBufferedNode(nil, triePath)
	ret.setPathFragment(pathFragment)
	ret.setValue(value, tr.Model())
	return ret
}
