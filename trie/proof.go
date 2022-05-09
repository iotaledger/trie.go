package trie

import (
	"bytes"
	"fmt"
)

// ProofGeneric represents a generic proof of inclusion or a maximal path in the trie which corresponds to the 'unpackedKey'
// The Ending indicates what represent the proof: it can be either 'proof of inclusion' of a unpackedKey/value Terminal,
// or a reorg code, which means what operation on the trie must be performed in order to update the unpackedKey/value pair
type ProofGeneric struct {
	Key    []byte
	Path   [][]byte
	Ending ProofEndingCode
}

type ProofEndingCode byte

const (
	EndingTerminal = iota
	EndingSplit
	EndingExtend
)

func (e ProofEndingCode) String() string {
	switch e {
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
	for i, k := range p.Path {
		ret += fmt.Sprintf("   #%d: %s\n", i, string(k))
	}
	return ret
}

// GetProofGeneric returns generic proof path. Contains references trie node cache.
// Should be immediately converted into the specific proof model independent of the trie
// Normally only called by the model
func GetProofGeneric(tr NodeStore, unpackedKey []byte) *ProofGeneric {
	p, _, ending := proofPath(tr, unpackedKey)
	return &ProofGeneric{
		Key:    unpackedKey,
		Path:   p,
		Ending: ending,
	}
}

// proofPath takes full unpackedKey as 'path' and collects the trie path up to the deepest possible node
// It returns:
// - path of keys which leads to 'finalKey'
// - common prefix between the last unpackedKey and the fragment
// - the 'endingCode' which indicates how it ends:
// -- EndingTerminal means 'finalKey' points to the node with non-nil Terminal commitment, thus the path is a proof of inclusion
// -- EndingSplit means the 'finalKey' is a new unpackedKey, it does not point to any node and none of existing TrieReader are
//    prefix of the 'finalKey'. The trie must be reorged to include the new unpackedKey
// -- EndingExtend the path is a prefix of the 'finalKey', so trie must be extended to the same direction with new node
func proofPath(trieAccess NodeStore, unpackedKey []byte) ([][]byte, []byte, ProofEndingCode) {
	n, ok := trieAccess.GetNode(nil)
	if !ok {
		return nil, nil, 0
	}

	proof := make([][]byte, 0)
	var key []byte

	for {
		proof = append(proof, key)
		Assert(len(key) <= len(unpackedKey), "len(unpackedKey) <= len(unpackedKey)")
		if bytes.Equal(unpackedKey[len(key):], n.PathFragment()) {
			return proof, nil, EndingTerminal
		}
		prefix := commonPrefix(unpackedKey[len(key):], n.PathFragment())

		if len(prefix) < len(n.PathFragment()) {
			return proof, prefix, EndingSplit
		}
		Assert(len(prefix) == len(n.PathFragment()), "len(prefix)==len(n.PathFragment)")
		childIndexPosition := len(key) + len(prefix)
		Assert(childIndexPosition < len(unpackedKey), "childIndexPosition<len(unpackedKey)")

		key = childKey(n, unpackedKey[childIndexPosition])

		n, ok = trieAccess.GetNode(key)
		if !ok {
			// if there are no commitment to the child at the position, it means trie must be extended at this point
			return proof, prefix, EndingExtend
		}
	}
}

func commonPrefix(b1, b2 []byte) []byte {
	ret := make([]byte, 0)
	for i := 0; i < len(b1) && i < len(b2); i++ {
		if b1[i] != b2[i] {
			break
		}
		ret = append(ret, b1[i])
	}
	return ret
}
