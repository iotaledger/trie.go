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
	cache      map[string]*common.NodeData
}

const (
	TrieStorePartition = byte(iota)
	TrieStorePartitionValue
	TrieStorePartitionOther
)

// MustInitRoot initializes new empty root with the given identity
func MustInitRoot(store common.KVStore, m common.CommitmentModel, identity []byte) common.VCommitment {
	common.Assert(len(identity) > 0, "MustInitRoot: identity of the root cannot be empty")
	parts := common.WriterPartitions(store, TrieStorePartition, TrieStorePartitionValue)
	// create a node with the commitment to the identity as terminal for the root
	// stores identity in the value store if it does not fit the commitment
	// assigns state index 0
	rootNodeData := common.NewNodeData()
	rootNodeData.Terminal = m.CommitToData(identity)
	n := newBufferedNode(rootNodeData, nil)

	commitNode(m, n)
	rootNodeData.StateIndex = new(uint32)
	// persist the node
	n.mustPersist(parts[0], m)
	_, dataIsInCommitment := m.ExtractDataFromTCommitment(rootNodeData.Terminal)
	// persist the value if needed
	if !dataIsInCommitment {
		parts[1].Set(rootNodeData.Terminal.Bytes(), identity)
	}
	return n.nodeData.Commitment
}

func OpenNodeStore(store common.KVReader, model common.CommitmentModel) *NodeStore {
	parts := common.ReaderPartitions(store, TrieStorePartition, TrieStorePartitionValue)
	return &NodeStore{
		m:          model,
		trieStore:  parts[0],
		valueStore: parts[1],
		cache:      make(map[string]*common.NodeData),
	}
}

func noValueStore(_ []byte) ([]byte, error) {
	panic("internal inconsistency: all terminal value must be stored in the trie node")
}

func (ns *NodeStore) StateIndexAtNode(c common.VCommitment) (uint32, bool) {
	nodeData, found := ns.FetchNodeData(c)
	if !found {
		return 0, false
	}
	if nodeData.StateIndex == nil {
		return 0, false
	}
	return *nodeData.StateIndex, true
}

func (ns *NodeStore) FetchNodeData(nodeCommitment common.VCommitment) (*common.NodeData, bool) {
	dbKey := common.AsKey(nodeCommitment)
	if ret, inCache := ns.cache[string(dbKey)]; inCache {
		return ret, true
	}
	nodeBin := ns.trieStore.Get(dbKey)
	if len(nodeBin) == 0 {
		return nil, false
	}
	ret, err := common.NodeDataFromBytes(ns.m, nodeBin, ns.m.PathArity(), noValueStore)
	common.Assert(err == nil, "NodeStore::FetchNodeData err: '%v' nodeBin: '%s', commitment: %s, arity: %s",
		err, hex.EncodeToString(nodeBin), nodeCommitment, ns.m.PathArity())
	ret.Commitment = nodeCommitment
	return ret, true
}

func (ns *NodeStore) MustFetchNodeData(nodeCommitment common.VCommitment) *common.NodeData {
	ret, ok := ns.FetchNodeData(nodeCommitment)
	common.Assert(ok, "NodeStore::MustFetchNodeData: cannot find node data: commitment: '%s'", nodeCommitment.String())
	return ret
}

func (ns *NodeStore) clearCache() {
	ns.cache = make(map[string]*common.NodeData)
}
