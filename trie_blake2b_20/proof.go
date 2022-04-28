package trie_blake2b_20

import (
	"bytes"
	"fmt"
	trie_go "github.com/iotaledger/trie.go"
	"github.com/iotaledger/trie.go/trie256p"
	"golang.org/x/xerrors"
	"io"
)

// blake2b 32 byte model-specific proof of inclusion
type Proof struct {
	Key  []byte
	Path []*ProofElement
}

type ProofElement struct {
	PathFragment []byte
	Children     map[byte]*vectorCommitment
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
func (m *CommitmentModel) Proof(key []byte, tr trie256p.NodeStore) *Proof {
	proofGeneric := trie256p.GetProofGeneric(tr, key)
	if proofGeneric == nil {
		return nil
	}
	ret := &Proof{
		Key:  proofGeneric.Key,
		Path: make([]*ProofElement, len(proofGeneric.Path)),
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
			childIndex = int(key[elemKeyPosition])
			elemKeyPosition++
		} else {
			switch proofGeneric.Ending {
			case trie256p.EndingTerminal:
				childIndex = 256
			case trie256p.EndingExtend, trie256p.EndingSplit:
				childIndex = 257
			default:
				panic("wrong ending code")
			}
		}
		em := &ProofElement{
			PathFragment: node.PathFragment(),
			Children:     make(map[byte]*vectorCommitment),
			Terminal:     nil,
			ChildIndex:   childIndex,
		}
		if node.Terminal() != nil {
			em.Terminal = node.Terminal().(*terminalCommitment)
		}
		for k, v := range node.ChildCommitments() {
			if int(k) == childIndex {
				// skipping the commitment which must come from the next child
				continue
			}
			em.Children[k] = v.(*vectorCommitment)
		}
		ret.Path[i] = em
	}
	return ret
}

func (p *Proof) Bytes() []byte {
	return trie_go.MustBytes(p)
}

// MustKeyWithTerminal returns key and terminal commitment the proof is about. It returns:
// - key
// - commitment slice of up to hashSize bytes long. If it is nil, the proof is a proof of absence
// - false if it is original data, true if it is a blake2b hash of the data
// It does not verify the proof, so this function should be used only after Validate()
func (p *Proof) MustKeyWithTerminal() ([]byte, []byte, bool) {
	if len(p.Path) == 0 {
		return nil, nil, false
	}
	lastElem := p.Path[len(p.Path)-1]
	switch {
	case lastElem.ChildIndex < 256:
		if _, ok := lastElem.Children[byte(lastElem.ChildIndex)]; ok {
			panic("nil child commitment expected for proof of absence")
		}
		return p.Key, nil, false
	case lastElem.ChildIndex == 256:
		if lastElem.Terminal == nil {
			return p.Key, nil, false
		}
		d, ishash := lastElem.Terminal.value()
		return p.Key, d, ishash
	case lastElem.ChildIndex == 257:
		return p.Key, nil, false
	}
	panic("wrong lastElem.ChildIndex")
}

// IsProofOfAbsence checks if it is proof of absense. Proof that the trie commits to something else in the place
// where it would commit to the key if it would be present
func (p *Proof) IsProofOfAbsence() bool {
	_, r, _ := p.MustKeyWithTerminal()
	return r == nil
}

// Validate check the proof agains the provided root commitments
// if 'value' is specified, checks if commitment to that value is the terminal of the last element in path
func (p *Proof) Validate(root trie_go.VCommitment, value ...[]byte) error {
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
	cv := (vectorCommitment)(c)
	if !trie_go.EqualCommitments(&cv, root) {
		return xerrors.New("invalid proof: commitment not equal to the root")
	}
	if len(value) > 0 {
		tc := p.Path[len(p.Path)-1].Terminal
		tc1 := commitToTerminal(value[0])
		if !trie_go.EqualCommitments(tc1, tc) {
			return xerrors.New("invalid proof: terminal commitment and terminal proof are not equal")
		}
	}
	return nil
}

// CommitmentToTheTerminalNode returns hash of the last node in the proof
// If it is a valid proof, it s always contains terminal commitment
// It is useful to get commitment to the sub-state. It must contain some value
// at its nil postfix
func (p *Proof) CommitmentToTheTerminalNode() trie_go.VCommitment {
	if len(p.Path) == 0 {
		return nil
	}
	ret := p.Path[len(p.Path)-1].hashIt(nil)
	return (*vectorCommitment)(&ret)
}

