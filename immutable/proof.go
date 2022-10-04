package immutable

import (
	"encoding/hex"
	"fmt"

	"github.com/iotaledger/trie.go/common"
)

// ProofGeneric represents a generic proof of inclusion or a maximal path in the trie which corresponds to the 'unpackedKey'
// The Ending indicates what represent the proof: it can be either 'proof of inclusion' of a unpackedKey/value terminal,
// or a reorg code, which means what operation on the trie must be performed in order to update the unpackedKey/value pair
type ProofGeneric struct {
	Key    []byte
	Path   []ProofGenericElement
	Ending ProofEndingCode
}

type ProofGenericElement struct {
	NodeData   *common.NodeData
	ChildIndex byte
}

type ProofEndingCode byte

const (
	EndingNone = ProofEndingCode(iota)
	EndingTerminal
	EndingSplit
	EndingExtend
)

func (e ProofEndingCode) String() string {
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

func (p *ProofGeneric) String() string {
	ret := fmt.Sprintf("GENERIC PROOF. Key: '%s', Ending: '%s'\n", string(p.Key), p.Ending)
	for i := range p.Path {
		ret += fmt.Sprintf("   #%d: pathFrag '%s', childIdx: %d\n", i,
			hex.EncodeToString(p.Path[i].NodeData.PathFragment), p.Path[i].ChildIndex)
	}
	return ret
}

// GetProofGeneric returns generic proof path. Contains references trie node cache.
// Should be immediately converted into the specific proof model independent of the trie
// Normally only called by the model
func GetProofGeneric(nodeStore *NodeStore, root common.VCommitment, triePath []byte) *ProofGeneric {
	p, ending := fetchPath(nodeStore, root, triePath)
	return &ProofGeneric{
		Key:    triePath,
		Path:   p,
		Ending: ending,
	}
}

// proofPath takes full unpackedKey as 'path' and collects the trie path up to the deepest possible node
// It returns:
// - path of keys which leads to 'finalKey'
// - common prefix between the last unpackedKey and the fragment
// - the 'endingCode' which indicates how it ends:
// -- EndingTerminal means 'finalKey' points to the node with non-nil terminal commitment, thus the path is a proof of inclusion
// -- EndingSplit means the 'finalKey' is a new unpackedKey, it does not point to any node and none of existing TrieReader are
//    prefix of the 'finalKey'. The trie must be reorged to include the new unpackedKey
// -- EndingExtend the path is a prefix of the 'finalKey', so trie must be extended to the same direction with new node
// - terminal of the last node
//func (tr *Trie) proofPath(unpackedKey []byte) ([]*bufferedNode, []byte, ProofEndingCode) {
//	n := tr.mutatedRoot
//
//	proof := make([]*bufferedNode, 0)
//	var trieKey []byte
//
//	for {
//		proof = append(proof, n)
//		Assert(len(trieKey) <= len(unpackedKey), "trie::proofPath assert: len(unpackedKey) <= len(unpackedKey), trieKey: '%s', unpackedKey: '%s'",
//			hex.EncodeToString(trieKey), hex.EncodeToString(unpackedKey))
//		if bytes.Equal(unpackedKey[len(trieKey):], n.pathFragment()) {
//			return proof, nil, EndingTerminal
//		}
//		prefix := commonPrefix(unpackedKey[len(trieKey):], n.pathFragment())
//
//		if len(prefix) < len(n.pathFragment()) {
//			return proof, prefix, EndingSplit
//		}
//		Assert(len(prefix) == len(n.pathFragment()), "trie::proofPath assert: len(prefix)==len(n.pathFragment), prefix: '%s', pathFragment: '%s'",
//			hex.EncodeToString(prefix), hex.EncodeToString(n.pathFragment()))
//		childIndexPosition := len(trieKey) + len(prefix)
//		Assert(childIndexPosition < len(unpackedKey), "childIndexPosition<len(unpackedKey)")
//
//		n = n.getChild(unpackedKey[childIndexPosition], tr.nodeStore)
//
//		if n == nil {
//			// if there is no commitment to the child at the position, it means trie must be extended at this point
//			return proof, prefix, EndingExtend
//		}
//	}
//}

// getLeafByKey goes along the path the same way proofPath, just does not produce the proof but instead returns last terminal, if found
func getLeafByKey(nodeStore *NodeStore, root common.VCommitment, triePath []byte) common.TCommitment {
	panic("implement me")
	//n, found := nodeStore.FetchNodeData(AsKey(mutatedRoot), nil)
	//if !found {
	//	return nil
	//}
	//
	//var trieKey []byte
	//for {
	//	Assert(len(trieKey) <= len(triePath), "trie::getLeafByKey assert: len(triePath) <= len(triePath), trieKey: '%s', triePath: '%s'",
	//		hex.EncodeToString(trieKey), hex.EncodeToString(triePath))
	//	if bytes.Equal(triePath[len(trieKey):], n.pathFragment) {
	//		return n.terminal // found trieKey
	//	}
	//	prefix := commonPrefix(triePath[len(trieKey):], n.pathFragment)
	//
	//	if len(prefix) < len(n.pathFragment) {
	//		return nil
	//	}
	//	Assert(len(prefix) == len(n.pathFragment), "trie::getLeafByKey assert: len(prefix)==len(n.pathFragment), prefix: '%s', pathFragment: '%s'",
	//		hex.EncodeToString(prefix), hex.EncodeToString(n.pathFragment))
	//	childIndexPosition := len(trieKey) + len(prefix)
	//	Assert(childIndexPosition < len(triePath), "childIndexPosition<len(triePath)")
	//
	//	n, trieKey = n.FetchChild(triePath[childIndexPosition], trieKey, nodeStore)
	//	if n == nil {
	//		return nil
	//	}
	//}
}

func fetchPath(nodeStore *NodeStore, root common.VCommitment, triePath []byte) ([]ProofGenericElement, ProofEndingCode) {
	panic("implement me")
	//n, found := nodeStore.FetchNodeData(AsKey(mutatedRoot), nil)
	//if !found {
	//	return nil, EndingRootNotFound
	//}
	//ret := make([]ProofGenericElement, 0)
	//var childIndex byte
	//var trieKey []byte
	//for {
	//	ret = append(ret, ProofGenericElement{
	//		NodeData:   n,
	//		ChildIndex: childIndex,
	//	})
	//
	//	Assert(len(trieKey) <= len(triePath), "trie::getLeafByKey assert: len(triePath) <= len(triePath), trieKey: '%s', triePath: '%s'",
	//		hex.EncodeToString(trieKey), hex.EncodeToString(triePath))
	//	if bytes.Equal(triePath[len(trieKey):], n.pathFragment) {
	//		return ret, EndingTerminal // found trieKey
	//	}
	//	prefix := commonPrefix(triePath[len(trieKey):], n.pathFragment)
	//
	//	if len(prefix) < len(n.pathFragment) {
	//		return ret, EndingSplit
	//	}
	//	Assert(len(prefix) == len(n.pathFragment), "trie::getLeafByKey assert: len(prefix)==len(n.pathFragment), prefix: '%s', pathFragment: '%s'",
	//		hex.EncodeToString(prefix), hex.EncodeToString(n.pathFragment))
	//	childIndexPosition := len(trieKey) + len(prefix)
	//	Assert(childIndexPosition < len(triePath), "childIndexPosition<len(triePath)")
	//
	//	childIndex = triePath[childIndexPosition]
	//	n, trieKey = n.FetchChild(childIndex, trieKey, nodeStore)
	//	if n == nil {
	//		return ret, EndingExtend
	//	}
	//}
}
