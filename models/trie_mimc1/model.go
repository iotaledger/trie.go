// Package trie_mimc implements trie.CommitmentModel based on mimc 32-byte hashing
package trie_mimc1

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/iotaledger/trie.go/trie"
)

const HashSize = 32

// terminalCommitment commits to the data of arbitrary size.
// len(bytes) can't be > 32
// if isHash == true, len(bytes) must be 32
// otherwise it is not hashed value, mus be len(bytes) <= 32
type terminalCommitment struct {
	rawCommitment      []byte
	isCostlyCommitment bool
}

// vectorCommitment is a MIMC hash of the vector elements
type vectorCommitment [HashSize]byte

// CommitmentModel provides commitment model implementation for the 256+ trie
// valueSizeOptimizationThreshold means that for terminal commitments to values
// longer than threshold, the terminal commitments will always be stored with the trie node,
// i.e. ForceStoreTerminalWithNode will return true. For terminal commitments
// of this or smaller size, the choice depends on the trie setup
// Default valueSizeOptimizationThreshold = 0, which means that by default all
// values are stored in the node.
// If valueSizeOptimizationThreshold > 0 valueStore must be specified in the trie parameters
// Reasonable value of valueSizeOptimizationThreshold, allows significantly optimize trie storage without
// requiring hashing big data each time
type CommitmentModel struct {
	arity                          trie.PathArity
	valueSizeOptimizationThreshold int
}

// New creates new CommitmentModel.
func New(arity trie.PathArity, valueSizeOptimizationThreshold ...int) *CommitmentModel {
	t := 0
	if len(valueSizeOptimizationThreshold) > 0 {
		t = valueSizeOptimizationThreshold[0]
	}
	return &CommitmentModel{
		arity:                          arity,
		valueSizeOptimizationThreshold: t,
	}
}

func (m *CommitmentModel) PathArity() trie.PathArity {
	return m.arity
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
			return bytes.Equal(t1.rawCommitment, t2.rawCommitment)
		}
	}
	if v1, ok1 := c1.(*vectorCommitment); ok1 {
		if v2, ok2 := c2.(*vectorCommitment); ok2 {
			return *v1 == *v2
		}
	}
	return false
}

// UpdateNodeCommitment computes update to the node data and, optionally, updates existing commitment
// In MIMC implementation delta it just means computing the hash of data
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
		ret := new(vectorCommitment)
		copy((*ret)[:], HashTheVector(m.makeHashVector(mutate), m.arity))
		*update = ret
	}
}

// CalcNodeCommitment computes commitment of the node. It is suboptimal in KZG trie.
// Used in computing root commitment
func (m *CommitmentModel) CalcNodeCommitment(par *trie.NodeData) trie.VCommitment {
	if len(par.ChildCommitments) == 0 && par.Terminal == nil {
		return nil
	}
	ret := new(vectorCommitment)
	copy((*ret)[:], HashTheVector(m.makeHashVector(par), m.arity))
	return ret
}

func (m *CommitmentModel) CommitToData(data []byte) trie.TCommitment {
	if len(data) == 0 {
		// empty slice -> no data (deleted)
		return nil
	}
	return &terminalCommitment{
		rawCommitment:      CommitToDataRaw(data),
		isCostlyCommitment: len(data) > m.valueSizeOptimizationThreshold,
	}
}

func (m *CommitmentModel) Description() string {
	return fmt.Sprintf("trie commitment model implementation based on MIMC (32 bytes), arity: %s, terminal optimization threshold: %d",
		m.arity, m.valueSizeOptimizationThreshold)
}

func (m *CommitmentModel) ShortName() string {
	return "mimc1_" + m.PathArity().String()
}

// NewTerminalCommitment creates empty terminal commitment
func (m *CommitmentModel) NewTerminalCommitment() trie.TCommitment {
	return newTerminalCommitment()
}

// NewVectorCommitment create empty vector commitment
func (m *CommitmentModel) NewVectorCommitment() trie.VCommitment {
	return new(vectorCommitment)
}

func (m *CommitmentModel) ForceStoreTerminalWithNode(c trie.TCommitment) bool {
	return c.(*terminalCommitment).isCostlyCommitment
}

