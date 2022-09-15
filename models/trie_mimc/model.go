// Package trie_mimc implements trie.CommitmentModel based on mimc 32-byte hashing
package trie_mimc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/iotaledger/trie.go/trie"
)

// terminalCommitment commits to the data of arbitrary size.
// len(bytes) can't be > 32
// if isHash == true, len(bytes) must be 32
// otherwise it is not hashed value, mus be len(bytes) <= 32
type terminalCommitment struct {
	bytes              []byte
	isCostlyCommitment bool
}

// vectorCommitment is a blake2b hash of the vector elements
type vectorCommitment []byte

type HashSize byte

const (
	HashSize256 = HashSize(32)
)

var AllHashSize = []HashSize{HashSize256}

func (hs HashSize) MaxCommitmentSize() int {
	return int(hs) + 1
}

func (hs HashSize) String() string {
	switch hs {
	case HashSize256:
		return "HashSize(256)"
	}
	panic("wrong hash size")
}

// CommitmentModel provides commitment model implementation for the 256+ trie
type CommitmentModel struct {
	hashSize                       HashSize
	arity                          trie.PathArity
	valueSizeOptimizationThreshold int
}

// New creates new CommitmentModel.
// Parameter valueSizeOptimizationThreshold means that for terminal commitments to values
// longer than threshold, the terminal commitments will always be stored with the trie node,
// i.e. ForceStoreTerminalWithNode will return true. For terminal commitments
// of this or smaller size, the choice depends on the trie setup
// Default valueSizeOptimizationThreshold = 0, which means that by default all
// values are stored in the node.
// If valueSizeOptimizationThreshold > 0 valueStore must be specified in the trie parameters
// Reasonable value of valueSizeOptimizationThreshold, allows significantly optimize trie storage without
// requiring hashing big data each time
func New(arity trie.PathArity, hashSize HashSize, valueSizeOptimizationThreshold ...int) *CommitmentModel {
	t := 0
	if len(valueSizeOptimizationThreshold) > 0 {
		t = valueSizeOptimizationThreshold[0]
	}
	return &CommitmentModel{
		hashSize:                       hashSize,
		arity:                          arity,
		valueSizeOptimizationThreshold: t,
	}
}

func (m *CommitmentModel) PathArity() trie.PathArity {
	return m.arity
}

func (m *CommitmentModel) HashSize() HashSize {
	return m.hashSize
}
func (m *CommitmentModel) EqualCommitments(c1, c2 trie.Serializable) bool {
	return equalCommitments(c1, c2)
}

func equalCommitments(c1, c2 trie.Serializable) bool {
	if equals, conclusive := trie.CheckNils(c1, c2); conclusive {
		return equals
	}
	// both not nils
	if t1, ok1 := c1.(*terminalCommitment); ok1 {
		if t2, ok2 := c2.(*terminalCommitment); ok2 {
			return bytes.Equal(t1.bytes, t2.bytes)
		}
	}
	if v1, ok1 := c1.(vectorCommitment); ok1 {
		if v2, ok2 := c2.(vectorCommitment); ok2 {
			return bytes.Equal(v1, v2)
		}
	}
	return false
}

// UpdateNodeCommitment computes update to the node data and, optionally, updates existing commitment
// In mimc implementation delta it just means computing the hash of data
func (m *CommitmentModel) UpdateNodeCommitment(mutate *trie.NodeData, childUpdates map[byte]trie.VCommitment, _ bool, newTerminalUpdate trie.TCommitment, update *trie.VCommitment) {
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
	mutate.Terminal = newTerminalUpdate // for hash commitment just replace
	if len(mutate.ChildCommitments) == 0 && mutate.Terminal == nil {
		return
	}
	if update != nil {
		*update = (vectorCommitment)(HashTheVector(m.makeHashVector(mutate), m.arity, m.hashSize))
	}
}

// CalcNodeCommitment computes commitment of the node. It is suboptimal in KZG trie.
// Used in computing root commitment
func (m *CommitmentModel) CalcNodeCommitment(par *trie.NodeData) trie.VCommitment {
	if len(par.ChildCommitments) == 0 && par.Terminal == nil {
		return nil
	}
	return vectorCommitment(HashTheVector(m.makeHashVector(par), m.arity, m.hashSize))
}

func (m *CommitmentModel) CommitToData(data []byte) trie.TCommitment {
	if len(data) == 0 {
		// empty slice -> no data (deleted)
		return nil
	}
	return m.commitToData(data)
}

