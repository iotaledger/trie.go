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

type PathEndingCode byte

const (
	EndingNone = PathEndingCode(iota)
	EndingTerminal
	EndingSplit
	EndingExtend
)

func (e PathEndingCode) String() string {
	switch e {
	case EndingNone:
		return "EndingNone"
	case EndingTerminal:
		return "EndingTerminal"
	case EndingSplit:
		return "EndingSplit"
	case EndingExtend:
		return "EndingExtend"
	default:
		panic("wrong ending code")
	}
}

// NodePath returns path PathElement-s along the triePath (the key) with the ending code
// to determine is it is proof of inclusion or absence
func (tr *TrieReader) NodePath(triePath []byte) ([]*PathElement, PathEndingCode) {
	ret := make([]*PathElement, 0)
	var endingCode PathEndingCode
	tr.traverseImmutablePath(triePath, func(n *common.NodeData, trieKey []byte, ending PathEndingCode) {
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
	return ret, endingCode
}

func (tr *TrieReader) traverseImmutablePath(triePath []byte, fun func(n *common.NodeData, trieKey []byte, ending PathEndingCode)) {
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
			child, childTrieKey := tr.nodeStore.FetchChild(n, childIndex, trieKey)
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

func (tr *TrieUpdatable) traverseMutatedPath(triePath []byte, fun func(n *bufferedNode, ending PathEndingCode)) {
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
