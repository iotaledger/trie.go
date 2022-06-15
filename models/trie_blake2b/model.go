// Package trie_blake2b_20 implements trie.CommitmentModel based on blake2b 32-byte hashing
package trie_blake2b

import (
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/crypto/blake2b"
	"io"

	"github.com/iotaledger/trie.go/trie"
)

// terminalCommitment commits to the data of arbitrary size.
// len(bytes) can't be > 32
// if isHash == true, len(bytes) must be 32
// otherwise it is not hashed value, mus be len(bytes) <= 32
type terminalCommitment struct {
	bytes  []byte
	isHash bool
	// not persistent
	hashSize HashSize
}

// vectorCommitment is a blake2b hash of the vector elements
type vectorCommitment []byte

type HashSize byte

const (
	HashSize160 = HashSize(20)
	HashSize256 = HashSize(32)
)

func (hs HashSize) String() string {
	switch hs {
	case HashSize256:
		return "HashSize(256)"
	case HashSize160:
		return "HashSize(160)"
	}
	panic("wrong hash size")
}

// CommitmentModel provides commitment model implementation for the 256+ trie
type CommitmentModel struct {
	HashSize
	arity             trie.PathArity
	optimizeTerminals bool
}

// New creates new CommitmentModel.
// if optimizeTerminals == true, function StoreTerminalWithNode return true only for hashed values
// The trie node serialization takes advantage of it and does not serialize terminals shorter than
// hash. It prevents duplication of short values in DB
func New(arity trie.PathArity, hashSize HashSize, optimizeTerminals ...bool) *CommitmentModel {
	o := false
	if len(optimizeTerminals) > 0 {
		o = optimizeTerminals[0]
	}
	return &CommitmentModel{
		HashSize:          hashSize,
		arity:             arity,
		optimizeTerminals: o,
	}
}

func (m *CommitmentModel) PathArity() trie.PathArity {
	return m.arity
}

// TODO optimize vector size wrt path arity

// UpdateNodeCommitment computes update to the node data and, optionally, updates existing commitment
// In blake2b implementation delta it just means computing the hash of data
func (m *CommitmentModel) UpdateNodeCommitment(mutate *trie.NodeData, childUpdates map[byte]trie.VCommitment, _ bool, newTerminalUpdate trie.TCommitment, update *trie.VCommitment) {
	hashes := make([][]byte, m.arity.VectorLength())

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
		hashes[i] = c.(vectorCommitment)
	}
	mutate.Terminal = newTerminalUpdate // for hash commitment just replace
	if mutate.Terminal != nil {
		// arity+1 is the position of the terminal commitment, if any
		hashes[m.arity.TerminalCommitmentIndex()] = mutate.Terminal.(*terminalCommitment).bytes
	}
	if len(mutate.ChildCommitments) == 0 && mutate.Terminal == nil {
		return
	}
	hashes[m.arity.PathFragmentCommitmentIndex()] = commitToData(mutate.PathFragment, m.HashSize)
	if update != nil {
		*update = (vectorCommitment)(hashVector(hashes, m.HashSize))
	}
}

// CalcNodeCommitment computes commitment of the node. It is suboptimal in KZG trie.
// Used in computing root commitment
func (m *CommitmentModel) CalcNodeCommitment(par *trie.NodeData) trie.VCommitment {
	hashes := make([][]byte, m.arity.VectorLength())

	if len(par.ChildCommitments) == 0 && par.Terminal == nil {
		return nil
	}
	for i, c := range par.ChildCommitments {
		trie.Assert(int(i) < m.arity.VectorLength(), "int(i)<m.arity.VectorLength()")
		hashes[i] = c.(vectorCommitment)
	}
	if par.Terminal != nil {
		hashes[m.arity.TerminalCommitmentIndex()] = par.Terminal.(*terminalCommitment).bytes
	}
	hashes[m.arity.PathFragmentCommitmentIndex()] = commitToData(par.PathFragment, m.HashSize)
	return vectorCommitment(hashVector(hashes, m.HashSize))
}

func (m *CommitmentModel) CommitToData(data []byte) trie.TCommitment {
	if len(data) == 0 {
		// empty slice -> no data (deleted)
		return nil
	}
	return commitToTerminal(data, m.HashSize)
}

func (m *CommitmentModel) Description() string {
	return fmt.Sprintf("trie commitment model implementation based on blake2b %s, arity: %s", m.HashSize, m.arity)
}

