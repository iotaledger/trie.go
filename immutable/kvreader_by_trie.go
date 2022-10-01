package immutable

type KVReaderByTrie struct {
	nodeStore NodeStore
}

func NewKVReaderByTrie(nodeStore NodeStore) *KVReaderByTrie {
	Assert(nodeStore.ValueStore() != nil, "NewKVReaderTrie: value store must be provided")
	Assert(nodeStore.Model().AlwaysStoreTerminalInNode(), "NewKVReaderTrie: model must always force store terminal commitment in the node")
	return &KVReaderByTrie{nodeStore}
}

func (r *KVReaderByTrie) Get(key []byte) []byte {
	unpackedKey := UnpackBytes(key, r.nodeStore.PathArity())
	term := getLeafByKey(r.nodeStore, unpackedKey)
	Assert(term != nil, "terminal commitment must be not nil")

	return term.AsValue(r.nodeStore.ValueStore())
}

func (r *KVReaderByTrie) Has(key []byte) bool {
	unpackedKey := UnpackBytes(key, r.nodeStore.PathArity())
	term := getLeafByKey(r.nodeStore, unpackedKey)
	return term != nil
}
