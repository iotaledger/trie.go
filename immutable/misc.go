package immutable

import (
	"encoding/hex"

	"github.com/iotaledger/trie.go/common"
)

func FetchChild(n *common.NodeData, childIdx byte, trieKey []byte, nodeStore *NodeStore) (*common.NodeData, []byte) {
	c, childFound := n.ChildCommitments[childIdx]
	if !childFound {
		return nil, nil
	}
	common.Assert(!common.IsNil(c), "immutable::FetchChild: unexpected nil commitment")
	childTriePath := common.Concat(trieKey, n.PathFragment, childIdx)

	ret, ok := nodeStore.FetchNodeData(c)
	common.Assert(ok, "immutable::FetchChild: failed to fetch node. trieKey: '%s', childIndex: %d",
		hex.EncodeToString(trieKey), childIdx)
	return ret, childTriePath
}
