package immutable

import (
	"encoding/hex"

	"github.com/iotaledger/trie.go/common"
)

// NodeStore direct access to trie db
type NodeStore struct {
	m          common.CommitmentModel
	trieStore  common.KVReader
	valueStore common.KVReader
	arity      common.PathArity
	cache      map[string]*common.NodeData
}

func NewNodeStore(trieStore, valueStore common.KVReader, model common.CommitmentModel, arity common.PathArity) *NodeStore {
	return &NodeStore{
		m:          model,
		trieStore:  trieStore,
		valueStore: valueStore,
		arity:      arity,
		cache:      make(map[string]*common.NodeData),
	}
}

func (ns *NodeStore) FetchNodeData(dbKey, triePath []byte) (*common.NodeData, bool) {
	if ret, inCache := ns.cache[string(dbKey)]; inCache {
		return ret, true
	}
	nodeBin := ns.trieStore.Get(dbKey)
	if len(nodeBin) == 0 {
		return nil, false
	}
	ret, err := common.NodeDataFromBytes(ns.m, nodeBin, triePath, dbKey, ns.arity, ns.valueStore)
	common.Assert(err == nil, "trie::trieBuffer::FetchNodeData err: '%v' nodeBin: '%s', unpackedKey: '%s', arity: %s",
		err, hex.EncodeToString(nodeBin), hex.EncodeToString(dbKey), ns.arity.String())
	return ret, true
}

func (ns *NodeStore) MustFetchNodeData(dbKey, triePath []byte) *common.NodeData {
	ret, ok := ns.FetchNodeData(dbKey, triePath)
	common.Assert(ok, "trie::trieBuffer::MustFetchNodeData: cannot find node data: dbKey: '%s', triePath: '%s'",
		hex.EncodeToString(dbKey), hex.EncodeToString(triePath))
	return ret
}

func (ns *NodeStore) ClearCache() {
	ns.cache = make(map[string]*common.NodeData)
}
