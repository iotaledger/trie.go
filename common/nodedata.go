package common

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"golang.org/x/xerrors"
)

func NewNodeData() *NodeData {
	return &NodeData{
		PathFragment:     nil,
		ChildCommitments: make(map[byte]VCommitment),
		Terminal:         nil,
	}
}

func NodeDataFromBytes(model CommitmentModel, data []byte, arity PathArity, getValueFunc func(pathFragment []byte) ([]byte, error)) (*NodeData, error) {
	ret := NewNodeData()
	rdr := bytes.NewReader(data)
	if err := ret.Read(rdr, model, arity, getValueFunc); err != nil {
		return nil, err
	}
	if rdr.Len() != 0 {
		// not all data was consumed
		return nil, ErrNotAllBytesConsumed
	}
	return ret, nil
}

// Clone deep copy
func (n *NodeData) Clone() *NodeData {
	ret := &NodeData{
		PathFragment:     Concat(n.PathFragment),
		ChildCommitments: make(map[byte]VCommitment),
	}
	if !IsNil(n.Terminal) {
		ret.Terminal = n.Terminal.Clone()
	}
	if !IsNil(n.Commitment) {
		ret.Commitment = n.Commitment.Clone()
	}
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
	childIdx := make([]byte, 0)
	for i := 0; i < 256; i++ {
		_, ok := n.ChildCommitments[byte(i)]
		if !ok {
			continue
		}
		childIdx = append(childIdx, byte(i))
	}
	return fmt.Sprintf("c: %s, pf: '%s', childrenIdx: %v, term: '%s'",
		n.Commitment, string(n.PathFragment), childIdx, t)
}

// Read/Write implements optimized serialization of the trie node
// The serialization of the node takes advantage of the fact that most of the
// nodes has just few children.
// the 'smallFlags' (1 byte) contains information:
// - 'takeTerminalFromKeyFlag' does node contain terminal commitment
// - 'serializeChildrenFlag' does node contain at least one child
// - 'terminalExistsFlag' is optimization case when commitment to the terminal == commitment to the unpackedKey
//    In this case terminal is not serialized
// - 'serializePathFragmentFlag' flag means node has non-empty path fragment
// By the semantics of the trie, 'smallFlags' cannot be 0
// 'childrenFlags' (32 bytes array or 256 bits) are only present if node contains at least one child commitment
// In this case:
// if node has a child commitment at the position of i, 0 <= p <= 255, it has a bit in the byte array
// at the index i/8. The bit position in the byte is i % 8

const (
	terminalExistsFlag        = 0x01
	takeTerminalFromValueFlag = 0x02
	serializeChildrenFlag     = 0x04
	serializePathFragmentFlag = 0x08
)

// cflags 256 flags, one for each child
type cflags []byte

func cflagsSize(arity PathArity) int {
	if ret := (int(arity) + 1) / 8; ret != 0 {
		return ret
	}
	return 1
}

func newCflags(arity PathArity) cflags {
	return make(cflags, cflagsSize(arity))
}

func readCflags(r io.Reader, arity PathArity) (cflags, error) {
	ret := newCflags(arity)
	n, err := r.Read(ret)
	if err != nil {
		return nil, err
	}
	if n != cflagsSize(arity) {
		return nil, fmt.Errorf("expected %d bytes, got %d", cflagsSize(arity), n)
	}
	return ret, nil
}

func (fl cflags) setFlag(i byte) {
	fl[i/8] |= 0x1 << (i % 8)
}

func (fl cflags) hasFlag(i byte) bool {
	return fl[i/8]&(0x1<<(i%8)) != 0
}

// Write serialized node data
func (n *NodeData) Write(w io.Writer, arity PathArity, skipTerminal bool) error {
	var smallFlags byte
	if n.Terminal != nil {
		smallFlags |= terminalExistsFlag
	}
	if skipTerminal {
		smallFlags |= takeTerminalFromValueFlag
	}
	if len(n.ChildCommitments) > 0 {
		smallFlags |= serializeChildrenFlag
	}
	if smallFlags == 0 {
		return xerrors.New("non-committing node can't be serialized")
	}
	var pathFragmentEncoded []byte
	var err error
	if len(n.PathFragment) > 0 {
		smallFlags |= serializePathFragmentFlag
		if pathFragmentEncoded, err = EncodeUnpackedBytes(n.PathFragment, arity); err != nil {
			return err
		}
	}
	if err = WriteByte(w, smallFlags); err != nil {
		return err
	}
	if smallFlags&serializePathFragmentFlag != 0 {
		if err = WriteBytes16(w, pathFragmentEncoded); err != nil {
			return err
		}
	}
	// write terminal commitment if not skipped for at least one of three reasons
	if smallFlags&terminalExistsFlag != 0 {
		// terminal exists
		if smallFlags&takeTerminalFromValueFlag == 0 {
			// terminal will be stored in the node
			if err = n.Terminal.Write(w); err != nil {
				return err
			}
		}
	}
	// write child commitments if any
	if smallFlags&serializeChildrenFlag != 0 {
		childrenFlags := newCflags(arity)
		// compress children childrenFlags 32 bytes, if any
		for i := range n.ChildCommitments {
			childrenFlags.setFlag(i)
		}
		if _, err = w.Write(childrenFlags); err != nil {
			return err
		}
		for i := 0; i < int(arity)+1; i++ {
			child, ok := n.ChildCommitments[uint8(i)]
			if !ok {
				continue
			}
			if err = child.Write(w); err != nil {
				return err
			}
		}
	}
	return nil
}

// Read deserialize node data
func (n *NodeData) Read(r io.Reader, model CommitmentModel, arity PathArity, getValue func(pathFragment []byte) ([]byte, error)) error {
	var err error
	var smallFlags byte
	if smallFlags, err = ReadByte(r); err != nil {
		return err
	}
	if smallFlags&serializePathFragmentFlag != 0 {
		encoded, err := ReadBytes16(r)
		if err != nil {
			return err
		}
		if n.PathFragment, err = DecodeToUnpackedBytes(encoded, arity); err != nil {
			return err
		}
	} else {
		n.PathFragment = nil
	}
	n.Terminal = nil
	if smallFlags&terminalExistsFlag != 0 {
		// terminal exists. Should be taken from 1 or 3 locations
		if smallFlags&takeTerminalFromValueFlag != 0 {
			value, err := getValue(n.PathFragment)
			if err != nil {
				return err
			}
			n.Terminal = model.CommitToData(value)
		} else {
			n.Terminal = model.NewTerminalCommitment()
			if err = n.Terminal.Read(r); err != nil {
				return err
			}
		}
	} else {
		// terminal does not exist. Enforce other flags to be 0
		if smallFlags&(takeTerminalFromValueFlag) != 0 {
			return errors.New("wrong flag")
		}
	}
	if smallFlags&serializeChildrenFlag != 0 {
		var flags cflags
		if flags, err = readCflags(r, arity); err != nil {
			return err
		}
		for i := 0; i < int(arity)+1; i++ {
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

func (n *NodeData) IterateChildren(f func(byte, VCommitment) bool) bool {
	for i := 0; i < 256; i++ {
		i := byte(i)
		if v, ok := n.ChildCommitments[i]; ok {
			if !f(i, v) {
				return false
			}
		}
	}
	return true
}
