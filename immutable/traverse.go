package immutable

import (
	"bytes"
	"fmt"

	"github.com/iotaledger/trie.go/common"
)

func (tr *TrieReader) traverseImmutablePath(triePath []byte, fun func(n *common.NodeData, trieKey []byte, ending ProofEndingCode)) {
	n, found := tr.nodeStore.FetchNodeData(tr.persistentRoot)
	if !found {
		return
	}
	var trieKey []byte
	for {
		keyPlusPathFragment := common.Concat(trieKey, n.PathFragment)
		switch {
		case len(triePath) < len(keyPlusPathFragment):
			fun(n, trieKey, EndingSplit)
			return
		case len(triePath) == len(keyPlusPathFragment):
			if bytes.Equal(keyPlusPathFragment, triePath) {
				fun(n, trieKey, EndingTerminal)
			} else {
				fun(n, trieKey, EndingSplit)
			}
			return
		default:
			common.Assert(len(keyPlusPathFragment) < len(triePath), "len(keyPlusPathFragment) < len(triePath)")
			prefix, _, _ := commonPrefix(keyPlusPathFragment, triePath)
			if !bytes.Equal(prefix, keyPlusPathFragment) {
				fun(n, trieKey, EndingSplit)
				return
			}
			childIndex := triePath[len(keyPlusPathFragment)]
			child, childTrieKey := FetchChild(n, childIndex, trieKey, tr.nodeStore)
			if child == nil {
				fun(n, childTrieKey, EndingExtend)
				return
			}
			fun(n, trieKey, EndingNone)
			trieKey = childTrieKey
			n = child
		}
	}
}

func (tr *Trie) traverseMutatedPath(triePath []byte, fun func(n *bufferedNode, ending ProofEndingCode)) {
	n := tr.mutatedRoot
	for {
		keyPlusPathFragment := common.Concat(n.triePath, n.pathFragment)
		switch {
		case len(triePath) < len(keyPlusPathFragment):
			fun(n, EndingSplit)
			return
		case len(triePath) == len(keyPlusPathFragment):
			if bytes.Equal(keyPlusPathFragment, triePath) {
				fun(n, EndingTerminal)
			} else {
				fun(n, EndingSplit)
			}
			return
		default:
			common.Assert(len(keyPlusPathFragment) < len(triePath), "len(keyPlusPathFragment) < len(triePath)")
			prefix, _, _ := commonPrefix(keyPlusPathFragment, triePath)
			if !bytes.Equal(prefix, keyPlusPathFragment) {
				fun(n, EndingSplit)
				return
			}
			childIndex := triePath[len(keyPlusPathFragment)]
			child := n.getChild(childIndex, tr.nodeStore)
			if child == nil {
				fun(n, EndingExtend)
				return
			}
			fun(n, EndingNone)
			n = child
		}
	}
}