func (m *CommitmentModel) ShortName() string {
	return fmt.Sprintf("b2b_%s_%s", m.PathArity(), m.HashSize)
}

// NewTerminalCommitment creates empty terminal commitment
func (m *CommitmentModel) NewTerminalCommitment() trie.TCommitment {
	return newTerminalCommitment(m.HashSize)
}

// NewVectorCommitment create empty vector commitment
func (m *CommitmentModel) NewVectorCommitment() trie.VCommitment {
	return newVectorCommitment(m.HashSize)
}

func (m *CommitmentModel) StoreTerminalWithNode(c trie.TCommitment) bool {
	return !m.optimizeTerminals || c.(*terminalCommitment).isHash
}

// *vectorCommitment implements trie_go.VCommitment
var _ trie.VCommitment = &vectorCommitment{}

func newVectorCommitment(sz HashSize) vectorCommitment {
	return make([]byte, sz)
}

func (v vectorCommitment) Bytes() []byte {
	return trie.MustBytes(v)
}

func (v vectorCommitment) Read(r io.Reader) error {
	_, err := r.Read(v)
	return err
}

func (v vectorCommitment) Write(w io.Writer) error {
	_, err := w.Write(v)
	return err
}

func (v vectorCommitment) String() string {
	return hex.EncodeToString(v)
}

func (v vectorCommitment) Clone() trie.VCommitment {
	if len(v) == 0 {
		return nil
	}
	ret := make([]byte, len(v))
	copy(ret, v)
	return vectorCommitment(ret)
}

func (v vectorCommitment) Update(delta trie.VCommitment) {
	m, ok := delta.(vectorCommitment)
	if !ok {
		panic("blake2b hash commitment expected")
	}
	copy(v, m)
}

// *terminalCommitment implements trie_go.TCommitment
var _ trie.TCommitment = &terminalCommitment{}

func newTerminalCommitment(sz HashSize) *terminalCommitment {
	// all 0 non hashed value
	return &terminalCommitment{
		bytes:    make([]byte, sz),
		isHash:   false,
		hashSize: sz,
	}
}

func (t *terminalCommitment) Write(w io.Writer) error {
	trie.Assert(len(t.bytes) <= int(t.hashSize), "len(t.bytes)<=hasSize")
	l := byte(len(t.bytes))
	if t.isHash {
		l = byte(t.hashSize) + 1
	}
	if err := trie.WriteByte(w, l); err != nil {
		return err
	}
	_, err := w.Write(t.bytes)
	return err
}

func (t *terminalCommitment) Read(r io.Reader) error {
	var err error
	var l byte
	if l, err = trie.ReadByte(r); err != nil {
		return err
	}
	if l > byte(t.hashSize)+1 {
		return fmt.Errorf("terminal commitment size byte must be <= %d", byte(t.hashSize)+1)
	}
	t.isHash = l == byte(t.hashSize)+1
	if t.isHash {
		l = byte(t.hashSize)
	}
	if len(t.bytes) < int(l) {
		t.bytes = make([]byte, l)
	}
	n, err := r.Read(t.bytes[:l])
	if err != nil {
		return err
	}
	if n != int(l) {
		return errors.New("bad data length")
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

func commitToData(data []byte, sz HashSize) []byte {
	if len(data) <= int(sz) {
		ret := make([]byte, sz)
		copy(ret[:], data)
		return ret
	}
	return blakeIt(data, sz)
}

func blakeIt(data []byte, sz HashSize) []byte {
	switch sz {
	case HashSize160:
		ret := trie.Blake2b160(data)
		return ret[:]
	case HashSize256:
		ret := blake2b.Sum256(data)
		return ret[:]
	}
	panic("must be 160 of 256")
}

func commitToTerminal(data []byte, sz HashSize) *terminalCommitment {
	ret := &terminalCommitment{
		bytes:    commitToData(data, sz),
		hashSize: sz,
	}
	ret.isHash = len(data) > int(sz)
	return ret
}

func hashVector(hashes [][]byte, sz HashSize) []byte {
	buf := make([]byte, len(hashes)*int(sz))
	for i, h := range hashes {
		if h == nil {
			continue
		}
		pos := int(sz) * i
		copy(buf[pos:pos+int(sz)], h)
	}
	return blakeIt(buf, sz)
}
