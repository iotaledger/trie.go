package trie_blake2b

// TODO make Merkle proofs independent from trie types

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/iotaledger/trie.go/trie"
	"golang.org/x/xerrors"
	"io"
)

// Proof blake2b 20 byte model-specific proof of inclusion
type Proof struct {
	PathArity trie.PathArity
	HashSize  HashSize
	Key       []byte
	Path      []*ProofElement
}

type ProofElement struct {
	PathFragment []byte
	Children     map[byte]vectorCommitment
	Terminal     *terminalCommitment
	ChildIndex   int
}

func ProofFromBytes(data []byte) (*Proof, error) {
	ret := &Proof{}
	if err := ret.Read(bytes.NewReader(data)); err != nil {
		return nil, err
	}
	return ret, nil
}

// Proof converts generic proof path to the Merkle proof path
func (m *CommitmentModel) Proof(key []byte, tr trie.NodeStore) *Proof {
	unpackedKey := trie.UnpackBytes(key, tr.PathArity())
	proofGeneric := trie.GetProofGeneric(tr, unpackedKey)
	if proofGeneric == nil {
		return nil
	}
	ret := &Proof{
		PathArity: tr.PathArity(),
		HashSize:  m.HashSize,
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
			case trie.EndingTerminal:
				childIndex = m.arity.TerminalCommitmentIndex()
			case trie.EndingExtend, trie.EndingSplit:
				childIndex = m.arity.PathFragmentCommitmentIndex()
			default:
				panic("wrong ending code")
			}
		}
		em := &ProofElement{
			PathFragment: node.PathFragment(),
			Children:     make(map[byte]vectorCommitment),
			Terminal:     nil,
			ChildIndex:   childIndex,
		}
		if node.Terminal() != nil {
			em.Terminal = node.Terminal().(*terminalCommitment)
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

func (p *Proof) Bytes() []byte {
	return trie.MustBytes(p)
}

// MustKeyWithTerminal returns key and terminal commitment the proof is about. It returns:
// - key
// - commitment slice of up to hashSize bytes long. If it is nil, the proof is a proof of absence
// It does not verify the proof, so this function should be used only after Validate()
func (p *Proof) MustKeyWithTerminal() ([]byte, []byte) {
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
		return p.Key, lastElem.Terminal.bytes
	case lastElem.ChildIndex == p.PathArity.PathFragmentCommitmentIndex():
		return p.Key, nil
	}
	panic("wrong lastElem.ChildIndex")
}

// IsProofOfAbsence checks if it is proof of absence. Proof that the trie commits to something else in the place
// where it would commit to the key if it would be present
func (p *Proof) IsProofOfAbsence() bool {
	_, r := p.MustKeyWithTerminal()
	return r == nil
}

// Validate check the proof against the provided root commitments
func (p *Proof) Validate(root trie.VCommitment) error {
	if len(p.Path) == 0 {
		if root != nil {
			return xerrors.New("proof is empty")
		}
		return nil
	}
	c, err := p.verify(0, 0)
	if err != nil {
		return err
	}
	if !equalCommitments(vectorCommitment(c), root) {
		return xerrors.New("invalid proof: commitment not equal to the root")
	}
	return nil
}

// CommitmentToTheTerminalNode returns hash of the last node in the proof
// If it is a valid proof, it s always contains terminal commitment
// It is useful to get commitment to the sub-state. It must contain some value
// at its nil postfix
func (p *Proof) CommitmentToTheTerminalNode() trie.VCommitment {
	if len(p.Path) == 0 {
		return nil
	}
	ret := p.Path[len(p.Path)-1].hashIt(nil, p.PathArity, p.HashSize)
	return (*vectorCommitment)(&ret)
}

func (p *Proof) verify(pathIdx, keyIdx int) ([]byte, error) {
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
		c, err := p.verify(pathIdx+1, nextKeyIdx)
		if err != nil {
			return nil, err
		}
		return elem.hashIt(c, p.PathArity, p.HashSize), nil
	}
	// it is the last in the path
	if p.PathArity.IsChildIndex(elem.ChildIndex) {
		c := elem.Children[byte(elem.ChildIndex)]
		if c != nil {
			return nil, fmt.Errorf("wrong proof: child commitment of the last element expected to be nil. Path position: %d, key position %d", pathIdx, keyIdx)
		}
		return elem.hashIt(nil, p.PathArity, p.HashSize), nil
	}
	if elem.ChildIndex != p.PathArity.TerminalCommitmentIndex() && elem.ChildIndex != p.PathArity.PathFragmentCommitmentIndex() {
		return nil, fmt.Errorf("wrong proof: child index expected to be %d or %d. Path position: %d, key position %d",
			p.PathArity.TerminalCommitmentIndex(), p.PathArity.PathFragmentCommitmentIndex(), pathIdx, keyIdx)
	}
	return elem.hashIt(nil, p.PathArity, p.HashSize), nil
}

