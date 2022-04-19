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

// RootCommitment is defined on NodeStore
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
	model CommitmentModel
	store trie_go.KVReader
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
	return sr.getNode(key)
}

func (sr *NodeStoreReader) getNode(key []byte) (*nodeReadOnly, bool) {
	nodeBin := sr.store.Get(key)
	if nodeBin == nil {
		return nil, false
	}
	node, err := nodeReadOnlyFromBytes(sr.model, nodeBin, key)
	trie_go.Assert(err == nil, "getNode: %v", err)
	return node, true

}

func (sr *NodeStoreReader) Model() CommitmentModel {
	return sr.model
}

// Trie is an updatable trie implemented on top of the key/value store. It is virtualized and optimized by caching of the
// trie update operation and keeping consistent trie in the cache
type Trie struct {
	// persisted trie
	nodeStoreReader NodeStoreReader
	// buffered part of the trie
	nodeCache map[string]*bufferedNode
	// cached deleted nodes
	deleted map[string]struct{}
}

// Trie implements NodeStore interface. It buffers all NodeStoreReader for optimization purposes: multiple updates of trie do not require DB NodeStoreReader
var _ NodeStore = &Trie{}

func New(model CommitmentModel, store trie_go.KVReader) *Trie {
	ret := &Trie{
		nodeStoreReader: *NewNodeStoreReader(store, model),
		nodeCache:       make(map[string]*bufferedNode),
		deleted:         make(map[string]struct{}),
	}
	return ret
}

// Clone is a deep copy of the trie, including its buffered data
func (tr *Trie) Clone() *Trie {
	ret := &Trie{
		nodeStoreReader: tr.nodeStoreReader,
		nodeCache:       make(map[string]*bufferedNode),
		deleted:         make(map[string]struct{}),
	}
	for k, v := range tr.nodeCache {
		ret.nodeCache[k] = v.Clone()
	}
	for k := range tr.deleted {
		ret.deleted[k] = struct{}{}
	}
	return ret
}

// GetNode fetches node from the trie
func (tr *Trie) GetNode(key []byte) (Node, bool) {
	return tr.getNode(key)
}

func (tr *Trie) Model() CommitmentModel {
	return tr.nodeStoreReader.model
}

// getNode takes node form the cache or fetches it from stored tries
func (tr *Trie) getNode(key []byte) (*bufferedNode, bool) {
	if _, isDeleted := tr.deleted[string(key)]; isDeleted {
		return nil, false
	}
	ret, ok := tr.nodeCache[string(key)]
	if ok {
		return ret, true
	}
	n, ok := tr.nodeStoreReader.getNode(key)
	if !ok {
		return nil, false
	}
	ret = newBufferedNode(key)
	ret.n = n.n
	ret.newTerminal = n.n.Terminal
	tr.nodeCache[string(key)] = ret
	return ret, true
}

func (tr *Trie) mustGetNode(key []byte) *bufferedNode {
	ret, ok := tr.getNode(key)
	trie_go.Assert(ok, "mustGetNode: not found key '%x'", key)
	return ret
}

// removeKey marks key deleted
func (tr *Trie) removeKey(key []byte) {
	delete(tr.nodeCache, string(key))
	tr.deleted[string(key)] = struct{}{}
}

// unDelete removes deletion mark, if any
func (tr *Trie) unDelete(key []byte) {
	delete(tr.deleted, string(key))
}

func (tr *Trie) insertNewNode(n *bufferedNode) {
	tr.unDelete(n.key) // in case was marked deleted previously
	_, already := tr.nodeCache[string(n.key)]
	trie_go.Assert(!already, "!already")
	tr.nodeCache[string(n.key)] = n
}

func (tr *Trie) replaceNode(n *bufferedNode) {
	_, already := tr.nodeCache[string(n.key)]
	trie_go.Assert(already, "already")
	tr.nodeCache[string(n.key)] = n
}

// PersistMutations persists the cache to the key/value store
// Does not clear cache
func (tr *Trie) PersistMutations(store trie_go.KVWriter) {
	for k, v := range tr.nodeCache {
		store.Set([]byte(k), v.Bytes())
	}
	for k := range tr.deleted {
		_, inCache := tr.nodeCache[k]
		trie_go.Assert(!inCache, "!inCache")
		store.Set([]byte(k), nil)
	}
}

// ClearCache clears the node cache
func (tr *Trie) ClearCache() {
	tr.nodeCache = make(map[string]*bufferedNode)
	tr.deleted = make(map[string]struct{})
}
