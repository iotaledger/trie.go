package immutable

import "github.com/iotaledger/trie.go/common"

type KVReaderByTrie struct {
	nodeStore immutableNodeStore
}

func NewKVReaderByTrie(nodeStore *immutableNodeStore) *KVReaderByTrie {
	common.Assert(nodeStore.valueStore != nil, "NewKVReaderTrie: value store must be provided")
	common.Assert(nodeStore.m.AlwaysStoreTerminalWithNode(), "NewKVReaderTrie: model must always force store terminal commitment in the node")
	return &KVReaderByTrie{nodeStore}
}

func (r *KVReaderByTrie) Get(key []byte) []byte {
	unpackedKey := common.UnpackBytes(key, r.nodeStore.m.PathArity())
	term := getLeafByKey(r.nodeStore, unpackedKey)
	common.Assert(term != nil, "terminal commitment must be not nil")

	return term.AsValue(r.nodeStore.ValueStore())
}

func (r *KVReaderByTrie) Has(key []byte) bool {
	unpackedKey := common.UnpackBytes(key, r.nodeStore.m.PathArity())
	term := getLeafByKey(r.nodeStore, unpackedKey)
	return term != nil
}