func (p *Proof) verify(pathIdx, keyIdx int) ([hashSize]byte, error) {
	trie_go.Assert(pathIdx < len(p.Path), "assertion: pathIdx < lenPlus1(p.Path)")
	trie_go.Assert(keyIdx <= len(p.Key), "assertion: keyIdx <= lenPlus1(p.Key)")

	elem := p.Path[pathIdx]
	tail := p.Key[keyIdx:]
	isPrefix := bytes.HasPrefix(tail, elem.PathFragment)
	last := pathIdx == len(p.Path)-1
	if !last && !isPrefix {
		return [hashSize]byte{}, fmt.Errorf("wrong proof: proof path does not follow the key. Path position: %d, key position %d", pathIdx, keyIdx)
	}
	if !last {
		trie_go.Assert(isPrefix, "assertion: isPrefix")
		if elem.ChildIndex > 255 {
			return [hashSize]byte{}, fmt.Errorf("wrong proof: wrong child index. Path position: %d, key position %d", pathIdx, keyIdx)
		}
		if _, ok := elem.Children[byte(elem.ChildIndex)]; ok {
			return [hashSize]byte{}, fmt.Errorf("wrong proof: unexpected commitment at child index %d. Path position: %d, key position %d", elem.ChildIndex, pathIdx, keyIdx)
		}
		nextKeyIdx := keyIdx + len(elem.PathFragment) + 1
		if nextKeyIdx > len(p.Key) {
			return [hashSize]byte{}, fmt.Errorf("wrong proof: proof path out of key bounds. Path position: %d, key position %d", pathIdx, keyIdx)
		}
		c, err := p.verify(pathIdx+1, nextKeyIdx)
		if err != nil {
			return [hashSize]byte{}, err
		}
		return elem.hashIt(&c), nil
	}
	// it is the last in the path
	if elem.ChildIndex < 256 {
		c := elem.Children[byte(elem.ChildIndex)]
		if c != nil {
			return [hashSize]byte{}, fmt.Errorf("wrong proof: child commitment of the last element expected to be nil. Path position: %d, key position %d", pathIdx, keyIdx)
		}
		return elem.hashIt(nil), nil
	}
	if elem.ChildIndex != 256 && elem.ChildIndex != 257 {
		return [hashSize]byte{}, fmt.Errorf("wrong proof: child index expected to be 256 or 257. Path position: %d, key position %d", pathIdx, keyIdx)
	}
	return elem.hashIt(nil), nil
}

func (e *ProofElement) hashIt(missingCommitment *[hashSize]byte) [hashSize]byte {
	var hashes [258]*[hashSize]byte
	for idx, c := range e.Children {
		hashes[idx] = (*[hashSize]byte)(c)
	}
	if e.Terminal != nil {
		hashes[256] = &e.Terminal.bytes
	}
	cd := commitToData(e.PathFragment)
	hashes[257] = &cd
	if e.ChildIndex < 256 {
		hashes[e.ChildIndex] = missingCommitment
	}
	return hashVector(&hashes)
}

func (p *Proof) Write(w io.Writer) error {
	if err := trie_go.WriteBytes16(w, p.Key); err != nil {
		return err
	}
	if err := trie_go.WriteUint16(w, uint16(len(p.Path))); err != nil {
		return err
	}
	for _, e := range p.Path {
		if err := e.Write(w); err != nil {
			return err
		}
	}
	return nil
}

func (p *Proof) Read(r io.Reader) error {
	var err error
	if p.Key, err = trie_go.ReadBytes16(r); err != nil {
		return err
	}
	var size uint16
	if err = trie_go.ReadUint16(r, &size); err != nil {
		return err
	}
	p.Path = make([]*ProofElement, size)
	for i := range p.Path {
		p.Path[i] = &ProofElement{}
		if err = p.Path[i].Read(r); err != nil {
			return err
		}
	}
	return nil
}

const (
	hasTerminalValueFlag = 0x01
	hasChildrenFlag      = 0x02
)

func (e *ProofElement) Write(w io.Writer) error {
	if err := trie_go.WriteBytes16(w, e.PathFragment); err != nil {
		return err
	}
	if err := trie_go.WriteUint16(w, uint16(e.ChildIndex)); err != nil {
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
	if err := trie_go.WriteByte(w, smallFlags); err != nil {
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
		for i := 0; i < 256; i++ {
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

func (e *ProofElement) Read(r io.Reader) error {
	var err error
	if e.PathFragment, err = trie_go.ReadBytes16(r); err != nil {
		return err
	}
	var idx uint16
	if err := trie_go.ReadUint16(r, &idx); err != nil {
		return err
	}
	e.ChildIndex = int(idx)
	var smallFlags byte
	if smallFlags, err = trie_go.ReadByte(r); err != nil {
		return err
	}
	if smallFlags&hasTerminalValueFlag != 0 {
		e.Terminal = &terminalCommitment{}
		if err := e.Terminal.Read(r); err != nil {
			return err
		}
	} else {
		e.Terminal = nil
	}
	e.Children = make(map[byte]*vectorCommitment)
	if smallFlags&hasChildrenFlag != 0 {
		var flags [32]byte
		if _, err := r.Read(flags[:]); err != nil {
			return err
		}
		for i := 0; i < 256; i++ {
			ib := uint8(i)
			if flags[i/8]&(0x1<<(i%8)) != 0 {
				e.Children[ib] = &vectorCommitment{}
				if err := e.Children[ib].Read(r); err != nil {
					return err
				}
			}
		}
	}
	return nil

}
