// Package trie_blake2b_20 implements trie.CommitmentModel based on blake2b 32-byte hashing
package trie_blake2b

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/iotaledger/trie.go/common"
	"golang.org/x/crypto/blake2b"
)

// terminalCommitment commits to the data of arbitrary size.
// len(bytes) can't be > 32
// if isHash == true, len(bytes) must be 32
// otherwise it is not hashed value, mus be len(bytes) <= 32
type terminalCommitment struct {
	bytes               []byte
	isValueInCommitment bool
	isCostlyCommitment  bool
}

// vectorCommitment is a blake2b hash of the vector elements
type vectorCommitment []byte

type HashSize byte

const (
	HashSize160 = HashSize(20)
	HashSize192 = HashSize(24)
	HashSize256 = HashSize(32)
)

var AllHashSize = []HashSize{HashSize160, HashSize256}

func (hs HashSize) MaxCommitmentSize() int {
	return int(hs) + 1
}

func (hs HashSize) String() string {
	switch hs {
	case HashSize256:
		return "HashSize(256)"
	case HashSize160:
		return "HashSize(160)"
	}
	panic("wrong hash size")
}

// CommitmentModel provides commitment common implementation for the 256+ trie
type CommitmentModel struct {
	hashSize                       HashSize
	arity                          common.PathArity
	valueSizeOptimizationThreshold int
}

