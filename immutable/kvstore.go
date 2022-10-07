package immutable

import (
	"bytes"
	"encoding/hex"

	"github.com/iotaledger/trie.go/common"
)

// Update updates Trie with the unpackedKey/value. Reorganizes and re-calculates trie, keeps cache consistent
func (tr *Trie) Update(key []byte, value []byte) {
	if len(key) == 0 {
		// we never update root identity
		return
	}
	unpackedTriePath := common.UnpackBytes(key, tr.PathArity())
	if len(value) == 0 {
		tr.delete(unpackedTriePath)
	} else {
		tr.update(unpackedTriePath, value)
	}
}

// Delete deletes Key/value from the Trie
func (tr *Trie) Delete(key []byte) {
	if len(key) == 0 {
		// we do not want to delete root
		return
	}
	tr.delete(common.UnpackBytes(key, tr.PathArity()))
}

// DeletePrefix deletes all kv pairs with the prefix. It is a very fast operation, it modifies only one node
// and all children (any number) disappears from the next root
func (tr *Trie) DeletePrefix(pathPrefix []byte) bool {
	if len(pathPrefix) == 0 {
		// we do not want to delete root, or do we?
		return false
	}
	unpackedPrefix := common.UnpackBytes(pathPrefix, tr.Model().PathArity())
	return tr.deletePrefix(unpackedPrefix)
}

// Get reads the trie with the key
func (tr *TrieReader) Get(key []byte) []byte {
	//fmt.Printf("**** Get key: %s\n", string(key))
	unpackedTriePath := common.UnpackBytes(key, tr.PathArity())
	found := false
	var terminal common.TCommitment
	tr.traverseImmutablePath(unpackedTriePath, func(n *common.NodeData, _ []byte, ending ProofEndingCode) {
		//fmt.Printf("          --- traverse commitment: %s\n", n.Commitment)
		if ending == EndingTerminal {
			if !common.IsNil(n.Terminal) {
				found = true
				terminal = n.Terminal
			}
		}
	})
	if !found {
		return nil
	}
	value, valueInCommitment := common.ExtractValue(terminal)
	if valueInCommitment {
		common.Assert(len(value) > 0, "value in commitment must be not nil. Unpacked key: '%s'",
			hex.EncodeToString(unpackedTriePath))
		return value
	}
	value = tr.nodeStore.valueStore.Get(common.AsKey(terminal))
	common.Assert(len(value) > 0, "value in the value store must be not nil. Unpacked key: '%s'",
		hex.EncodeToString(unpackedTriePath))
	return value
}

// Has check existence of the key in the trie
func (tr *TrieReader) Has(key []byte) bool {
	unpackedTriePath := common.UnpackBytes(key, tr.PathArity())
	found := false
	tr.traverseImmutablePath(unpackedTriePath, func(n *common.NodeData, _ []byte, ending ProofEndingCode) {
		if ending == EndingTerminal {
			if !common.IsNil(n.Terminal) {
				found = true
			}
		}
	})
	return found
}

// Iterate iterates whole trie
func (tr *TrieReader) Iterate(f func(k []byte, v []byte) bool) {
	tr.iteratePrefix(f, nil)
}

// TrieIterator implements common.KVIterator interface for keys in the trie with given prefix
type TrieIterator struct {
	prefix []byte
	tr     *TrieReader
}

func (ti *TrieIterator) Iterate(fun func(k []byte, v []byte) bool) {
	ti.tr.iteratePrefix(fun, ti.prefix)
}

