package trie256p

import trie_go "github.com/iotaledger/trie.go"

// nodeStoreBackend is access with real encoded keys
type nodeStoreBackend interface {
	getNode(key []byte) (Node, bool)
	model() CommitmentModel
}

// NodeStoreReader direct access to trie
type nodeStore struct {
	m     CommitmentModel
	store trie_go.KVReader
}

// NodeStoreReader implements NodeStore
var _ nodeStoreBackend = &nodeStore{}

func newNodeStore(store trie_go.KVReader, model CommitmentModel) *nodeStore {
	return &nodeStore{
		m:     model,
		store: store,
	}
}

func (sr *nodeStore) getNode(key []byte) (Node, bool) {
	return sr.getNodeIntern(key)
}

func (sr *nodeStore) getNodeIntern(key []byte) (*nodeReadOnly, bool) {
	nodeBin := sr.store.Get(key)
	if nodeBin == nil {
		return nil, false
	}
	n, err := nodeReadOnlyFromBytes(sr.m, nodeBin, key)
	trie_go.Assert(err == nil, "nodeStore::getNodeIntern: %v", err)
	return n, true
}

func (sr *nodeStore) model() CommitmentModel {
	return sr.m
}

type nodeStoreBuffered struct {
	// persisted trie
	reader nodeStore
	// buffered part of the trie
	nodeCache map[string]*bufferedNode
	// cached deleted nodes
	deleted map[string]struct{}
}

// nodeStoreBuffered implements nodeStoreBackend interface.
var _ nodeStoreBackend = &nodeStoreBuffered{}

func newNodeStoreBuffered(model CommitmentModel, store trie_go.KVReader) *nodeStoreBuffered {
	ret := &nodeStoreBuffered{
		reader:    *newNodeStore(store, model),
		nodeCache: make(map[string]*bufferedNode),
		deleted:   make(map[string]struct{}),
	}
	return ret
}

// Clone is a deep copy of the trie, including its buffered data
func (sc *nodeStoreBuffered) Clone() *nodeStoreBuffered {
	ret := &nodeStoreBuffered{
		reader:    sc.reader,
		nodeCache: make(map[string]*bufferedNode),
		deleted:   make(map[string]struct{}),
	}
	for k, v := range sc.nodeCache {
		ret.nodeCache[k] = v.Clone()
	}
	for k := range sc.deleted {
		ret.deleted[k] = struct{}{}
	}
	return ret
}

func (sc *nodeStoreBuffered) model() CommitmentModel {
	return sc.reader.m
}

func (sc *nodeStoreBuffered) getNode(key []byte) (Node, bool) {
	return sc.getNodeIntern(key)
}

// GetNode fetches node from the trie
func (sc *nodeStoreBuffered) getNodeIntern(key []byte) (*bufferedNode, bool) {
	if _, isDeleted := sc.deleted[string(key)]; isDeleted {
		return nil, false
	}
	ret, ok := sc.nodeCache[string(key)]
	if ok {
		return ret, true
	}
	n, ok := sc.reader.getNodeIntern(key)
	if !ok {
		return nil, false
	}
	ret = newBufferedNode(key)
	ret.n = n.n
	ret.newTerminal = n.n.Terminal
	sc.nodeCache[string(key)] = ret
	return ret, true
}

// removeKey marks key deleted
func (sc *nodeStoreBuffered) removeKey(key []byte) {
	delete(sc.nodeCache, string(key))
	sc.deleted[string(key)] = struct{}{}
}

// unDelete removes deletion mark, if any
func (sc *nodeStoreBuffered) unDelete(key []byte) {
	delete(sc.deleted, string(key))
}

func (sc *nodeStoreBuffered) insertNewNode(n *bufferedNode) {
	sc.unDelete(n.key) // in case was marked deleted previously
	_, already := sc.nodeCache[string(n.key)]
	trie_go.Assert(!already, "!already")
	sc.nodeCache[string(n.key)] = n
}

func (sc *nodeStoreBuffered) replaceNode(n *bufferedNode) {
	_, already := sc.nodeCache[string(n.key)]
	trie_go.Assert(already, "already")
	sc.nodeCache[string(n.key)] = n
}
