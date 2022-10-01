package immutable

import (
	"encoding/hex"

	"github.com/iotaledger/trie.go/common"
)

func FetchChild(n *common.NodeData, childIdx byte, triePath []byte, nodeStore *NodeStore) (*common.NodeData, []byte) {
	c, childFound := n.ChildCommitments[childIdx]
	if !childFound {
		return nil, nil
	}
	common.Assert(!common.IsNil(c), "NodeData::FetchChild: unexpected nil commitment")
	childTriePath := common.Concat(triePath, n.PathFragment, childIdx)

	ret, ok := nodeStore.FetchNodeData(common.AsKey(c), childTriePath)
	common.Assert(ok, "Trie::getChild: can't fetch node. triePath: '%s', childIndex: %d",
		hex.EncodeToString(triePath), childIdx)
	return ret, childTriePath
}
