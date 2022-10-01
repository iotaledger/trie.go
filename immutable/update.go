package immutable

import "bytes"

func (tr *Trie) update(node *bufferedNode, triePath []byte, terminal TCommitment) *bufferedNode {
	trieKey := node.triePath
	Assert(len(trieKey) <= len(triePath), "len(trieKey) <= len(triePath)")
	remainingTriePath := triePath[len(trieKey):]

	prefix, triePathTail, pathFragmentTail := commonPrefix(node.PathFragment(), remainingTriePath)

	if len(triePathTail) == 0 && len(pathFragmentTail) == 0 {
		// it is a terminal node, finish
		node.setTerminal(terminal, tr.Model())
		return node
	}

	if len(pathFragmentTail) == 0 {
		// nowhere to continue, extend the current node
		Assert(len(triePathTail) > 0, "len(triePathTail) > 0") // we are not at the end yet
		childIndex := triePathTail[0]                          // we will continue with this index

		nextTrieKey := Concat(trieKey, node.PathFragment(), childIndex)
		child := node.getChild(childIndex, tr.nodeStore)
		if child != nil {
			child = tr.update(child, triePath, terminal)
		} else {
			child = tr.newTerminalNode(nextTrieKey, triePathTail[1:], terminal)
		}
		node.setModifiedChild(child, childIndex)
		return node
	}

	// split the current node
	forkPathIndex := len(prefix)
	Assert(forkPathIndex < len(node.PathFragment()), "forkPathIndex<len(node.PathFragment())")
	Assert(forkPathIndex <= len(triePath), "forkPathIndex<=len(triePath)")

	childIndexContinue := pathFragmentTail[0]
	pathFragmentContinue := pathFragmentTail[1:]
	trieKeyToContinue := Concat(trieKey, prefix, childIndexContinue)

	node.setPathFragment(pathFragmentContinue)
	node.setTriePath(trieKeyToContinue)

	forkingNode := newBufferedNode(nil, trieKey) // will be at path of the old node
	forkingNode.setPathFragment(prefix)
	forkingNode.setModifiedChild(node)

	if len(triePathTail) == 0 {
		forkingNode.setTerminal(terminal, tr.Model())
	} else {
		childIndexToBranch := remainingTriePath[0]
		trieKeyToContinue = Concat(trieKey, prefix, childIndexToBranch)

		newNodeWithTerminal := tr.newTerminalNode(trieKeyToContinue, triePath[len(trieKeyToContinue):], terminal)
		forkingNode.setModifiedChild(newNodeWithTerminal)
	}
	return forkingNode
}

func (tr *Trie) delete(node *bufferedNode, triePath []byte) (*bufferedNode, bool) {
	keyPlusPathFragment := Concat(node.triePath, node.PathFragment())
	if len(triePath) < len(keyPlusPathFragment) {
		return nil, false
	}
	if bytes.Equal(keyPlusPathFragment, triePath) {
		if IsNil(node.Terminal()) {
			return node, false
		}
		node.setTerminal(nil, tr.Model())
		return tr.mergeNodeIfNeeded(node), true
	}
	if len(triePath) == len(keyPlusPathFragment) {
		return node, false
	}
	Assert(len(triePath) > len(keyPlusPathFragment), "len(triePath) > len(keyPlusPathFragment)")
	childIndex := triePath[len(keyPlusPathFragment)]
	child := node.getChild(childIndex, tr.nodeStore)
	if child == nil {
		return node, false
	}
	ret, deleted := tr.delete(child, triePath)
	if deleted {
		node.setModifiedChild(ret, childIndex)
		return tr.mergeNodeIfNeeded(node), true
	}
	return node, false
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
	newPathFragment := Concat(node.PathFragment(), theOnlyChildToMergeWith.indexAsChild(), theOnlyChildToMergeWith.PathFragment())
	theOnlyChildToMergeWith.setPathFragment(newPathFragment)
	return theOnlyChildToMergeWith
}
