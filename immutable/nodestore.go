package immutable

import (
	"encoding/hex"

	"github.com/iotaledger/trie.go/common"
)

// NodeStore immutable node store
type NodeStore struct {
	m          common.CommitmentModel
	trieStore  common.KVReader
	valueStore common.KVReader
	cache      map[string]*common.NodeData
}

const (
	PartitionTrieNodes = byte(iota)
	PartitionValues
	PartitionOther
)

// MustInitRoot initializes new empty root with the given identity
func MustInitRoot(store common.KVStore, m common.CommitmentModel, identity []byte) common.VCommitment {
	common.Assert(len(identity) > 0, "MustInitRoot: identity of the root cannot be empty")
	// create a node with the commitment to the identity as terminal for the root
	// stores identity in the value store if it does not fit the commitment
	// assigns state index 0
	rootNodeData := common.NewNodeData()
	n := newBufferedNode(rootNodeData, nil)
	n.setValue(identity, m)

	trieStore := common.MakeWriterPartition(store, PartitionTrieNodes)
	valueStore := common.MakeWriterPartition(store, PartitionValues)
	commitNode(trieStore, valueStore, m, n)

	return n.nodeData.Commitment.Clone()
}

func OpenImmutableNodeStore(store common.KVReader, model common.CommitmentModel) *NodeStore {
	return &NodeStore{
		m:          model,
		trieStore:  common.MakeReaderPartition(store, PartitionTrieNodes),
		valueStore: common.MakeReaderPartition(store, PartitionValues),
		cache:      make(map[string]*common.NodeData),
	}
}

func noValueStore(_ []byte) ([]byte, error) {
	panic("internal inconsistency: all terminal value must be stored in the trie node")
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

func (ns *NodeStore) FetchChild(n *common.NodeData, childIdx byte, trieKey []byte) (*common.NodeData, []byte) {
	c, childFound := n.ChildCommitments[childIdx]
	if !childFound {
		return nil, nil
	}
	common.Assert(!common.IsNil(c), "immutable::FetchChild: unexpected nil commitment")
	childTriePath := common.Concat(trieKey, n.PathFragment, childIdx)

	ret, ok := ns.FetchNodeData(c)
	common.Assert(ok, "immutable::FetchChild: failed to fetch node. trieKey: '%s', childIndex: %d",
		hex.EncodeToString(trieKey), childIdx)
	return ret, childTriePath
}

func (ns *NodeStore) clearCache() {
	ns.cache = make(map[string]*common.NodeData)
}