func (m *CommitmentModel) Description() string {
	return fmt.Sprintf("trie commitment model implementation based on mimc %s, arity: %s, terminal optimization threshold: %d",
		m.hashSize, m.arity, m.valueSizeOptimizationThreshold)
}

func (m *CommitmentModel) ShortName() string {
	return fmt.Sprintf("mimc_%s_%s", m.PathArity(), m.hashSize)
}

// NewTerminalCommitment creates empty terminal commitment
func (m *CommitmentModel) NewTerminalCommitment() trie.TCommitment {
	return newTerminalCommitment(m.hashSize)
}

// NewVectorCommitment create empty vector commitment
func (m *CommitmentModel) NewVectorCommitment() trie.VCommitment {
	return newVectorCommitment(m.hashSize)
}

func (m *CommitmentModel) ForceStoreTerminalWithNode(c trie.TCommitment) bool {
	return c.(*terminalCommitment).isCostlyCommitment
}

// CommitToDataRaw commits to data
func CommitToDataRaw(data []byte, sz HashSize) []byte {
	var ret []byte
	if len(data) <= int(sz) {
		ret = make([]byte, len(data))
		copy(ret, data)
	} else {
		ret = mimcIt(data, sz)
	}
	return ret
}

func (m *CommitmentModel) commitToData(data []byte) *terminalCommitment {
	return &terminalCommitment{
		bytes:              CommitToDataRaw(data, m.hashSize),
		isCostlyCommitment: len(data) > m.valueSizeOptimizationThreshold,
	}
}

func mimcIt(data []byte, sz HashSize) []byte {
	switch sz {
	case HashSize256:
		h := mimc.NewMiMC()
		h.Write(data)
		ret := h.Sum(nil)
		return ret[:]
	}
	panic("must be 256")
}

func MIMCIt(data []byte, sz HashSize) []byte {
	switch sz {
	case HashSize256:
		h := mimc.NewMiMC()
		h.Write(data)
		ret := h.Sum(nil)
		return ret[:]
	}
	panic("must be 256")
}

// makeHashVector makes the node vector to be hashed. Missing children are nil
func (m *CommitmentModel) makeHashVector(nodeData *trie.NodeData) [][]byte {
	hashes := make([][]byte, m.arity.VectorLength())
	for i, c := range nodeData.ChildCommitments {
		trie.Assert(int(i) < m.arity.VectorLength(), "int(i)<m.arity.VectorLength()")
		hashes[i] = c.Bytes()
	}
	if nodeData.Terminal != nil {
		hashes[m.arity.TerminalCommitmentIndex()] = nodeData.Terminal.(*terminalCommitment).bytes
	}
	hashes[m.arity.PathFragmentCommitmentIndex()] = CommitToDataRaw(nodeData.PathFragment, m.hashSize)
	return hashes
}

func HashTheVector(hashes [][]byte, arity trie.PathArity, sz HashSize) []byte {
	msz := sz.MaxCommitmentSize()
	buf := make([]byte, arity.VectorLength()*msz)
	for i, h := range hashes {
		if h == nil {
			continue
		}
		pos := i * msz
		copy(buf[pos:pos+msz], h)
	}
	// fmt.Printf("buf = %v, sz = %d \n", buf, sz)
	return mimcIt(buf, sz)
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
		panic("mimc hash commitment expected")
	}
	copy(v, m)
}

// *terminalCommitment implements trie_go.TCommitment
var _ trie.TCommitment = &terminalCommitment{}

func newTerminalCommitment(sz HashSize) *terminalCommitment {
	// all 0 non hashed value
	return &terminalCommitment{
		bytes:              make([]byte, 0, sz),
		isCostlyCommitment: false,
	}
}

const (
	sizeMask             = uint8(0x3F)
	costlyCommitmentMask = ^sizeMask
)

func (t *terminalCommitment) Write(w io.Writer) error {
	trie.Assert(len(t.bytes) <= 32, "len(t.bytes) <= 32")
	l := byte(len(t.bytes))
	if t.isCostlyCommitment {
		l |= costlyCommitmentMask
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
	t.isCostlyCommitment = (l & costlyCommitmentMask) != 0
	l &= sizeMask

	if l > 32 {
		return fmt.Errorf("wrong data size")
	}
	if l > 0 {
		t.bytes = make([]byte, l)

		n, err := r.Read(t.bytes)
		if err != nil {
			return err
		}
		if n != int(l) {
			return errors.New("bad data length")
		}
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