func (tr *Trie) update(triePath []byte, value []byte) {
	common.Assert(len(value) > 0, "len(value)>0")

	nodes := make([]*bufferedNode, 0)
	var ends ProofEndingCode
	tr.traverseMutatedPath(triePath, func(n *bufferedNode, ending ProofEndingCode) {
		nodes = append(nodes, n)
		ends = ending
	})
	common.Assert(len(nodes) > 0, "len(nodes) > 0")
	for i := len(nodes) - 2; i >= 0; i-- {
		nodes[i].setModifiedChild(nodes[i+1])
	}
	lastNode := nodes[len(nodes)-1]
	fmt.Printf("+++++ lastnode trieKey: '%+v', pf: '%+v', ending '%+v'\n",
		lastNode.triePath, lastNode.pathFragment, ends.String())
	switch ends {
	case EndingTerminal:
		// reached the end just for the terminal
		lastNode.setValue(value, tr.Model())

	case EndingExtend:
		// extend the current node with the new terminal node
		keyPlusPathFragment := common.Concat(lastNode.triePath, lastNode.pathFragment)
		common.Assert(len(keyPlusPathFragment) < len(triePath), "len(keyPlusPathFragment) < len(triePath)")
		childTriePath := triePath[:len(keyPlusPathFragment)+1]
		childIndex := childTriePath[len(childTriePath)-1]
		common.Assert(lastNode.getChild(childIndex, tr.nodeStore) == nil, "lastNode.getChild(childIndex, tr.nodeStore)==nil")
		child := tr.newTerminalNode(childTriePath, triePath[len(keyPlusPathFragment)+1:], value)
		lastNode.setModifiedChild(child)

	case EndingSplit:
		// split the last node
		var prevNode *bufferedNode
		if len(nodes) >= 2 {
			prevNode = nodes[len(nodes)-2]
		}
		trieKey := lastNode.triePath
		common.Assert(len(trieKey) <= len(triePath), "len(trieKey) <= len(triePath)")
		remainingTriePath := triePath[len(trieKey):]

		prefix, pathFragmentTail, triePathTail := commonPrefix(lastNode.pathFragment, remainingTriePath)
		//forkPathIndex := len(prefix)
		//common.Assert(forkPathIndex < len(lastNode.pathFragment), "forkPathIndex < len(lastNode.pathFragment)")
		//common.Assert(forkPathIndex <= len(triePath), "forkPathIndex <= len(triePath)")

		childIndexContinue := pathFragmentTail[0]
		pathFragmentContinue := pathFragmentTail[1:]
		trieKeyToContinue := common.Concat(trieKey, prefix, childIndexContinue)

		prevNode.removeModifiedChild(lastNode)
		lastNode.setPathFragment(pathFragmentContinue)
		lastNode.setTriePath(trieKeyToContinue)

		forkingNode := newBufferedNode(nil, trieKey) // will be at path of the old node
		forkingNode.setPathFragment(prefix)
		forkingNode.setModifiedChild(lastNode)
		prevNode.setModifiedChild(forkingNode)

		if len(triePathTail) == 0 {
			forkingNode.setValue(value, tr.Model())
		} else {
			childIndexToBranch := triePathTail[0]
			branchPathFragment := triePathTail[1:]
			trieKeyToContinue = common.Concat(trieKey, prefix, childIndexToBranch)

			newNodeWithTerminal := tr.newTerminalNode(trieKeyToContinue, branchPathFragment, value)
			forkingNode.setModifiedChild(newNodeWithTerminal)
		}

	default:
		common.Assert(false, "inconsistency: wrong value")
	}
}

func (tr *Trie) delete(triePath []byte) {
	nodes := make([]*bufferedNode, 0)
	var ends ProofEndingCode
	tr.traverseMutatedPath(triePath, func(n *bufferedNode, ending ProofEndingCode) {
		nodes = append(nodes, n)
		ends = ending
	})
	common.Assert(len(nodes) > 0, "len(nodes) > 0")
	if ends != EndingTerminal {
		// the key is not present in the trie, do nothing
		return
	}
	//for i := len(nodes) - 2; i >= 0; i-- {
	//	nodes[i].setModifiedChild(nodes[i+1])
	//}

	nodes[len(nodes)-1].setValue(nil, tr.Model())

	for i := len(nodes) - 1; i >= 1; i-- {
		idxAsChild := nodes[i].indexAsChild()
		n := tr.mergeNodeIfNeeded(nodes[i])
		if n != nil {
			nodes[i-1].removeModifiedChild(nodes[i])
			nodes[i-1].setModifiedChild(n)
		} else {
			nodes[i-1].removeModifiedChild(nil, idxAsChild)
		}
	}
	common.Assert(nodes[0] != nil, "please do not delete root")
}

func (tr *Trie) mergeNodeIfNeeded(node *bufferedNode) *bufferedNode {
	toRemove, theOnlyChildToMergeWith := node.hasToBeRemoved(tr.nodeStore)
	if !toRemove {
		return node
	}
	if theOnlyChildToMergeWith == nil {
		// just remove
		return nil
	}
	// merge with child
	newPathFragment := common.Concat(node.pathFragment, theOnlyChildToMergeWith.indexAsChild(), theOnlyChildToMergeWith.pathFragment)
	theOnlyChildToMergeWith.setPathFragment(newPathFragment)
	theOnlyChildToMergeWith.setTriePath(node.triePath)
	return theOnlyChildToMergeWith
}

