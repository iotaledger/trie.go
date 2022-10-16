package trie_blake2b_verify

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/iotaledger/trie.go/models/trie_blake2b"
	"golang.org/x/xerrors"
)

// MustKeyWithTerminal returns key and terminal commitment the proof is about. It returns:
// - key
// - terminal commitment. If it is nil, the proof is a proof of absence
// It does not verify the proof, so this function should be used only after Validate()
func MustKeyWithTerminal(p *trie_blake2b.Proof) ([]byte, []byte) {
	if len(p.Path) == 0 {
		return nil, nil
	}
	lastElem := p.Path[len(p.Path)-1]
	switch {
	case p.PathArity.IsChildIndex(lastElem.ChildIndex):
		if _, ok := lastElem.Children[byte(lastElem.ChildIndex)]; ok {
			panic("nil child commitment expected for proof of absence")
		}
		return p.Key, nil
	case lastElem.ChildIndex == p.PathArity.TerminalCommitmentIndex():
		if lastElem.Terminal == nil {
			return p.Key, nil
		}
		return p.Key, lastElem.Terminal
	case lastElem.ChildIndex == p.PathArity.PathFragmentCommitmentIndex():
		return p.Key, nil
	}
	panic("wrong lastElem.ChildIndex")
}

// IsProofOfAbsence checks if it is proof of absence. Proof that the trie commits to something else in the place
// where it would commit to the key if it would be present
func IsProofOfAbsence(p *trie_blake2b.Proof) bool {
	_, r := MustKeyWithTerminal(p)
	return r == nil
}

// Validate check the proof against the provided root commitments
func Validate(p *trie_blake2b.Proof, rootBytes []byte) error {
	if len(p.Path) == 0 {
		if len(rootBytes) != 0 {
			return xerrors.New("proof is empty")
		}
		return nil
	}
	c, err := verify(p, 0, 0)
	if err != nil {
		return err
	}
	if !bytes.Equal(c, rootBytes) {
		return xerrors.New("invalid proof: commitment not equal to the root")
	}
	return nil
}

// ValidateWithTerminal checks the proof and checks if the proof commits to the specific value
// The check is dependent on the commitment model because of valueOptimisationThreshold
func ValidateWithTerminal(p *trie_blake2b.Proof, rootBytes, terminalBytes []byte) error {
	if err := Validate(p, rootBytes); err != nil {
		return err
	}
	_, terminalBytesInProof := MustKeyWithTerminal(p)
	if !bytes.Equal(terminalBytes, terminalBytesInProof) {
		return errors.New("key does not correspond to the given value commitment")
	}
	return nil
}

func verify(p *trie_blake2b.Proof, pathIdx, keyIdx int) ([]byte, error) {
	common.Assert(pathIdx < len(p.Path), "assertion: pathIdx < lenPlus1(p.Path)")
	common.Assert(keyIdx <= len(p.Key), "assertion: keyIdx <= lenPlus1(p.Key)")

	elem := p.Path[pathIdx]
	tail := p.Key[keyIdx:]
	isPrefix := bytes.HasPrefix(tail, elem.PathFragment)
	last := pathIdx == len(p.Path)-1
	if !last && !isPrefix {
		return nil, fmt.Errorf("wrong proof: proof path does not follow the key. Path position: %d, key position %d", pathIdx, keyIdx)
	}
	if !last {
		common.Assert(isPrefix, "assertion: isPrefix")
		if !p.PathArity.IsChildIndex(elem.ChildIndex) {
			return nil, fmt.Errorf("wrong proof: wrong child index. Path position: %d, key position %d", pathIdx, keyIdx)
		}
		if _, ok := elem.Children[byte(elem.ChildIndex)]; ok {
			return nil, fmt.Errorf("wrong proof: unexpected commitment at child index %d. Path position: %d, key position %d", elem.ChildIndex, pathIdx, keyIdx)
		}
		nextKeyIdx := keyIdx + len(elem.PathFragment) + 1
		if nextKeyIdx > len(p.Key) {
			return nil, fmt.Errorf("wrong proof: proof path out of key bounds. Path position: %d, key position %d", pathIdx, keyIdx)
		}
		c, err := verify(p, pathIdx+1, nextKeyIdx)
		if err != nil {
			return nil, err
		}
		return hashIt(elem, c, p.PathArity, p.HashSize), nil
	}
	// it is the last in the path
	if p.PathArity.IsChildIndex(elem.ChildIndex) {
		c := elem.Children[byte(elem.ChildIndex)]
		if c != nil {
			return nil, fmt.Errorf("wrong proof: child commitment of the last element expected to be nil. Path position: %d, key position %d", pathIdx, keyIdx)
		}
		return hashIt(elem, nil, p.PathArity, p.HashSize), nil
	}
	if elem.ChildIndex != p.PathArity.TerminalCommitmentIndex() && elem.ChildIndex != p.PathArity.PathFragmentCommitmentIndex() {
		return nil, fmt.Errorf("wrong proof: child index expected to be %d or %d. Path position: %d, key position %d",
			p.PathArity.TerminalCommitmentIndex(), p.PathArity.PathFragmentCommitmentIndex(), pathIdx, keyIdx)
	}
	return hashIt(elem, nil, p.PathArity, p.HashSize), nil
}

func makeHashVector(e *trie_blake2b.ProofElement, missingCommitment []byte, arity common.PathArity, sz trie_blake2b.HashSize) [][]byte {
	hashes := make([][]byte, arity.VectorLength())
	for idx, c := range e.Children {
		common.Assert(arity.IsChildIndex(int(idx)), "arity.IsChildIndex(int(idx)")
		hashes[idx] = c
	}
	if len(e.Terminal) > 0 {
		hashes[arity.TerminalCommitmentIndex()] = e.Terminal
	}
	rawBytes, _ := trie_blake2b.CommitToDataRaw(e.PathFragment, sz)
	hashes[arity.PathFragmentCommitmentIndex()] = rawBytes
	if arity.IsChildIndex(e.ChildIndex) {
		hashes[e.ChildIndex] = missingCommitment
	}
	return hashes
}

func hashIt(e *trie_blake2b.ProofElement, missingCommitment []byte, arity common.PathArity, sz trie_blake2b.HashSize) []byte {
	return trie_blake2b.HashTheVector(makeHashVector(e, missingCommitment, arity, sz), arity, sz)
}
