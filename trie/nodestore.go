package trie

import (
	"fmt"
	"sort"
)

// nodeStore direct access to trie
type nodeStore struct {
	m          CommitmentModel
	trieStore  KVReader
	valueStore KVReader
	arity      PathArity
}

func newNodeStore(trieStore, valueStore KVReader, model CommitmentModel, arity PathArity) *nodeStore {
	return &nodeStore{
		m:          model,
		trieStore:  trieStore,
		valueStore: valueStore,
		arity:      arity,
	}
}

func (sr *nodeStore) getNode(unpackedKey []byte) (*nodeReadOnly, bool) {
	// original (unpacked) unpackedKey is encoded to access the node in the kvstore
	encodedKey, err := EncodeUnpackedBytes(unpackedKey, sr.arity)
	Assert(err == nil, "nodeStore::getNode: %v", err)

	nodeBin := sr.trieStore.Get(encodedKey)
	if nodeBin == nil {
		return nil, false
	}
	n, err := nodeReadOnlyFromBytes(sr.m, nodeBin, unpackedKey, sr.arity, sr.valueStore)
	Assert(err == nil, "nodeStore::getNode: %v", err)
	return n, true
}

type nodeStoreBuffered struct {
	// persisted trie
	reader nodeStore
	// buffered part of the trie
	nodeCache map[string]*bufferedNode
	// cached deleted nodes
	deleted                map[string]struct{}
	arity                  PathArity
	optimizeKeyCommitments bool
}

func newNodeStoreBuffered(model CommitmentModel, trieStore, valueStore KVReader, arity PathArity, optimizeKeyCommitments bool) *nodeStoreBuffered {
	ret := &nodeStoreBuffered{
		reader:                 *newNodeStore(trieStore, valueStore, model, arity),
		nodeCache:              make(map[string]*bufferedNode),
		deleted:                make(map[string]struct{}),
		arity:                  arity,
		optimizeKeyCommitments: optimizeKeyCommitments,
	}
	return ret
}

// clone is a deep copy of the trie, including its buffered data
func (sc *nodeStoreBuffered) clone() *nodeStoreBuffered {
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

// GetNode fetches node from the trie
func (sc *nodeStoreBuffered) getNode(unpackedKey []byte) (*bufferedNode, bool) {
	if _, isDeleted := sc.deleted[string(unpackedKey)]; isDeleted {
		return nil, false
	}
	ret, ok := sc.nodeCache[string(unpackedKey)]
	if ok {
		return ret, true
	}
	n, ok := sc.reader.getNode(unpackedKey)
	if !ok {
		return nil, false
	}
	ret = newBufferedNode(unpackedKey)
	ret.n = n.n
	ret.newTerminal = n.n.Terminal
	sc.nodeCache[string(unpackedKey)] = ret
	return ret, true
}

func (sc *nodeStoreBuffered) mustGetNode(key []byte) *bufferedNode {
	ret, ok := sc.getNode(key)
	Assert(ok, "can't find node")
	return ret
}

// removeKey marks unpackedKey deleted
func (sc *nodeStoreBuffered) removeKey(unpackedKey []byte) {
	delete(sc.nodeCache, string(unpackedKey))
	sc.deleted[string(unpackedKey)] = struct{}{}
}

// unDelete removes deletion mark, if any
func (sc *nodeStoreBuffered) unDelete(key []byte) {
	delete(sc.deleted, string(key))
}

func (sc *nodeStoreBuffered) insertNewNode(n *bufferedNode) {
	sc.unDelete(n.unpackedKey) // in case was marked deleted previously
	_, already := sc.nodeCache[string(n.unpackedKey)]
	Assert(!already, "!already")
	sc.nodeCache[string(n.unpackedKey)] = n
}

func (sc *nodeStoreBuffered) replaceNode(n *bufferedNode) {
	_, already := sc.nodeCache[string(n.unpackedKey)]
	Assert(already, "already")
	sc.nodeCache[string(n.unpackedKey)] = n
}

// PersistMutations persists the cache to the unpackedKey/value store
// Does not clear cache
func (sc *nodeStoreBuffered) persistMutations(store KVWriter) int {
	counter := 0
	for _, v := range sc.nodeCache {
		store.Set(mustEncodeUnpackedBytes(v.unpackedKey, sc.arity), v.Bytes(sc.reader.m, sc.arity, sc.optimizeKeyCommitments))
		counter++
	}
	for k := range sc.deleted {
		_, inCache := sc.nodeCache[k]
		Assert(!inCache, "!inCache")
		store.Set(mustEncodeUnpackedBytes([]byte(k), sc.arity), nil)
		counter++
	}
	return counter
}

// ClearCache clears the node cache
func (sc *nodeStoreBuffered) clearCache() {
	sc.nodeCache = make(map[string]*bufferedNode)
	sc.deleted = make(map[string]struct{})
}

func (sc *nodeStoreBuffered) dangerouslyDumpCacheToString() string {
	ret := ""
	keys := make([]string, 0)
	for k := range sc.nodeCache {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ret += fmt.Sprintf("'%s': C = %s\n%s\n", k, sc.reader.m.CalcNodeCommitment(&sc.nodeCache[k].n), sc.nodeCache[k].n.String())
	}
	return ret
}