//
//func (tr *Trie) delete1(node *bufferedNode, triePath []byte) (*bufferedNode, bool) {
//	keyPlusPathFragment := common.Concat(node.triePath, node.pathFragment)
//	if len(triePath) < len(keyPlusPathFragment) {
//		return nil, false
//	}
//	if bytes.Equal(keyPlusPathFragment, triePath) {
//		if common.IsNil(node.terminal) {
//			return node, false
//		}
//		node.setValue(nil, tr.Model())
//		return tr.mergeNodeIfNeeded(node), true
//	}
//	if len(triePath) == len(keyPlusPathFragment) {
//		return node, false
//	}
//	common.Assert(len(triePath) > len(keyPlusPathFragment), "len(triePath) > len(keyPlusPathFragment)")
//	childIndex := triePath[len(keyPlusPathFragment)]
//	child := node.getChild(childIndex, tr.nodeStore)
//	if child == nil {
//		return node, false
//	}
//	ret, deleted := tr.delete1(child, triePath)
//	if deleted {
//		node.setModifiedChild(ret, childIndex)
//		return tr.mergeNodeIfNeeded(node), true
//	}
//	return node, false
//}
//
//func (tr *Trie) update1(node *bufferedNode, triePath []byte, terminal common.TCommitment) *bufferedNode {
//	trieKey := node.triePath
//	common.Assert(len(trieKey) <= len(triePath), "len(trieKey) <= len(triePath)")
//	remainingTriePath := triePath[len(trieKey):]
//
//	prefix, triePathTail, pathFragmentTail := commonPrefix(node.pathFragment, remainingTriePath)
//
//	if len(triePathTail) == 0 && len(pathFragmentTail) == 0 {
//		// it is a terminal node, finish
//		node.setValue(terminal, tr.Model())
//		return node
//	}
//
//	if len(pathFragmentTail) == 0 {
//		// nowhere to continue, extend the current node
//		common.Assert(len(triePathTail) > 0, "len(triePathTail) > 0") // we are not at the end yet
//		childIndex := triePathTail[0]                                 // we will continue with this index
//
//		nextTrieKey := common.Concat(trieKey, node.pathFragment, childIndex)
//		child := node.getChild(childIndex, tr.nodeStore)
//		if child != nil {
//			child = tr.update1(child, triePath, terminal)
//		} else {
//			child = tr.newTerminalNode(nextTrieKey, triePathTail[1:], terminal)
//		}
//		node.setModifiedChild(child, childIndex)
//		return node
//	}
//
//	// split the current node
//	forkPathIndex := len(prefix)
//	common.Assert(forkPathIndex < len(node.pathFragment), "forkPathIndex<len(node.pathFragment())")
//	common.Assert(forkPathIndex <= len(triePath), "forkPathIndex<=len(triePath)")
//
//	childIndexContinue := pathFragmentTail[0]
//	pathFragmentContinue := pathFragmentTail[1:]
//	trieKeyToContinue := common.Concat(trieKey, prefix, childIndexContinue)
//
//	node.setPathFragment(pathFragmentContinue)
//	node.setTriePath(trieKeyToContinue)
//
//	forkingNode := newBufferedNode(nil, trieKey) // will be at path of the old node
//	forkingNode.setPathFragment(prefix)
//	forkingNode.setModifiedChild(node)
//
//	if len(triePathTail) == 0 {
//		forkingNode.setValue(terminal, tr.Model())
//	} else {
//		childIndexToBranch := remainingTriePath[0]
//		trieKeyToContinue = common.Concat(trieKey, prefix, childIndexToBranch)
//
//		newNodeWithTerminal := tr.newTerminalNode(trieKeyToContinue, triePath[len(trieKeyToContinue):], terminal)
//		forkingNode.setModifiedChild(newNodeWithTerminal)
//	}
//	return forkingNode
//}
