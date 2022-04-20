package trie256p

import (
	"bytes"
	"fmt"
	trie_go "github.com/iotaledger/trie.go"
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

func NodeDataFromBytes(model CommitmentModel, data []byte) (*NodeData, error) {
	ret := NewNodeData()
	if err := ret.Read(bytes.NewReader(data), model); err != nil {
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
// The serialization of the node takes advantage of the fact that most of the nodes has just few children
// the 'smallFlags' (1 byte) contains information:
// - does node contain Terminal commitment
// - does node contain at least one child
// By the semantics of the trie, 'smallFlags' cannot be 0
// 'childrenFlags' (32 bytes array or 256 bits) are only present if node contains at least one child commitment
// In this case:
// if node has a child commitment at the position of i, 0 <= p <= 255, it has a bit in the byte array
// at the index i/8. The bit position in the byte is i % 8

const (
	hasTerminalValueFlag = 0x01
	hasChildrenFlag      = 0x02
)

type cflags [32]byte

func (fl *cflags) setFlag(i byte) {
	fl[i/8] |= 0x1 << (i % 8)
}

func (fl *cflags) hasFlag(i byte) bool {
	return fl[i/8]&(0x1<<(i%8)) != 0
}

func (n *NodeData) Write(w io.Writer) error {
	if err := trie_go.WriteBytes16(w, n.PathFragment); err != nil {
		return err
	}

	var smallFlags byte
	if n.Terminal != nil {
		smallFlags = hasTerminalValueFlag
	}
	// compress children childrenFlags 32 bytes (if any)
	var childrenFlags cflags
	for i := range n.ChildCommitments {
		childrenFlags.setFlag(i)
		smallFlags |= hasChildrenFlag
	}
	if err := trie_go.WriteByte(w, smallFlags); err != nil {
		return err
	}
	// write Terminal commitment if any
	if smallFlags&hasTerminalValueFlag != 0 {
		if err := n.Terminal.Write(w); err != nil {
			return err
		}
	}
	// write child commitments if any
	if smallFlags&hasChildrenFlag != 0 {
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

func (n *NodeData) Read(r io.Reader, setup CommitmentModel) error {
	var err error
	if n.PathFragment, err = trie_go.ReadBytes16(r); err != nil {
		return err
	}
	var smallFlags byte
	if smallFlags, err = trie_go.ReadByte(r); err != nil {
		return err
	}
	if smallFlags&hasTerminalValueFlag != 0 {
		n.Terminal = setup.NewTerminalCommitment()
		if err := n.Terminal.Read(r); err != nil {
			return err
		}
	} else {
		n.Terminal = nil
	}
	if smallFlags&hasChildrenFlag != 0 {
		var flags cflags
		if _, err := r.Read(flags[:]); err != nil {
			return err
		}
		for i := 0; i < 256; i++ {
			ib := uint8(i)
			if flags.hasFlag(ib) {
				n.ChildCommitments[ib] = setup.NewVectorCommitment()
				if err := n.ChildCommitments[ib].Read(r); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (n *NodeData) Bytes() []byte {
	return trie_go.MustBytes(n)
}
