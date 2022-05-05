package trie256p

import (
	"bytes"
	"fmt"
	trie_go "github.com/iotaledger/trie.go"
	"golang.org/x/xerrors"
	"io"
)

// NodeData contains all data trie node needs to compute commitment
type NodeData struct {
	PathFragment     []byte
	ChildCommitments map[byte]trie_go.VCommitment
	Terminal         trie_go.TCommitment
}

func NewNodeData() *NodeData {
	return &NodeData{
		PathFragment:     nil,
		ChildCommitments: make(map[byte]trie_go.VCommitment),
		Terminal:         nil,
	}
}

func NodeDataFromBytes(model CommitmentModel, data, unpackedKey []byte, arity PathArity) (*NodeData, error) {
	ret := NewNodeData()
	if err := ret.Read(bytes.NewReader(data), model, unpackedKey, arity); err != nil {
		return nil, err
	}
	return ret, nil
}

// Clone deep copy
func (n *NodeData) Clone() *NodeData {
	ret := &NodeData{
		PathFragment:     make([]byte, len(n.PathFragment)),
		ChildCommitments: make(map[byte]trie_go.VCommitment),
	}
	if n.Terminal != nil {
		ret.Terminal = n.Terminal.Clone()
	}
	copy(ret.PathFragment, n.PathFragment)
	for i, c := range n.ChildCommitments {
		ret.ChildCommitments[i] = c.Clone()
	}
	return ret
}

func (n *NodeData) String() string {
	t := "<nil>"
	if n.Terminal != nil {
		t = n.Terminal.String()
	}
	ret := fmt.Sprintf("pf: '%s', term: '%s', ch: %d\n", string(n.PathFragment), t, len(n.ChildCommitments))
	for i := 0; i < 256; i++ {
		if c, ok := n.ChildCommitments[byte(i)]; ok {
			ret += fmt.Sprintf("    %d('%c'): %s\n", i, i, c)
		}
	}
	return ret
}

// Read/Write implements optimized serialization of the trie node
// The serialization of the node takes advantage of the fact that most of the
// nodes has just few children.
// the 'smallFlags' (1 byte) contains information:
// - 'serializeTerminalValueFlag' does node contain Terminal commitment
// - 'serializeChildrenFlag' does node contain at least one child
// - 'isKeyCommitmentFlag' is optimization case when commitment to the terminal == commitment to the key
//    In this case terminal is not serialized
// - 'serializePathFragmentFlag' flag means node has non-empty path fragment
// By the semantics of the trie, 'smallFlags' cannot be 0
// 'childrenFlags' (32 bytes array or 256 bits) are only present if node contains at least one child commitment
// In this case:
// if node has a child commitment at the position of i, 0 <= p <= 255, it has a bit in the byte array
// at the index i/8. The bit position in the byte is i % 8

const (
	isKeyCommitmentFlag        = 0x01
	serializeTerminalValueFlag = 0x02
	serializeChildrenFlag      = 0x04
	serializePathFragmentFlag  = 0x08
	optimizedPathArityFlag     = 0x10 // if set, it is binary or hexary
	binaryPath                 = 0x20 // is set, it is binary, otherwise hexary. Only makes sense if optimizedPathArityFlag is set.
)

// cflags256 256 flags, one for each child
type cflags256 [32]byte

func (fl *cflags256) setFlag(i byte) {
	fl[i/8] |= 0x1 << (i % 8)
}

func (fl *cflags256) hasFlag(i byte) bool {
	return fl[i/8]&(0x1<<(i%8)) != 0
}

func (n *NodeData) Write(w io.Writer, arity PathArity, isKeyCommitment bool) error {
	var smallFlags byte
	if isKeyCommitment {
		smallFlags |= isKeyCommitmentFlag
	}
	if !isKeyCommitment && n.Terminal != nil {
		smallFlags |= serializeTerminalValueFlag
	}
	if len(n.ChildCommitments) > 0 {
		smallFlags |= serializeChildrenFlag
	}
	var childrenFlags cflags256
	if smallFlags&serializeChildrenFlag != 0 {
		// compress children childrenFlags 32 bytes, if any
		for i := range n.ChildCommitments {
			childrenFlags.setFlag(i)
		}
	}
	if smallFlags == 0 {
		return xerrors.New("non-committing node can't be serialized")
	}
	var pathFragmentEncoded []byte
	var err error
	if len(n.PathFragment) > 0 {
		smallFlags |= serializePathFragmentFlag
		if pathFragmentEncoded, err = encodeKey(pathFragmentEncoded, arity); err != nil {
			return err
		}
	}
	if err := trie_go.WriteByte(w, smallFlags); err != nil {
		return err
	}
	if smallFlags&serializePathFragmentFlag != 0 {
		if err := trie_go.WriteBytes16(w, pathFragmentEncoded); err != nil {
			return err
		}
	}
	// write Terminal commitment if needed
	// if key is committed as terminal, terminal is not serialized
	if smallFlags&serializeTerminalValueFlag != 0 {
		if err := n.Terminal.Write(w); err != nil {
			return err
		}
	}
	// write child commitments if any
	if smallFlags&serializeChildrenFlag != 0 {
		if _, err := w.Write(childrenFlags[:]); err != nil {
			return err
		}
		for i := 0; i < 256; i++ {
			child, ok := n.ChildCommitments[uint8(i)]
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

// Read deserialized node data and returns isKeyCommitmentFlag value
func (n *NodeData) Read(r io.Reader, model CommitmentModel, unpackedKey []byte, arity PathArity) error {
	var err error
	var smallFlags byte
	if smallFlags, err = trie_go.ReadByte(r); err != nil {
		return err
	}
	if smallFlags&serializePathFragmentFlag != 0 {
		encoded, err := trie_go.ReadBytes16(r)
		if err != nil {
			return err
		}
		if n.PathFragment, err = decodeKey(encoded, arity); err != nil {
			return err
		}
	} else {
		n.PathFragment = nil
	}
	n.Terminal = nil
	if smallFlags&serializeTerminalValueFlag != 0 {
		if smallFlags&isKeyCommitmentFlag != 0 {
			return xerrors.New("wrong flag")
		}
		n.Terminal = model.NewTerminalCommitment()
		if err = n.Terminal.Read(r); err != nil {
			return err
		}
	} else {
		if smallFlags&isKeyCommitmentFlag != 0 {
			if len(unpackedKey) == 0 {
				return xerrors.New("non-empty unpackedKey expected")
			}
			n.Terminal = model.CommitToData(trie_go.Concat(unpackedKey, n.PathFragment))
		}
	}
	if smallFlags&serializeChildrenFlag != 0 {
		var flags cflags256
		if _, err = r.Read(flags[:]); err != nil {
			return err
		}
		for i := 0; i < 256; i++ {
			ib := uint8(i)
			if flags.hasFlag(ib) {
				n.ChildCommitments[ib] = model.NewVectorCommitment()
				if err = n.ChildCommitments[ib].Read(r); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
