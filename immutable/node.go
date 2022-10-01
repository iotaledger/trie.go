package immutable

import (
	"bytes"
	"encoding/hex"

	"github.com/iotaledger/trie.go/common"
)

// bufferedNode is a modified node
type bufferedNode struct {
	// persistent
	nodeFetched  *common.NodeData
	nodeModified *common.NodeData
	// non-persistent
	uncommittedChildren map[byte]*bufferedNode // children which has been modified
	triePath            []byte
}

func newBufferedNode(n *common.NodeData, triePath []byte) *bufferedNode {
	if n == nil {
		n = common.NewNodeData()
	}
	ret := &bufferedNode{
		nodeFetched:         n,
		nodeModified:        n.Clone(),
		uncommittedChildren: make(map[byte]*bufferedNode),
		triePath:            triePath,
	}
	return ret
}

func (n *bufferedNode) isRoot() bool {
	return len(n.triePath) == 0
}

// indexAsChild return index of the node as a child in the parent commitment and flag if it is a mutatedRoot
func (n *bufferedNode) indexAsChild() byte {
	common.Assert(!n.isRoot(), "indexAsChild:: receiver can't be a root node")
	return n.triePath[len(n.triePath)-1]

}

func (n *bufferedNode) setModifiedChild(child *bufferedNode, idx ...byte) {
	var index byte

	if child != nil {
		index = child.indexAsChild()
	} else {
		common.Assert(len(idx) > 0, "setModifiedChild: index of the child must be specified if the child is nil")
		index = idx[0]
	}
	n.uncommittedChildren[index] = child
	n.nodeModified.Commitment = nil
}

func (n *bufferedNode) setPathFragment(pf []byte) {
	n.nodeModified.PathFragment = pf
	if !bytes.Equal(n.nodeFetched.PathFragment, pf) {
		n.nodeModified.Commitment = nil
	}
}

func (n *bufferedNode) setTerminal(term common.TCommitment, m common.CommitmentModel) {
	n.nodeModified.Terminal = term
	if !m.EqualCommitments(n.nodeFetched.Terminal, n.nodeModified.Terminal) {
		n.nodeModified.Commitment = nil
	}
}

func (n *bufferedNode) setTriePath(triePath []byte) {
	n.triePath = triePath
}

func (n *bufferedNode) pathFragment() []byte {
	return n.nodeModified.PathFragment
}

func (n *bufferedNode) terminal() common.TCommitment {
	return n.nodeModified.Terminal
}

func (n *bufferedNode) commitment() common.VCommitment {
	return n.nodeModified.Commitment
}

func (n *bufferedNode) getChild(childIndex byte, db *immutableNodeStore) *bufferedNode {
	if ret, already := n.uncommittedChildren[childIndex]; already {
		return ret
	}
	childCommitment, ok := n.nodeFetched.ChildCommitments[childIndex]
	if !ok {
		return nil
	}
	common.Assert(!common.IsNil(childCommitment), "Trie::getChild: child commitment can be nil")
	childTriePath := common.Concat(n.triePath, n.pathFragment(), childIndex)

	nodeFetched, ok := db.FetchNodeData(childCommitment, childTriePath)
	common.Assert(ok, "Trie::getChild: can't fetch node. triePath: '%s', dbKey: '%s",
		hex.EncodeToString(common.AsKey(childCommitment)), hex.EncodeToString(childTriePath))

	return newBufferedNode(nodeFetched, childTriePath)
}

func (n *bufferedNode) isCommitted() bool {
	return !common.IsNil(n.nodeModified.Commitment)
}

// node is in the trie if at least one of the two is true:
// - it commits to terminal
// - it commits to at least 2 children
// Otherwise node has to be merged/removed
// It can only happen during deletion
func (n *bufferedNode) hasToBeRemoved(nodeStore *immutableNodeStore) (bool, *bufferedNode) {
	if n.terminal() != nil {
		return false, nil
	}
	var theOnlyChildCommitted *bufferedNode

	for i := 0; i < 256; i++ {
		child := n.getChild(byte(i), nodeStore)
		if child != nil {
			if theOnlyChildCommitted != nil {
				// at least 2 children
				return false, nil
			}
			theOnlyChildCommitted = child
		}
	}
	return true, theOnlyChildCommitted
}

//
//func ToString(n Node) string {
//	return fmt.Sprintf("nodeData(dbKey: '%s', pathFragment: '%s', term: '%s', numChildren: %d",
//		hex.EncodeToString(common.AsKey(n.commitment())),
//		hex.EncodeToString(n.pathFragment()),
//		n.terminal().String(),
//		len(n.ChildCommitments()),
//	)
//}