// New creates new CommitmentModel.
// Parameter valueSizeOptimizationThreshold means that for terminal commitments to values
// longer than threshold, the terminal commitments will always be stored with the trie node,
// i.e. ForceStoreTerminalWithNode will return true. For terminal commitments
// of this or smaller size, the choice depends on the trie setup
// Default valueSizeOptimizationThreshold = 0, which means that by default all
// value commitments are stored in the node.
// If valueSizeOptimizationThreshold > 0 valueStore must be specified in the trie parameters
// Reasonable value of valueSizeOptimizationThreshold, allows significantly optimize trie storage without
// requiring hashing big data each time
func New(arity common.PathArity, hashSize HashSize, valueSizeOptimizationThreshold ...int) *CommitmentModel {
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

func (m *CommitmentModel) PathArity() common.PathArity {
	return m.arity
}

func (m *CommitmentModel) HashSize() HashSize {
	return m.hashSize
}
func (m *CommitmentModel) EqualCommitments(c1, c2 common.Serializable) bool {
	return equalCommitments(c1, c2)
}

func equalCommitments(c1, c2 common.Serializable) bool {
	if equals, conclusive := common.CheckNils(c1, c2); conclusive {
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
// In blake2b implementation delta it just means computing the hash of data
func (m *CommitmentModel) UpdateNodeCommitment(mutate *common.NodeData, childUpdates map[byte]common.VCommitment, newTerminalUpdate common.TCommitment, pathFragment []byte, _ bool) {
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
	mutate.PathFragment = pathFragment
	if len(mutate.ChildCommitments) == 0 && mutate.Terminal == nil {
		return
	}
	mutate.Commitment = (vectorCommitment)(HashTheVector(m.makeHashVector(mutate), m.arity, m.hashSize))
}

// CalcNodeCommitment computes commitment of the node. It is suboptimal in KZG trie.
// Used in computing root commitment
func (m *CommitmentModel) CalcNodeCommitment(par *common.NodeData) common.VCommitment {
	if len(par.ChildCommitments) == 0 && par.Terminal == nil {
		return nil
	}
	return vectorCommitment(HashTheVector(m.makeHashVector(par), m.arity, m.hashSize))
}

func (m *CommitmentModel) CommitToData(data []byte) common.TCommitment {
	if len(data) == 0 {
		// empty slice -> no data (deleted)
		return nil
	}
	return m.commitToData(data)
}

func (m *CommitmentModel) Description() string {
	return fmt.Sprintf("trie commitment common implementation based on blake2b %s, arity: %s, terminal optimization threshold: %d",
		m.hashSize, m.arity, m.valueSizeOptimizationThreshold)
}

func (m *CommitmentModel) ShortName() string {
	return fmt.Sprintf("b2b_%s_%s", m.PathArity(), m.hashSize)
}

// NewTerminalCommitment creates empty terminal commitment
func (m *CommitmentModel) NewTerminalCommitment() common.TCommitment {
	return newTerminalCommitment(m.hashSize)
}

// NewVectorCommitment create empty vector commitment
func (m *CommitmentModel) NewVectorCommitment() common.VCommitment {
	return newVectorCommitment(m.hashSize)
}

func (m *CommitmentModel) AlwaysStoreTerminalWithNode() bool {
	return m.valueSizeOptimizationThreshold == 0
}

func (m *CommitmentModel) ForceStoreTerminalWithNode(c common.TCommitment) bool {
	return m.AlwaysStoreTerminalWithNode() || c.(*terminalCommitment).isCostlyCommitment
}

// CommitToDataRaw commits to data
func CommitToDataRaw(data []byte, sz HashSize) ([]byte, bool) {
	var ret []byte
	valueInCommitment := false
	if len(data) <= int(sz) {
		ret = make([]byte, len(data))
		valueInCommitment = true
		copy(ret, data)
	} else {
		ret = blakeIt(data, sz)
	}
	return ret, valueInCommitment
}

func (m *CommitmentModel) commitToData(data []byte) *terminalCommitment {
	commitmentBytes, isValueInCommitment := CommitToDataRaw(data, m.hashSize)
	return &terminalCommitment{
		bytes:               commitmentBytes,
		isValueInCommitment: isValueInCommitment,
		isCostlyCommitment:  len(data) > m.valueSizeOptimizationThreshold,
	}
}

func blakeIt(data []byte, sz HashSize) []byte {
	switch sz {
	case HashSize160:
		ret := common.Blake2b160(data)
		return ret[:]
	case HashSize192:
		panic("24 byte hashing not implemented")
	case HashSize256:
		ret := blake2b.Sum256(data)
		return ret[:]
	}
	panic("must be 160 of 256")
}

// makeHashVector makes the node vector to be hashed. Missing children are nil
func (m *CommitmentModel) makeHashVector(nodeData *common.NodeData) [][]byte {
	hashes := make([][]byte, m.arity.VectorLength())
	for i, c := range nodeData.ChildCommitments {
		common.Assert(int(i) < m.arity.VectorLength(), "int(i)<m.arity.VectorLength()")
		hashes[i] = c.Bytes()
	}
	if !common.IsNil(nodeData.Terminal) {
		hashes[m.arity.TerminalCommitmentIndex()] = nodeData.Terminal.Bytes()
		//nodeData.Terminal.(*terminalCommitment).bytes
	}
	pathFragmentCommitmentBytes, _ := CommitToDataRaw(nodeData.PathFragment, m.hashSize)
	hashes[m.arity.PathFragmentCommitmentIndex()] = pathFragmentCommitmentBytes
	return hashes
}

func HashTheVector(hashes [][]byte, arity common.PathArity, sz HashSize) []byte {
	msz := sz.MaxCommitmentSize()
	buf := make([]byte, arity.VectorLength()*msz)
	for i, h := range hashes {
		if h == nil {
			continue
		}
		pos := i * msz
		copy(buf[pos:pos+msz], h)
	}
	return blakeIt(buf, sz)
}

// *vectorCommitment implements trie_go.VCommitment
var _ common.VCommitment = &vectorCommitment{}

func newVectorCommitment(sz HashSize) vectorCommitment {
	return make([]byte, sz)
}

func (v vectorCommitment) Bytes() []byte {
	return common.MustBytes(v)
}

func (v vectorCommitment) Read(r io.Reader) error {
	_, err := r.Read(v)
	return err
}

func (v vectorCommitment) Write(w io.Writer) error {
	_, err := w.Write(v)
	return err
}

func (v vectorCommitment) AsKey() []byte {
	return v
}

func (v vectorCommitment) String() string {
	return hex.EncodeToString(v)
}

func (v vectorCommitment) Clone() common.VCommitment {
	if len(v) == 0 {
		return nil
	}
	ret := make([]byte, len(v))
	copy(ret, v)
	return vectorCommitment(ret)
}

func (v vectorCommitment) Update(delta common.VCommitment) {
	m, ok := delta.(vectorCommitment)
	if !ok {
		panic("blake2b hash commitment expected")
	}
	copy(v, m)
}

// *terminalCommitment implements trie_go.TCommitment
var _ common.TCommitment = &terminalCommitment{}

func newTerminalCommitment(sz HashSize) *terminalCommitment {
	// all 0 non hashed value
	return &terminalCommitment{
		bytes:               make([]byte, 0, sz),
		isValueInCommitment: false,
		isCostlyCommitment:  false,
	}
}

const (
	sizeMask              = uint8(0x3F)
	costlyCommitmentMask  = uint8(0x40)
	valueInCommitmentMask = uint8(0x80)
)

func (t *terminalCommitment) Write(w io.Writer) error {
	common.Assert(len(t.bytes) <= 32, "len(t.bytes) <= 32")
	l := byte(len(t.bytes))
	if t.isCostlyCommitment {
		l |= costlyCommitmentMask
	}
	if t.isValueInCommitment {
		l |= valueInCommitmentMask
	}
	if err := common.WriteByte(w, l); err != nil {
		return err
	}
	_, err := w.Write(t.bytes)
	return err
}

func (t *terminalCommitment) Read(r io.Reader) error {
	var err error
	var l byte
	if l, err = common.ReadByte(r); err != nil {
		return err
	}
	t.isCostlyCommitment = (l & costlyCommitmentMask) != 0
	t.isValueInCommitment = (l & valueInCommitmentMask) != 0
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
	return common.MustBytes(t)
}

func (t *terminalCommitment) String() string {
	return hex.EncodeToString(t.bytes[:])
}

func (t *terminalCommitment) Clone() common.TCommitment {
	if t == nil {
		return nil
	}
	ret := *t
	return &ret
}

func (t *terminalCommitment) AsKey() []byte {
	return t.Bytes()
}

func (t *terminalCommitment) ExtractValue() ([]byte, bool) {
	if t.isValueInCommitment {
		return t.bytes, true
	}
	return nil, false
}
