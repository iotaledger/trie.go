package immutable

import (
	"encoding/hex"

	"github.com/iotaledger/trie.go/common"
)

// immutableNodeStore direct access to trie db
type immutableNodeStore struct {
	m          common.CommitmentModel
	trieStore  common.KVReader
	valueStore common.KVReader
	arity      common.PathArity
	cache      map[string]*common.NodeData
}

func NewNodeStore(trieStore, valueStore common.KVReader, model common.CommitmentModel, arity common.PathArity) *immutableNodeStore {
	return &immutableNodeStore{
		m:          model,
		trieStore:  trieStore,
		valueStore: valueStore,
		arity:      arity,
		cache:      make(map[string]*common.NodeData),
	}
}

func noValueStore(_ []byte) ([]byte, error) {
	panic("internal inconsistency: all terminal value must be stored in the trie node")
}

func (ns *immutableNodeStore) FetchNodeData(nodeCommitment common.VCommitment, triePath []byte) (*common.NodeData, bool) {
	dbKey := nodeCommitment.AsKey()
	if ret, inCache := ns.cache[string(dbKey)]; inCache {
		return ret, true
	}
	nodeBin := ns.trieStore.Get(dbKey)
	if len(nodeBin) == 0 {
		return nil, false
	}
	ret, err := common.NodeDataFromBytes(ns.m, nodeBin, ns.arity, noValueStore)
	common.Assert(err == nil, "immutableNodeStore::FetchNodeData err: '%v' nodeBin: '%s', commitment: %s, triePath: '%s', arity: %s",
		err, hex.EncodeToString(nodeBin), nodeCommitment.String(), hex.EncodeToString(triePath), ns.arity.String())
	ret.Commitment = nodeCommitment
	return ret, true
}

func (ns *immutableNodeStore) MustFetchNodeData(nodeCommitment common.VCommitment, triePath []byte) *common.NodeData {
	ret, ok := ns.FetchNodeData(nodeCommitment, triePath)
	common.Assert(ok, "immutableNodeStore::MustFetchNodeData: cannot find node data: commitment: '%s', triePath: '%s'",
		nodeCommitment.String(), hex.EncodeToString(triePath))
	return ret
}

func (ns *immutableNodeStore) clearCache() {
	ns.cache = make(map[string]*common.NodeData)
}