// Iterator returns iterator for the sub-trie
func (tr *TrieReader) Iterator(prefix []byte) *TrieIterator {
	return &TrieIterator{
		prefix: prefix,
		tr:     tr,
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

		childIndexContinue := pathFragmentTail[0]
		pathFragmentContinue := pathFragmentTail[1:]
		trieKeyToContinue := common.Concat(trieKey, prefix, childIndexContinue)

		prevNode.removeChild(lastNode)
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

	nodes[len(nodes)-1].setValue(nil, tr.Model())

	for i := len(nodes) - 1; i >= 1; i-- {
		idxAsChild := nodes[i].indexAsChild()
		n := tr.mergeNodeIfNeeded(nodes[i])
		if n != nil {
			nodes[i-1].removeChild(nodes[i])
			nodes[i-1].setModifiedChild(n)
		} else {
			nodes[i-1].removeChild(nil, idxAsChild)
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

// iteratePrefix iterates the key/value with keys with prefix.
// The order of the iteration will be deterministic
func (tr *TrieReader) iteratePrefix(f func(k []byte, v []byte) bool, prefix []byte) {
	var root common.VCommitment
	var triePath []byte
	unpackedPrefix := common.UnpackBytes(prefix, tr.Model().PathArity())
	tr.traverseImmutablePath(unpackedPrefix, func(n *common.NodeData, trieKey []byte, ending ProofEndingCode) {
		if bytes.HasPrefix(common.Concat(trieKey, n.PathFragment), unpackedPrefix) {
			root = n.Commitment
			triePath = trieKey
		}
	})
	if !common.IsNil(root) {
		tr.iterate(root, triePath, f)
	}
}

func (tr *TrieReader) iterate(root common.VCommitment, triePath []byte, fun func(k []byte, v []byte) bool) bool {
	n, found := tr.nodeStore.FetchNodeData(root)
	common.Assert(found, "can't fetch node. triePath: '%s', node commitment: %s", hex.EncodeToString(triePath), root)

	if !common.IsNil(n.Terminal) {
		key, err := common.PackUnpackedBytes(common.Concat(triePath, n.PathFragment), tr.Model().PathArity())
		value, inTheCommitment := n.Terminal.ExtractValue()
		if !inTheCommitment {
			value = tr.nodeStore.valueStore.Get(common.AsKey(n.Terminal))
			common.Assert(len(value) > 0, "can't fetch value. triePath: '%s', data commitment: %s", hex.EncodeToString(key), n.Terminal)
		}
		common.AssertNoError(err)
		if !fun(key, value) {
			return false
		}
	}
	for childIndex, childCommitment := range n.ChildCommitments {
		if !tr.iterate(childCommitment, common.Concat(triePath, n.PathFragment, childIndex), fun) {
			return false
		}
	}
	return true
}

// deletePrefix deletes all k/v pairs from the trie with the specified prefix
// It does nothing if prefix is nil, i.e. you can't delete the root
// return if deleted something
func (tr *Trie) deletePrefix(pathPrefix []byte) bool {
	nodes := make([]*bufferedNode, 0)

	prefixExists := false
	tr.traverseMutatedPath(pathPrefix, func(n *bufferedNode, ending ProofEndingCode) {
		nodes = append(nodes, n)
		if bytes.HasPrefix(common.Concat(n.triePath, n.nodeData.PathFragment), pathPrefix) {
			prefixExists = true
		}
	})
	if !prefixExists {
		return false
	}
	common.Assert(len(nodes) > 1, "len(nodes) > 0")
	// remove the last node and propagate

	// remove terminal and the children from the current node
	lastNode := nodes[len(nodes)-1]
	lastNode.setValue(nil, tr.Model())
	for i := 0; i < 256; i++ {
		if _, isModified := lastNode.uncommittedChildren[byte(i)]; isModified {
			lastNode.uncommittedChildren[byte(i)] = nil
			continue
		}
		if _, wasCommitted := lastNode.nodeData.ChildCommitments[byte(i)]; wasCommitted {
			lastNode.uncommittedChildren[byte(i)] = nil
		}
	}
	for i := len(nodes) - 1; i >= 1; i-- {
		idxAsChild := nodes[i].indexAsChild()
		n := tr.mergeNodeIfNeeded(nodes[i])
		if n != nil {
			nodes[i-1].removeChild(nodes[i])
			nodes[i-1].setModifiedChild(n)
		} else {
			nodes[i-1].removeChild(nil, idxAsChild)
		}
	}
	return true
}

// utility functions for testing

func (tr *TrieReader) GetStr(key string) string {
	return string(tr.Get([]byte(key)))
}

func (tr *TrieReader) HasStr(key string) bool {
	return tr.Has([]byte(key))
}

// UpdateStr updates key/value pair in the trie
func (tr *Trie) UpdateStr(key interface{}, value interface{}) {
	var k, v []byte
	if key != nil {
		switch kt := key.(type) {
		case []byte:
			k = kt
		case string:
			k = []byte(kt)
		default:
			panic("[]byte or string expected")
		}
	}
	if value != nil {
		switch vt := value.(type) {
		case []byte:
			v = vt
		case string:
			v = []byte(vt)
		default:
			panic("[]byte or string expected")
		}
	}
	tr.Update(k, v)
}

// DeleteStr removes key from trie
func (tr *Trie) DeleteStr(key interface{}) {
	var k []byte
	if key != nil {
		switch kt := key.(type) {
		case []byte:
			k = kt
		case string:
			k = []byte(kt)
		default:
			panic("[]byte or string expected")
		}
	}
	tr.Delete(k)
}
