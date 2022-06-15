package trie

import (
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/xerrors"
	"io"
)

// NodeData contains all data trie node needs to compute commitment
type NodeData struct {
	PathFragment     []byte
	ChildCommitments map[byte]VCommitment
	Terminal         TCommitment
}

func NewNodeData() *NodeData {
	return &NodeData{
		PathFragment:     nil,
		ChildCommitments: make(map[byte]VCommitment),
		Terminal:         nil,
	}
}

func NodeDataFromBytes(model CommitmentModel, data, unpackedKey []byte, arity PathArity, valueStore KVReader) (*NodeData, error) {
	ret := NewNodeData()
	if err := ret.Read(bytes.NewReader(data), model, unpackedKey, arity, valueStore); err != nil {
		return nil, err
	}
	return ret, nil
}

// Clone deep copy
func (n *NodeData) Clone() *NodeData {
	ret := &NodeData{
		PathFragment:     make([]byte, len(n.PathFragment)),
		ChildCommitments: make(map[byte]VCommitment),
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
// - 'takeTerminalFromKeyFlag' does node contain Terminal commitment
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
	takeTerminalFromKeyFlag   = 0x04
	serializeChildrenFlag     = 0x08
	serializePathFragmentFlag = 0x10
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
func (n *NodeData) Write(w io.Writer, arity PathArity, isKeyCommitment bool, skipTerminal bool) error {
	var smallFlags byte
	if n.Terminal != nil {
		smallFlags |= terminalExistsFlag
	}
	if skipTerminal {
		smallFlags |= takeTerminalFromValueFlag
	}
	if isKeyCommitment {
		smallFlags |= takeTerminalFromKeyFlag
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
	// write Terminal commitment if not skipped for at least one of three reasons
	if smallFlags&terminalExistsFlag != 0 &&
		smallFlags&takeTerminalFromKeyFlag == 0 &&
		smallFlags&takeTerminalFromValueFlag == 0 {
		if err = n.Terminal.Write(w); err != nil {
			return err
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
func (n *NodeData) Read(r io.Reader, model CommitmentModel, unpackedKey []byte, arity PathArity, valueStore KVReader) error {
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
		if smallFlags&takeTerminalFromKeyFlag != 0 {
			// terminal is in key
			if len(unpackedKey) == 0 {
				return xerrors.New("non-empty unpackedKey expected")
			}
			n.Terminal = model.CommitToData(Concat(unpackedKey, n.PathFragment))
		} else if smallFlags&takeTerminalFromValueFlag != 0 {
			// terminal should be taken from the value store
			if valueStore == nil {
				return errors.New("can't read node: value store not provided")
			}
			key, err := EncodeUnpackedBytes(unpackedKey, arity)
			if err != nil {
				return err
			}
			value := valueStore.Get(key)
			if value == nil {
				return fmt.Errorf("can't find terminal value for key %X", key)
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
		if smallFlags&(takeTerminalFromKeyFlag|takeTerminalFromValueFlag) != 0 {
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