func (e *ProofElement) makeVector(missingCommitment []byte, arity trie.PathArity, sz HashSize) [][]byte {
	hashes := make([][]byte, arity.VectorLength())
	for idx, c := range e.Children {
		trie.Assert(arity.IsChildIndex(int(idx)), "arity.IsChildIndex(int(idx)")
		hashes[idx] = c
	}
	if e.Terminal != nil {
		hashes[arity.TerminalCommitmentIndex()] = e.Terminal.bytesEssence()
	}
	hashes[arity.PathFragmentCommitmentIndex()] = commitToDataRaw(e.PathFragment, sz).bytesEssence()
	if arity.IsChildIndex(e.ChildIndex) {
		hashes[e.ChildIndex] = missingCommitment
	}
	return hashes
}

func (e *ProofElement) hashIt(missingCommitment []byte, arity trie.PathArity, sz HashSize) []byte {
	return hashTheVector(e.makeVector(missingCommitment, arity, sz), arity, sz)
}

func (p *Proof) Write(w io.Writer) error {
	var err error
	if err = trie.WriteByte(w, byte(p.PathArity)); err != nil {
		return err
	}
	if err = trie.WriteByte(w, byte(p.HashSize)); err != nil {
		return err
	}
	encodedKey, err := trie.EncodeUnpackedBytes(p.Key, p.PathArity)
	if err != nil {
		return err
	}
	if err = trie.WriteBytes16(w, encodedKey); err != nil {
		return err
	}
	if err = trie.WriteUint16(w, uint16(len(p.Path))); err != nil {
		return err
	}
	for _, e := range p.Path {
		if err = e.Write(w, p.PathArity); err != nil {
			return err
		}
	}
	return nil
}

func (p *Proof) Read(r io.Reader) error {
	b, err := trie.ReadByte(r)
	if err != nil {
		return err
	}
	p.PathArity = trie.PathArity(b)

	b, err = trie.ReadByte(r)
	if err != nil {
		return err
	}
	p.HashSize = HashSize(b)
	if p.HashSize != HashSize256 && p.HashSize != HashSize160 {
		return errors.New("wrong hash size")
	}

	var encodedKey []byte
	if encodedKey, err = trie.ReadBytes16(r); err != nil {
		return err
	}
	if p.Key, err = trie.DecodeToUnpackedBytes(encodedKey, p.PathArity); err != nil {
		return err
	}
	var size uint16
	if err = trie.ReadUint16(r, &size); err != nil {
		return err
	}
	p.Path = make([]*ProofElement, size)
	for i := range p.Path {
		p.Path[i] = &ProofElement{}
		if err = p.Path[i].Read(r, p.PathArity, p.HashSize); err != nil {
			return err
		}
	}
	return nil
}

const (
	hasTerminalValueFlag = 0x01
	hasChildrenFlag      = 0x02
)

func (e *ProofElement) Write(w io.Writer, arity trie.PathArity) error {
	encodedPathFragment, err := trie.EncodeUnpackedBytes(e.PathFragment, arity)
	if err != nil {
		return err
	}
	if err := trie.WriteBytes16(w, encodedPathFragment); err != nil {
		return err
	}
	if err := trie.WriteUint16(w, uint16(e.ChildIndex)); err != nil {
		return err
	}
	var smallFlags byte
	if e.Terminal != nil {
		smallFlags = hasTerminalValueFlag
	}
	// compress children flags 32 bytes (if any)
	var flags [32]byte
	for i := range e.Children {
		flags[i/8] |= 0x1 << (i % 8)
		smallFlags |= hasChildrenFlag
	}
	if err := trie.WriteByte(w, smallFlags); err != nil {
		return err
	}
	// write terminal commitment if any
	if smallFlags&hasTerminalValueFlag != 0 {
		if err := e.Terminal.Write(w); err != nil {
			return err
		}
	}
	// write child commitments if any
	if smallFlags&hasChildrenFlag != 0 {
		if _, err := w.Write(flags[:]); err != nil {
			return err
		}
		for i := 0; i < arity.VectorLength(); i++ {
			child, ok := e.Children[uint8(i)]
			if !ok {
				continue
			}
			if err := child.Write(w); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *ProofElement) Read(r io.Reader, arity trie.PathArity, sz HashSize) error {
	var err error
	var encodedPathFragment []byte
	if encodedPathFragment, err = trie.ReadBytes16(r); err != nil {
		return err
	}
	if e.PathFragment, err = trie.DecodeToUnpackedBytes(encodedPathFragment, arity); err != nil {
		return err
	}
	var idx uint16
	if err := trie.ReadUint16(r, &idx); err != nil {
		return err
	}
	e.ChildIndex = int(idx)
	var smallFlags byte
	if smallFlags, err = trie.ReadByte(r); err != nil {
		return err
	}
	if smallFlags&hasTerminalValueFlag != 0 {
		e.Terminal = newTerminalCommitment(sz)
		if err := e.Terminal.Read(r); err != nil {
			return err
		}
	} else {
		e.Terminal = nil
	}
	e.Children = make(map[byte]vectorCommitment)
	if smallFlags&hasChildrenFlag != 0 {
		var flags [32]byte
		if _, err := r.Read(flags[:]); err != nil {
			return err
		}
		for i := 0; i < arity.NumChildren(); i++ {
			ib := uint8(i)
			if flags[i/8]&(0x1<<(i%8)) != 0 {
				e.Children[ib] = newVectorCommitment(sz)
				if err := e.Children[ib].Read(r); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
