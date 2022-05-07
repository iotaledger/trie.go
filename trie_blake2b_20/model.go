// Package trie_blake2b_20 implements trie.CommitmentModel based on blake2b 32-byte hashing
package trie_blake2b_20

import (
	"encoding/hex"
	"io"

	trie_go "github.com/iotaledger/trie.go"
	"github.com/iotaledger/trie.go/trie"
	"golang.org/x/xerrors"
)

const hashSize = 20

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
}

func New() *CommitmentModel {
	return &CommitmentModel{}
}

// NewTerminalCommitment creates empty terminal commitment
func (m *CommitmentModel) NewTerminalCommitment() trie_go.TCommitment {
	return &terminalCommitment{}
}

// NewVectorCommitment create empty vector commitment
func (m *CommitmentModel) NewVectorCommitment() trie_go.VCommitment {
	return &vectorCommitment{}
}

// UpdateNodeCommitment computes update to the node data and, optionally, updates existing commitment
// In blake2b implementation delta it just means computing the hash of data
func (m *CommitmentModel) UpdateNodeCommitment(mutate *trie.NodeData, childUpdates map[byte]trie_go.VCommitment, _ bool, newTerminalUpdate trie_go.TCommitment, update *trie_go.VCommitment) {
	var hashes [258]*[hashSize]byte

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
		hashes[i] = (*[hashSize]byte)(c.(*vectorCommitment))
	}
	mutate.Terminal = newTerminalUpdate // for hash commitment just replace
	if mutate.Terminal != nil {
		// 256 is the position of the terminal commitment, if any
		hashes[256] = &mutate.Terminal.(*terminalCommitment).bytes
	}
	if len(mutate.ChildCommitments) == 0 && mutate.Terminal == nil {
		return
	}
	tmp := commitToData(mutate.PathFragment)

	hashes[257] = &tmp
	if update != nil {
		c := (vectorCommitment)(hashVector(&hashes))
		*update = &c
	}
}

// CalcNodeCommitment computes commitment of the node. It is suboptimal in KZG trie.
// Used in computing root commitment
func (m *CommitmentModel) CalcNodeCommitment(par *trie.NodeData) trie_go.VCommitment {
	var hashes [258]*[hashSize]byte

	if len(par.ChildCommitments) == 0 && par.Terminal == nil {
		return nil
	}
	for i, c := range par.ChildCommitments {
		hashes[i] = (*[hashSize]byte)(c.(*vectorCommitment))
	}
	if par.Terminal != nil {
		hashes[256] = &par.Terminal.(*terminalCommitment).bytes
	}
	tmp := commitToData(par.PathFragment)
	hashes[257] = &tmp
	c := (vectorCommitment)(hashVector(&hashes))
	return &c
}

func (m *CommitmentModel) CommitToData(data []byte) trie_go.TCommitment {
	if len(data) == 0 {
		// empty slice -> no data (deleted)
		return nil
	}
	return commitToTerminal(data)
}

func (m *CommitmentModel) Description() string {
	return "trie commitment model implementation based on blake2b 160 bit hashing"
}

func (m *CommitmentModel) ShortName() string {
	return "b2b20"
}

// *vectorCommitment implements trie_go.VCommitment
var _ trie_go.VCommitment = &vectorCommitment{}

func (v *vectorCommitment) Bytes() []byte {
	return trie_go.MustBytes(v)
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

func (v *vectorCommitment) Clone() trie_go.VCommitment {
	if v == nil {
		return nil
	}
	ret := *v
	return &ret
}

func (v *vectorCommitment) Update(delta trie_go.VCommitment) {
	m, ok := delta.(*vectorCommitment)
	if !ok {
		panic("hash commitment expected")
	}
	*v = *m
}

// *terminalCommitment implements trie_go.TCommitment
var _ trie_go.TCommitment = &terminalCommitment{}

func (t *terminalCommitment) Write(w io.Writer) error {
	if err := trie_go.WriteByte(w, t.lenPlus1); err != nil {
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
	if t.lenPlus1, err = trie_go.ReadByte(r); err != nil {
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
	return trie_go.MustBytes(t)
}

func (t *terminalCommitment) String() string {
	return hex.EncodeToString(t.bytes[:])
}

func (t *terminalCommitment) Clone() trie_go.TCommitment {
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

func hashVector(hashes *[258]*[hashSize]byte) [hashSize]byte {
	var buf [258 * hashSize]byte
	for i, h := range hashes {
		if h == nil {
			continue
		}
		pos := hashSize * int(i)
		copy(buf[pos:pos+hashSize], h[:])
	}
	return trie_go.Blake2b160(buf[:])
}

func commitToData(data []byte) (ret [hashSize]byte) {
	if len(data) <= hashSize {
		copy(ret[:], data)
	} else {
		ret = trie_go.Blake2b160(data)
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
