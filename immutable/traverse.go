package immutable

import (
	"bytes"

	"github.com/iotaledger/trie.go/common"
)

// PathElement proof element is common.NodeData together with the index of
// the next child in the path (except the last one in the proof path)
// Sequence of PathElement is used to generate proof
type PathElement struct {
	NodeData   *common.NodeData
	ChildIndex byte
}

// NodePath returns path PathElement-s along the triePath (the key) with the ending code
// to determine is it a proof of inclusion or absence
// Each path element contains index of the subsequent child, except the last one is set to 0
func (tr *TrieReader) NodePath(triePath []byte) ([]*PathElement, common.PathEndingCode) {
	ret := make([]*PathElement, 0)
	var endingCode common.PathEndingCode
	tr.traverseImmutablePath(triePath, func(n *common.NodeData, trieKey []byte, ending common.PathEndingCode) {
		elem := &PathElement{
			NodeData: n,
		}
		nextChildIdx := len(trieKey) + len(n.PathFragment)
		if nextChildIdx < len(triePath) {
			elem.ChildIndex = triePath[nextChildIdx]
		}
		endingCode = ending
		ret = append(ret, elem)
	})
	common.Assert(len(ret) > 0, "len(ret)>0")
	ret[len(ret)-1].ChildIndex = 0
	return ret, endingCode
}

func (tr *TrieReader) traverseImmutablePath(triePath []byte, fun func(n *common.NodeData, trieKey []byte, ending common.PathEndingCode)) {
	n, found := tr.nodeStore.FetchNodeData(tr.persistentRoot)
	if !found {
		return
	}
	var trieKey []byte
	for {
		keyPlusPathFragment := common.Concat(trieKey, n.PathFragment)
		switch {
		case len(triePath) < len(keyPlusPathFragment):
			fun(n, trieKey, common.EndingSplit)
			return
		case len(triePath) == len(keyPlusPathFragment):
			if bytes.Equal(keyPlusPathFragment, triePath) {
				fun(n, trieKey, common.EndingTerminal)
			} else {
				fun(n, trieKey, common.EndingSplit)
			}
			return
		default:
			common.Assert(len(keyPlusPathFragment) < len(triePath), "len(keyPlusPathFragment) < len(triePath)")
			prefix, _, _ := commonPrefix(keyPlusPathFragment, triePath)
			if !bytes.Equal(prefix, keyPlusPathFragment) {
				fun(n, trieKey, common.EndingSplit)
				return
			}
			childIndex := triePath[len(keyPlusPathFragment)]
			child, childTrieKey := tr.nodeStore.FetchChild(n, childIndex, trieKey)
			if child == nil {
				fun(n, childTrieKey, common.EndingExtend)
				return
			}
			fun(n, trieKey, common.EndingNone)
			trieKey = childTrieKey
			n = child
		}
	}
}

func (tr *TrieUpdatable) traverseMutatedPath(triePath []byte, fun func(n *bufferedNode, ending common.PathEndingCode)) {
	n := tr.mutatedRoot
	for {
		keyPlusPathFragment := common.Concat(n.triePath, n.pathFragment)
		switch {
		case len(triePath) < len(keyPlusPathFragment):
			fun(n, common.EndingSplit)
			return
		case len(triePath) == len(keyPlusPathFragment):
			if bytes.Equal(keyPlusPathFragment, triePath) {
				fun(n, common.EndingTerminal)
			} else {
				fun(n, common.EndingSplit)
			}
			return
		default:
			common.Assert(len(keyPlusPathFragment) < len(triePath), "len(keyPlusPathFragment) < len(triePath)")
			prefix, _, _ := commonPrefix(keyPlusPathFragment, triePath)
			if !bytes.Equal(prefix, keyPlusPathFragment) {
				fun(n, common.EndingSplit)
				return
			}
			childIndex := triePath[len(keyPlusPathFragment)]
			child := n.getChild(childIndex, tr.nodeStore)
			if child == nil {
				fun(n, common.EndingExtend)
				return
			}
			fun(n, common.EndingNone)
			n = child
		}
	}
}

func commonPrefix(b1, b2 []byte) ([]byte, []byte, []byte) {
	ret := make([]byte, 0)
	i := 0
	for ; i < len(b1) && i < len(b2); i++ {
		if b1[i] != b2[i] {
			break
		}
		ret = append(ret, b1[i])
	}
	var r1, r2 []byte
	if i < len(b1) {
		r1 = b1[i:]
	}
	if i < len(b2) {
		r2 = b2[i:]
	}

	return ret, r1, r2
}
