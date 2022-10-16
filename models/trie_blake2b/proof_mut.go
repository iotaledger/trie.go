package trie_blake2b

import (
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/iotaledger/trie.go/mutable"
)

// ProofMut converts generic proof path of the mutable trie implementation to the Merkle proof path
func (m *CommitmentModel) ProofMut(key []byte, tr mutable.NodeStore) *Proof {
	unpackedKey := common.UnpackBytes(key, tr.PathArity())
	proofGeneric := mutable.GetProofGeneric(tr, unpackedKey)
	if proofGeneric == nil {
		return nil
	}
	ret := &Proof{
		PathArity: tr.PathArity(),
		HashSize:  m.hashSize,
		Key:       proofGeneric.Key,
		Path:      make([]*ProofElement, len(proofGeneric.Path)),
	}
	var elemKeyPosition int
	var isLast bool
	var childIndex int

	for i, k := range proofGeneric.Path {
		node, ok := tr.GetNode(k)
		if !ok {
			panic(fmt.Errorf("can't find node key '%x'", k))
		}
		isLast = i == len(proofGeneric.Path)-1
		if !isLast {
			elemKeyPosition += len(node.PathFragment())
			childIndex = int(unpackedKey[elemKeyPosition])
			elemKeyPosition++
		} else {
			switch proofGeneric.Ending {
			case common.EndingTerminal:
				childIndex = m.arity.TerminalCommitmentIndex()
			case common.EndingExtend, common.EndingSplit:
				childIndex = m.arity.PathFragmentCommitmentIndex()
			default:
				panic("wrong ending code")
			}
		}
		em := &ProofElement{
			PathFragment: node.PathFragment(),
			Children:     make(map[byte][]byte),
			Terminal:     nil,
			ChildIndex:   childIndex,
		}
		if node.Terminal() != nil {
			em.Terminal = node.Terminal().Bytes()
			// (*terminalCommitment).bytes
		}
		for idx, v := range node.ChildCommitments() {
			if int(idx) == childIndex {
				// skipping the commitment which must come from the next child
				continue
			}
			em.Children[idx] = v.(vectorCommitment)
		}
		ret.Path[i] = em
	}
	return ret
}
