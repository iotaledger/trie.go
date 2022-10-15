package immutable

import (
	"bytes"

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

func (tr *TrieUpdatable) traverseMutatedPath(triePath []byte, fun func(n *bufferedNode, ending ProofEndingCode)) {
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