// CommitToDataRaw commits to data. It hashes it only if data size
// exceeds 32 bytes, otherwise raw data is used
func CommitToDataRaw(data []byte) []byte {
	var ret []byte
	if len(data) <= HashSize {
		ret = make([]byte, len(data))
		copy(ret, data)
	} else {
		ret = HashData(data)
	}
	return ret
}

// HashData is MIMC hashing
func HashData(data []byte) []byte {
	h := mimc.NewMiMC()
	h.Write(data)
	ret := h.Sum(nil)
	if len(ret) != HashSize {
		panic("internal inconsistency")
	}
	return ret
}

// makeHashVector makes the node vector to be hashed. Missing children are nil
// For MIMC implementation we make each non-nil vector element 32 byte hash
func (m *CommitmentModel) makeHashVector(nodeData *trie.NodeData) [][]byte {
	hashes := make([][]byte, m.arity.VectorLength())
	for i, c := range nodeData.ChildCommitments {
		trie.Assert(int(i) < m.arity.VectorLength(), "int(i)<m.arity.VectorLength()")
		hashes[i] = c.Bytes()
	}
	if nodeData.Terminal != nil {
		hashes[m.arity.TerminalCommitmentIndex()] = HashData(nodeData.Terminal.(*terminalCommitment).rawCommitment)
	}
	hashes[m.arity.PathFragmentCommitmentIndex()] = HashData(nodeData.PathFragment)
	return hashes
}

// HashTheVector concatenates all hashes and MIMC-hashes them
// nil commitments are treated as 32 byte long all-0 values
func HashTheVector(hashes [][]byte, arity trie.PathArity) []byte {
	buf := make([]byte, arity.VectorLength()*HashSize)
	for i, h := range hashes {
		if h == nil {
			continue
		}
		if len(h) != HashSize {
			panic("HashTheVector: wrong parameter size")
		}
		pos := i * HashSize
		copy(buf[pos:pos+HashSize], h)
	}
	return HashData(buf)
}

// *vectorCommitment implements trie_go.VCommitment
var _ trie.VCommitment = &vectorCommitment{}

func (v *vectorCommitment) Bytes() []byte {
	return (*v)[:]
}

func (v *vectorCommitment) Read(r io.Reader) error {
	n, err := r.Read((*v)[:])
	if err != nil {
		return err
	}
	if n != HashSize {
		return errors.New("wrong data size")
	}
	return nil
}

func (v *vectorCommitment) Write(w io.Writer) error {
	_, err := w.Write((*v)[:])
	return err
}

func (v *vectorCommitment) String() string {
	return hex.EncodeToString((*v)[:])
}

func (v *vectorCommitment) Clone() trie.VCommitment {
	ret := new(vectorCommitment)
	copy((*ret)[:], (*v)[:])
	return ret
}

func (v *vectorCommitment) Update(delta trie.VCommitment) {
	m, ok := delta.(*vectorCommitment)
	if !ok {
		panic("MIMC hash commitment expected")
	}
	*v = *m
}

// *terminalCommitment implements trie_go.TCommitment
var _ trie.TCommitment = &terminalCommitment{}

func newTerminalCommitment() *terminalCommitment {
	// all 0 non hashed value
	return &terminalCommitment{
		rawCommitment:      make([]byte, 0, HashSize),
		isCostlyCommitment: false,
	}
}

const (
	sizeMask             = uint8(0x3F)
	costlyCommitmentMask = ^sizeMask
)

func (t *terminalCommitment) Write(w io.Writer) error {
	trie.Assert(len(t.rawCommitment) <= 32, "len(t.bytes) <= 32")
	l := byte(len(t.rawCommitment))
	if t.isCostlyCommitment {
		l |= costlyCommitmentMask
	}
	if err := trie.WriteByte(w, l); err != nil {
		return err
	}
	_, err := w.Write(t.rawCommitment)
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
		t.rawCommitment = make([]byte, l)

		n, err := r.Read(t.rawCommitment)
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
	return hex.EncodeToString(t.rawCommitment[:])
}

func (t *terminalCommitment) Clone() trie.TCommitment {
	if t == nil {
		return nil
	}
	ret := *t
	return &ret
}
