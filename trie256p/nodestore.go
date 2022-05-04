package trie256p

import (
	trie_go "github.com/iotaledger/trie.go"
)

// NodeStore is an interface to NodeStoreReader to the trie as a set of NodeStoreReader represented as key/value pairs
// Two implementations:
// - NodeStoreReader is a direct, non-cached NodeStoreReader to key/value storage
// - Trie implement a cached NodeStoreReader
type NodeStore interface {
	GetNode(key []byte) (Node, bool)
	Model() CommitmentModel
}

// RootCommitment computes root commitment from the root node of the trie
func RootCommitment(tr NodeStore) trie_go.VCommitment {
	n, ok := tr.GetNode(nil)
	if !ok {
		return nil
	}
	return tr.Model().CalcNodeCommitment(&NodeData{
		PathFragment:     n.PathFragment(),
		ChildCommitments: n.ChildCommitments(),
		Terminal:         n.Terminal(),
	})
}

// NodeStoreReader direct access to trie
type NodeStoreReader struct {
	arity  Arity
	reader *NodeStoreReader
}

// NodeStoreReader implements NodeStore
var _ NodeStore = &NodeStoreReader{}

func NewNodeStoreReader(store trie_go.KVReader, model CommitmentModel) *NodeStoreReader {
	return &NodeStoreReader{
		model: model,
		store: store,
	}
}

func (sr *NodeStoreReader) GetNode(key []byte) (Node, bool) {
	return sr.getNodeIntern(key)
}

func (sr *NodeStoreReader) getNodeIntern(key []byte) (*nodeReadOnly, bool) {
	nodeBin := sr.store.Get(key)
	if nodeBin == nil {
		return nil, false
	}
	node, err := nodeReadOnlyFromBytes(sr.model, nodeBin, key)
	trie_go.Assert(err == nil, "getNodeIntern: %v", err)
	return node, true

}

func (sr *NodeStoreReader) Model() CommitmentModel {
	return sr.model
}

// Trie is an updatable trie implemented on top of the key/value store. It is virtualized and optimized by caching of the
// trie update operation and keeping consistent trie in the cache
type Trie struct {
	nodeStore              *nodeStoreBuffered
	arity                  Arity
	optimizeKeyCommitments bool
}

// Trie implements NodeStore interface. It buffers all NodeStoreReader for optimization purposes: multiple updates of trie do not require DB NodeStoreReader
var _ NodeStore = &Trie{}

func New(model CommitmentModel, store trie_go.KVReader, arity Arity, optimizeKeyCommitments bool) *Trie {
	ret := &Trie{
		nodeStore:              newNodeStoreBuffered(model, store),
		arity:                  arity,
		optimizeKeyCommitments: optimizeKeyCommitments,
	}
	return ret
}

// Clone is a deep copy of the trie, including its buffered data
func (tr *Trie) Clone() *Trie {
	return &Trie{
		nodeStore:              tr.nodeStore.Clone(),
		arity:                  tr.arity,
		optimizeKeyCommitments: tr.optimizeKeyCommitments,
	}
}

func (tr *Trie) Model() CommitmentModel {
	return tr.nodeStore.reader.m
}

// GetNode fetches node from the trie
func (tr *Trie) GetNode(key []byte) (Node, bool) {
	// TODO encode
	return tr.nodeStore.getNode(key)
}

// PersistMutations persists the cache to the key/value store
// Does not clear cache
func (tr *Trie) PersistMutations(store trie_go.KVWriter) int {
	counter := 0
	for _, v := range tr.nodeCache {
		store.Set(v.key, v.Bytes(tr.Model()))
		counter++
	}
	for k := range tr.deleted {
		_, inCache := tr.nodeCache[k]
		trie_go.Assert(!inCache, "!inCache")
		store.Set([]byte(k), nil)
		counter++
	}
	return counter
}

// ClearCache clears the node cache
func (tr *Trie) ClearCache() {
	tr.nodeCache = make(map[string]*bufferedNode)
	tr.deleted = make(map[string]struct{})
}
