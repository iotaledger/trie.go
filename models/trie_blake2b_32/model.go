// Package trie_blake2b_32 implements trie.CommitmentModel based on blake2b 32-byte hashing
package trie_blake2b_32

import (
	"encoding/hex"
	"fmt"
	"io"

	"github.com/iotaledger/trie.go/trie"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/xerrors"
)

const hashSize = 32

// terminalCommitment commits to the data of arbitrary size.
// If len(data) <= hashSize, then lenPlus1 = len(data)+1 and bytes contains the data itself padded with 0's in the end
// If len(data) > hashSize, the bytes contains blake2b hash of the data and lenPlus1 = 0
// So, a correct value of lenPlus1 is 0 to 33
type terminalCommitment struct {
	bytes    [hashSize]byte
	lenPlus1 uint8
}

// vectorCommitment is a blake2b hash of the vector elements
type vectorCommitment [hashSize]byte

// CommitmentModel provides commitment model implementation for the 256+ trie
type CommitmentModel struct {
	arity trie.PathArity
}

// New creates new CommitmentModel. Optional arity, if is specified and is less that trie.PathArity256,
// optimizes certain operations
func New(arity trie.PathArity) *CommitmentModel {
	return &CommitmentModel{arity: arity}
}

func (m *CommitmentModel) PathArity() trie.PathArity {
	return m.arity
}

// NewTerminalCommitment creates empty terminal commitment
func (m *CommitmentModel) NewTerminalCommitment() trie.TCommitment {
	return &terminalCommitment{}
}

// NewVectorCommitment create empty vector commitment
func (m *CommitmentModel) NewVectorCommitment() trie.VCommitment {
	return &vectorCommitment{}
}

// UpdateNodeCommitment computes update to the node data and, optionally, updates existing commitment
// In blake2b implementation delta it just means computing the hash of data
func (m *CommitmentModel) UpdateNodeCommitment(mutate *trie.NodeData, childUpdates map[byte]trie.VCommitment, _ bool, newTerminalUpdate trie.TCommitment, update *trie.VCommitment) {
	hashes := make([]*[hashSize]byte, m.arity.VectorLength())

	deleted := make([]byte, 0, 256)
	for i, upd := range childUpdates {
		mutate.ChildCommitments[i] = upd
		if upd == nil {
			// if update == nil, it means child commitment must be removed
			deleted = append(deleted, i)
		}
	}
	for _, i := range deleted {
		delete(mutate.ChildCommitments, i)
	}
	for i, c := range mutate.ChildCommitments {
		trie.Assert(int(i) < m.arity.VectorLength(), "int(i)<m.arity.VectorLength()")
		hashes[i] = (*[hashSize]byte)(c.(*vectorCommitment))
	}
	mutate.Terminal = newTerminalUpdate // for hash commitment just replace
	if mutate.Terminal != nil {
		// arity+1 is the position of the terminal commitment, if any
		hashes[m.arity.TerminalCommitmentIndex()] = &mutate.Terminal.(*terminalCommitment).bytes
	}
	if len(mutate.ChildCommitments) == 0 && mutate.Terminal == nil {
		return
	}
	commitmentToPathFragment := commitToData(mutate.PathFragment)
	hashes[m.arity.PathFragmentCommitmentIndex()] = &commitmentToPathFragment
	if update != nil {
		c := (vectorCommitment)(hashVector(hashes))
		*update = &c
	}
}

// CalcNodeCommitment computes commitment of the node. It is suboptimal in KZG trie.
// Used in computing root commitment
func (m *CommitmentModel) CalcNodeCommitment(par *trie.NodeData) trie.VCommitment {
	hashes := make([]*[hashSize]byte, m.arity.VectorLength())

	if len(par.ChildCommitments) == 0 && par.Terminal == nil {
		return nil
	}
	for i, c := range par.ChildCommitments {
		trie.Assert(int(i) < m.arity.VectorLength(), "int(i)<m.arity.VectorLength()")
		hashes[i] = (*[hashSize]byte)(c.(*vectorCommitment))
	}
	if par.Terminal != nil {
		hashes[m.arity.TerminalCommitmentIndex()] = &par.Terminal.(*terminalCommitment).bytes
	}
	commitmentToPathFragment := commitToData(par.PathFragment)
	hashes[m.arity.PathFragmentCommitmentIndex()] = &commitmentToPathFragment
	c := (vectorCommitment)(hashVector(hashes))
	return &c
}

