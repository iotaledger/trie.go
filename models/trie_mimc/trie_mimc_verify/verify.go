package trie_mimc_verify

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/iotaledger/trie.go/models/trie_mimc"
	"github.com/iotaledger/trie.go/trie"
	"golang.org/x/xerrors"
)

// MustKeyWithTerminal returns key and terminal commitment the proof is about. It returns:
// - key
// - commitment slice of up to hashSize bytes long. If it is nil, the proof is a proof of absence
// It does not verify the proof, so this function should be used only after Validate()
func MustKeyWithTerminal(p *trie_mimc.Proof) ([]byte, []byte) {
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
func IsProofOfAbsence(p *trie_mimc.Proof) bool {
	_, r := MustKeyWithTerminal(p)
	return r == nil
}

// Validate check the proof against the provided root commitments
func Validate(p *trie_mimc.Proof, rootBytes []byte) error {
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

func ValidateTest() {
	inputs := []byte{42, 194, 10, 96, 133, 113, 88, 31, 86, 136, 60, 65, 11, 106, 226, 218, 169, 220, 186, 36, 114, 230, 53, 147, 171, 202, 12, 106, 45, 89, 231, 132, 0, 41, 110, 104, 212, 140, 165, 49, 87, 44, 32, 181, 221, 58, 86, 232, 118, 168, 2, 169, 90, 13, 92, 128, 54, 250, 117, 253, 216, 155, 33, 211, 90, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	outputs := []byte{8, 205, 235, 168, 70, 155, 72, 62, 195, 30, 12, 199, 164, 235, 36, 86, 198, 201, 78, 155, 86, 119, 14, 124, 221, 225, 228, 148, 151, 171, 196, 35}

	results := trie_mimc.MIMCIt(inputs, 32)
	fmt.Println("ValidateTest(): Output =", outputs)
	fmt.Println("ValidateTest(): Results =", results)
}

// ValidateWithValue checks the proof and checks if the proof commits to the specific value
func ValidateWithValue(p *trie_mimc.Proof, rootBytes []byte, value []byte) error {
	if err := Validate(p, rootBytes); err != nil {
		return err
	}
	_, r := MustKeyWithTerminal(p)
	if len(r) == 0 {
		return errors.New("key is not present in the state")
	}
	if !bytes.Equal(trie_mimc.CommitToDataRaw(value, p.HashSize), r) {
		return errors.New("key does not correspond to the given value")
	}
	return nil
}

// CommitmentToTheTerminalNode returns hash of the last node in the proof
// If it is a valid proof, it s always contains terminal commitment
// It is useful to get commitment to the sub-state. It must contain some value
// at its nil postfix
func CommitmentToTheTerminalNode(p *trie_mimc.Proof) []byte {
	if len(p.Path) == 0 {
		return nil
	}
	return hashIt(p.Path[len(p.Path)-1], nil, p.PathArity, p.HashSize)
}

func verify(p *trie_mimc.Proof, pathIdx, keyIdx int) ([]byte, error) {
	trie.Assert(pathIdx < len(p.Path), "assertion: pathIdx < lenPlus1(p.Path)")
	trie.Assert(keyIdx <= len(p.Key), "assertion: keyIdx <= lenPlus1(p.Key)")

	elem := p.Path[pathIdx]
	tail := p.Key[keyIdx:]
	isPrefix := bytes.HasPrefix(tail, elem.PathFragment)
	last := pathIdx == len(p.Path)-1
	if !last && !isPrefix {
		return nil, fmt.Errorf("wrong proof: proof path does not follow the key. Path position: %d, key position %d", pathIdx, keyIdx)
	}
	if !last {
		trie.Assert(isPrefix, "assertion: isPrefix")
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
		fmt.Println("trie_mimc_verify: Current Sum =", c)
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

func makeHashVector(e *trie_mimc.ProofElement, missingCommitment []byte, arity trie.PathArity, sz trie_mimc.HashSize) [][]byte {
	hashes := make([][]byte, arity.VectorLength())
	for idx, c := range e.Children {
		trie.Assert(arity.IsChildIndex(int(idx)), "arity.IsChildIndex(int(idx)")
		hashes[idx] = c
	}
	if len(e.Terminal) > 0 {
		hashes[arity.TerminalCommitmentIndex()] = e.Terminal
	}
	hashes[arity.PathFragmentCommitmentIndex()] = trie_mimc.CommitToDataRaw(e.PathFragment, sz)
	if arity.IsChildIndex(e.ChildIndex) {
		hashes[e.ChildIndex] = missingCommitment
	}
	fmt.Println("makeHashVector")
	for idx, h := range hashes {
		if len(hashes[idx]) == 0 {
			fmt.Print(h, " ")
		} else {
			fmt.Print(h, " ")
		}
	}
	fmt.Println("")
	// fmt.Println(hashes[0], hashes[1], binary.BigEndian.Uint64(hashes[2]), hashes[3])
	return hashes
}

func hashIt(e *trie_mimc.ProofElement, missingCommitment []byte, arity trie.PathArity, sz trie_mimc.HashSize) []byte {
	return trie_mimc.HashTheVector(makeHashVector(e, missingCommitment, arity, sz), arity, sz)
}
