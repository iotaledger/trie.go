// Package trie_blake2b_verify contains functions for verification of the proofs of inclusion or absence
// in the trie with trie_blake2b commitment model. The package only depends on the commitment model
// implementation and the proof format it defines. The verification package is completely independent on
// the implementation of the Merkle tree (the trie)
//
// DISCLAIMER: THE FOLLOWING CODE IS SECURITY CRITICAL.
// ANY POTENTIAL BUG WHICH MAY LEAD TO FALSE POSITIVES OF PROOF VALIDITY CHECKS POTENTIALLY
// CREATES AN ATTACK VECTOR.
// THEREFORE, IT IS HIGHLY RECOMMENDED THE VERIFICATION CODE TO BE WRITTEN BY THE VERIFYING PARTY ITSELF,
// INSTEAD OF CLONING THIS PACKAGE. DO NOT TRUST ANYBODY BUT YOURSELF. IN ANY CASE, PERFORM A DETAILED
// AUDIT OF THE PROOF-VERIFYING CODE BEFORE USING IT
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
func MustKeyWithTerminal(p *trie_blake2b.MerkleProof) ([]byte, []byte) {
	if len(p.Path) == 0 {
		return nil, nil
	}
	lastElem := p.Path[len(p.Path)-1]
	switch {
	case p.PathArity.IsValidChildIndex(lastElem.ChildIndex):
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

// IsProofOfAbsence checks if it is proof of absence. MerkleProof that the trie commits to something else in the place
// where it would commit to the key if it would be present
func IsProofOfAbsence(p *trie_blake2b.MerkleProof) bool {
	_, r := MustKeyWithTerminal(p)
	return len(r) == 0
}

// Validate check the proof against the provided root commitments
func Validate(p *trie_blake2b.MerkleProof, rootBytes []byte) error {
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
func ValidateWithTerminal(p *trie_blake2b.MerkleProof, rootBytes, terminalBytes []byte) error {
	if err := Validate(p, rootBytes); err != nil {
		return err
	}
	_, terminalBytesInProof := MustKeyWithTerminal(p)
	compressedTerm, _ := trie_blake2b.CompressToHashSize(terminalBytes, p.HashSize)
	if !bytes.Equal(compressedTerm, terminalBytesInProof) {
		return errors.New("key does not correspond to the given value commitment")
	}
	return nil
}

func verify(p *trie_blake2b.MerkleProof, pathIdx, keyIdx int) ([]byte, error) {
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
		if !p.PathArity.IsValidChildIndex(elem.ChildIndex) {
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
		return hashProofElement(elem, c, p.PathArity, p.HashSize)
	}
	// it is the last in the path
	if p.PathArity.IsValidChildIndex(elem.ChildIndex) {
		c := elem.Children[byte(elem.ChildIndex)]
		if c != nil {
			return nil, fmt.Errorf("wrong proof: child commitment of the last element expected to be nil. Path position: %d, key position %d", pathIdx, keyIdx)
		}
		return hashProofElement(elem, nil, p.PathArity, p.HashSize)
	}
	if elem.ChildIndex != p.PathArity.TerminalCommitmentIndex() && elem.ChildIndex != p.PathArity.PathFragmentCommitmentIndex() {
		return nil, fmt.Errorf("wrong proof: child index expected to be %d or %d. Path position: %d, key position %d",
			p.PathArity.TerminalCommitmentIndex(), p.PathArity.PathFragmentCommitmentIndex(), pathIdx, keyIdx)
	}
	return hashProofElement(elem, nil, p.PathArity, p.HashSize)
}

const errTooLongCommitment = "too long commitment at position %d. Can't be longer than %d bytes"

func makeHashVector(e *trie_blake2b.MerkleProofElement, missingCommitment []byte, arity common.PathArity, sz trie_blake2b.HashSize) ([][]byte, error) {
	hashes := make([][]byte, arity.VectorLength())
	for idx, c := range e.Children {
		if !arity.IsValidChildIndex(int(idx)) {
			return nil, fmt.Errorf("wrong child index %d", idx)
		}
		if len(c) > int(sz) {
			return nil, fmt.Errorf(errTooLongCommitment, idx, int(sz))
		}
		hashes[idx] = c
	}
	if len(e.Terminal) > 0 {
		if len(e.Terminal) > int(sz) {
			return nil, fmt.Errorf(errTooLongCommitment+" (terminal)", arity.TerminalCommitmentIndex(), int(sz))
		}
		hashes[arity.TerminalCommitmentIndex()] = e.Terminal
	}
	rawBytes, _ := trie_blake2b.CompressToHashSize(e.PathFragment, sz)
	hashes[arity.PathFragmentCommitmentIndex()] = rawBytes
	if arity.IsValidChildIndex(e.ChildIndex) {
		if len(missingCommitment) > int(sz) {
			return nil, fmt.Errorf(errTooLongCommitment+" (skipped commitment)", e.ChildIndex, int(sz))
		}
		hashes[e.ChildIndex] = missingCommitment
	}
	return hashes, nil
}

func hashProofElement(e *trie_blake2b.MerkleProofElement, missingCommitment []byte, arity common.PathArity, sz trie_blake2b.HashSize) ([]byte, error) {
	hashVector, err := makeHashVector(e, missingCommitment, arity, sz)
	if err != nil {
		return nil, err
	}
	return trie_blake2b.HashTheVector(hashVector, arity, sz), nil
}
