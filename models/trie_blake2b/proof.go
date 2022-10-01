package trie_blake2b

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/iotaledger/trie.go/common"
	"github.com/iotaledger/trie.go/mutable"
)

// Proof blake2b 20 byte common-specific proof of inclusion
type Proof struct {
	PathArity common.PathArity
	HashSize  HashSize
	Key       []byte
	Path      []*ProofElement
}

type ProofElement struct {
	PathFragment []byte
	Children     map[byte][]byte
	Terminal     []byte
	ChildIndex   int
}

func ProofFromBytes(data []byte) (*Proof, error) {
	ret := &Proof{}
	rdr := bytes.NewReader(data)
	if err := ret.Read(rdr); err != nil {
		return nil, err
	}
	if rdr.Len() != 0 {
		return nil, common.ErrNotAllBytesConsumed
	}
	return ret, nil
}

// Proof converts generic proof path to the Merkle proof path
func (m *CommitmentModel) Proof(key []byte, tr mutable.NodeStore) *Proof {
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
			case mutable.EndingTerminal:
				childIndex = m.arity.TerminalCommitmentIndex()
			case mutable.EndingExtend, mutable.EndingSplit:
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
			em.Terminal = node.Terminal().(*terminalCommitment).bytes
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
	return common.MustBytes(p)
}

func (p *Proof) Write(w io.Writer) error {
	var err error
	if err = common.WriteByte(w, byte(p.PathArity)); err != nil {
		return err
	}
	if err = common.WriteByte(w, byte(p.HashSize)); err != nil {
		return err
	}
	encodedKey, err := common.EncodeUnpackedBytes(p.Key, p.PathArity)
	if err != nil {
		return err
	}
	if err = common.WriteBytes16(w, encodedKey); err != nil {
		return err
	}
	if err = common.WriteUint16(w, uint16(len(p.Path))); err != nil {
		return err
	}
	for _, e := range p.Path {
		if err = e.Write(w, p.PathArity, p.HashSize); err != nil {
			return err
		}
	}
	return nil
}

func (p *Proof) Read(r io.Reader) error {
	b, err := common.ReadByte(r)
	if err != nil {
		return err
	}
	p.PathArity = common.PathArity(b)

	b, err = common.ReadByte(r)
	if err != nil {
		return err
	}
	p.HashSize = HashSize(b)
	if p.HashSize != HashSize256 && p.HashSize != HashSize160 {
		return errors.New("wrong hash size")
	}

	var encodedKey []byte
	if encodedKey, err = common.ReadBytes16(r); err != nil {
		return err
	}
	if p.Key, err = common.DecodeToUnpackedBytes(encodedKey, p.PathArity); err != nil {
		return err
	}
	var size uint16
	if err = common.ReadUint16(r, &size); err != nil {
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

func (e *ProofElement) Write(w io.Writer, arity common.PathArity, sz HashSize) error {
	encodedPathFragment, err := common.EncodeUnpackedBytes(e.PathFragment, arity)
	if err != nil {
		return err
	}
	if err = common.WriteBytes16(w, encodedPathFragment); err != nil {
		return err
	}
	if err = common.WriteUint16(w, uint16(e.ChildIndex)); err != nil {
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
	if err := common.WriteByte(w, smallFlags); err != nil {
		return err
	}
	// write terminal commitment if any
	if smallFlags&hasTerminalValueFlag != 0 {
		if err = common.WriteBytes8(w, e.Terminal); err != nil {
			return err
		}
	}
	// write child commitments if any
	if smallFlags&hasChildrenFlag != 0 {
		if _, err = w.Write(flags[:]); err != nil {
			return err
		}
		for i := 0; i < arity.VectorLength(); i++ {
			child, ok := e.Children[uint8(i)]
			if !ok {
				continue
			}
			if len(child) != int(sz) {
				return fmt.Errorf("wrong data size. Expected %s, got %d", sz.String(), len(child))
			}
			if _, err = w.Write(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *ProofElement) Read(r io.Reader, arity common.PathArity, sz HashSize) error {
	var err error
	var encodedPathFragment []byte
	if encodedPathFragment, err = common.ReadBytes16(r); err != nil {
		return err
	}
	if e.PathFragment, err = common.DecodeToUnpackedBytes(encodedPathFragment, arity); err != nil {
		return err
	}
	var idx uint16
	if err := common.ReadUint16(r, &idx); err != nil {
		return err
	}
	e.ChildIndex = int(idx)
	var smallFlags byte
	if smallFlags, err = common.ReadByte(r); err != nil {
		return err
	}
	if smallFlags&hasTerminalValueFlag != 0 {
		if e.Terminal, err = common.ReadBytes8(r); err != nil {
			return err
		}
	} else {
		e.Terminal = nil
	}
	e.Children = make(map[byte][]byte)
	if smallFlags&hasChildrenFlag != 0 {
		var flags [32]byte
		if _, err = r.Read(flags[:]); err != nil {
			return err
		}
		for i := 0; i < arity.NumChildren(); i++ {
			ib := uint8(i)
			if flags[i/8]&(0x1<<(i%8)) != 0 {
				e.Children[ib] = make([]byte, sz)
				if _, err = r.Read(e.Children[ib]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