func (m *CommitmentModel) CommitToData(data []byte) trie.TCommitment {
	if len(data) == 0 {
		// empty slice -> no data (deleted)
		return nil
	}
	return commitToTerminal(data)
}

func (m *CommitmentModel) Description() string {
	return fmt.Sprintf("trie commitment model implementation based on blake2b 256 bit hashing, %s", m.arity)
}

func (m *CommitmentModel) ShortName() string {
	return "b2b20"
}

// *vectorCommitment implements trie_go.VCommitment
var _ trie.VCommitment = &vectorCommitment{}

func (v *vectorCommitment) Bytes() []byte {
	return trie.MustBytes(v)
}

func (v *vectorCommitment) Read(r io.Reader) error {
	_, err := r.Read((*v)[:])
	return err
}

func (v *vectorCommitment) Write(w io.Writer) error {
	_, err := w.Write((*v)[:])
	return err
}

func (v *vectorCommitment) String() string {
	return hex.EncodeToString(v[:])
}

func (v *vectorCommitment) Clone() trie.VCommitment {
	if v == nil {
		return nil
	}
	ret := *v
	return &ret
}

func (v *vectorCommitment) Update(delta trie.VCommitment) {
	m, ok := delta.(*vectorCommitment)
	if !ok {
		panic("hash commitment expected")
	}
	*v = *m
}

// *terminalCommitment implements trie_go.TCommitment
var _ trie.TCommitment = &terminalCommitment{}

func (t *terminalCommitment) Write(w io.Writer) error {
	if err := trie.WriteByte(w, t.lenPlus1); err != nil {
		return err
	}
	l := byte(hashSize)
	if t.lenPlus1 > 0 {
		l = t.lenPlus1 - 1
	}
	_, err := w.Write(t.bytes[:l])
	return err
}

func (t *terminalCommitment) Read(r io.Reader) error {
	var err error
	if t.lenPlus1, err = trie.ReadByte(r); err != nil {
		return err
	}
	if t.lenPlus1 > 33 {
		return xerrors.New("terminal commitment size byte must be <= 33")
	}
	l := byte(hashSize)
	if t.lenPlus1 > 0 {
		l = t.lenPlus1 - 1
	}
	t.bytes = [hashSize]byte{}
	n, err := r.Read(t.bytes[:l])
	if err != nil {
		return err
	}
	if n != int(l) {
		return xerrors.New("bad data length")
	}
	return nil
}

func (t *terminalCommitment) Bytes() []byte {
	return trie.MustBytes(t)
}

func (t *terminalCommitment) String() string {
	return hex.EncodeToString(t.bytes[:])
}

func (t *terminalCommitment) Clone() trie.TCommitment {
	if t == nil {
		return nil
	}
	ret := *t
	return &ret
}

// return value of the terminal commitment and a flag which indicates if it is a hashed value (true) or original data (false)
func (t *terminalCommitment) value() ([]byte, bool) {
	return t.bytes[:t.lenPlus1-1], t.lenPlus1 == 0
}

func commitToData(data []byte) (ret [hashSize]byte) {
	if len(data) <= hashSize {
		copy(ret[:], data)
	} else {
		ret = blake2b.Sum256(data)
	}
	return
}

func commitToTerminal(data []byte) *terminalCommitment {
	ret := &terminalCommitment{
		bytes: commitToData(data),
	}
	if len(data) <= hashSize {
		ret.lenPlus1 = uint8(len(data)) + 1 // 1-33
	}
	return ret
}

func hashVector(hashes []*[hashSize]byte) [hashSize]byte {
	buf := make([]byte, len(hashes)*hashSize)
	for i, h := range hashes {
		if h == nil {
			continue
		}
		pos := hashSize * int(i)
		copy(buf[pos:pos+hashSize], h[:])
	}
	return blake2b.Sum256(buf[:])
}
