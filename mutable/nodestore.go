package mutable

import (
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/iotaledger/trie.go/common"
)

// nodeStore direct access to trie
type nodeStore struct {
	m          common.CommitmentModel
	trieStore  common.KVReader
	valueStore common.KVReader
	arity      common.PathArity
}

func newNodeStore(trieStore, valueStore common.KVReader, model common.CommitmentModel, arity common.PathArity) *nodeStore {
	return &nodeStore{
		m:          model,
		trieStore:  trieStore,
		valueStore: valueStore,
		arity:      arity,
	}
}

func (sr *nodeStore) getNode(unpackedKey []byte) (*nodeReadOnly, bool) {
	// original (unpacked) unpackedKey is encoded to access the node in the kvstore
	encodedKey, err := common.EncodeUnpackedBytes(unpackedKey, sr.arity)
	common.Assert(err == nil, "trie::nodeStore::getNode assert 1: err: '%v' unpackedKey: '%s', arity: %s",
		err, hex.EncodeToString(unpackedKey), sr.arity.String())

	nodeBin := sr.trieStore.Get(encodedKey)
	if len(nodeBin) == 0 {
		return nil, false
	}
	n, err := nodeReadOnlyFromBytes(sr.m, nodeBin, unpackedKey, sr.arity, sr.valueStore)
	common.Assert(err == nil, "trie::nodeStore::getNode assert 2: err: '%v' nodeBin: '%s', unpackedKey: '%s', arity: %s",
		err, hex.EncodeToString(nodeBin), hex.EncodeToString(unpackedKey), sr.arity.String())
	return n, true
}

type nodeStoreBuffered struct {
	// persisted trie
	reader nodeStore
	// buffered part of the trie
	nodeCache map[string]*bufferedNode
	// cached deleted nodes
	deleted map[string]struct{}
	arity   common.PathArity
}

func newNodeStoreBuffered(model common.CommitmentModel, trieStore, valueStore common.KVReader, arity common.PathArity) *nodeStoreBuffered {
	ret := &nodeStoreBuffered{
		reader:    *newNodeStore(trieStore, valueStore, model, arity),
		nodeCache: make(map[string]*bufferedNode),
		deleted:   make(map[string]struct{}),
		arity:     arity,
	}
	return ret
}

// clone is a deep copy of the trie, including its buffered data
func (sc *nodeStoreBuffered) clone() *nodeStoreBuffered {
	ret := &nodeStoreBuffered{
		reader:    sc.reader,
		nodeCache: make(map[string]*bufferedNode),
		deleted:   make(map[string]struct{}),
		arity:     sc.arity,
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
	common.Assert(ok, "trie::mustGetNode assert missing node: key: '%s'", hex.EncodeToString(key))
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
	common.Assert(!already, "trie::insertNewNode:: node already exists, key: '%s'",
		hex.EncodeToString(n.unpackedKey))
	sc.nodeCache[string(n.unpackedKey)] = n
}

func (sc *nodeStoreBuffered) replaceNode(n *bufferedNode) {
	_, already := sc.nodeCache[string(n.unpackedKey)]
	common.Assert(already, "trie::replaceNode:: missing key: '%s'", hex.EncodeToString(n.unpackedKey))
	sc.nodeCache[string(n.unpackedKey)] = n
}

// PersistMutations persists the cache to the unpackedKey/value store
// Does not clear cache
func (sc *nodeStoreBuffered) persistMutations(store common.KVWriter) int {
	counter := 0
	for _, v := range sc.nodeCache {
		store.Set(common.MustEncodeUnpackedBytes(v.unpackedKey, sc.arity), v.Bytes(sc.reader.m, sc.arity))
		counter++
	}
	for k := range sc.deleted {
		_, inCache := sc.nodeCache[k]
		common.Assert(!inCache, "trie::persistMutations:: inconsistency. Non-existent key is marked for deletion: '%s'",
			hex.EncodeToString([]byte(k)))
		store.Set(common.MustEncodeUnpackedBytes([]byte(k), sc.arity), nil)
		counter++
	}
	return counter
}

// clearCache clears the node cache
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
