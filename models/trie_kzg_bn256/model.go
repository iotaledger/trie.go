package trie_kzg_bn256

import (
	"bytes"
	"io"

	"github.com/iotaledger/trie.go/common"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	"golang.org/x/crypto/blake2b"
)

type terminalCommitment struct {
	kyber.Scalar
}

type vectorCommitment struct {
	kyber.Point
}

// *vectorCommitment implements trie_go.VCommitment
var _ common.VCommitment = &vectorCommitment{}

func (v *vectorCommitment) Bytes() []byte {
	return common.MustBytes(v)
}

func (v *vectorCommitment) Read(r io.Reader) error {
	_, err := v.Point.UnmarshalFrom(r)
	return err
}

func (v *vectorCommitment) Write(w io.Writer) error {
	_, err := v.Point.MarshalTo(w)
	return err
}

func (v *vectorCommitment) AsKey() []byte {
	if common.IsNil(v.Point) {
		return nil
	}
	b, err := v.Point.MarshalBinary()
	if err != nil {
		panic(err)
	}
	ret := blake2b.Sum256(b)
	return ret[:]
}

func (v *vectorCommitment) String() string {
	return v.Point.String()
}

func (v *vectorCommitment) Clone() common.VCommitment {
	if v == nil {
		return nil
	}
	return &vectorCommitment{Point: v.Point.Clone()}
}

func (v *vectorCommitment) Equals(other common.VCommitment) bool {
	o, ok := other.(*vectorCommitment)
	if !ok {
		return false
	}
	return v.Point.Equal(o.Point)
}

// *terminalCommitment implements trie_go.TCommitment
var _ common.TCommitment = &terminalCommitment{}

func (t *terminalCommitment) Write(w io.Writer) error {
	_, err := t.Scalar.MarshalTo(w)
	return err
}

func (t *terminalCommitment) Read(r io.Reader) error {
	_, err := t.Scalar.UnmarshalFrom(r)
	return err
}

func (t *terminalCommitment) Bytes() []byte {
	return common.MustBytes(t)
}

func (t *terminalCommitment) String() string {
	return t.Scalar.String()
}

func (t *terminalCommitment) Clone() common.TCommitment {
	if t == nil {
		return nil
	}
	return &terminalCommitment{Scalar: t.Scalar.Clone()}
}

func (t *terminalCommitment) AsKey() []byte {
	return t.Bytes()
}

func (t *terminalCommitment) ExtractValue() ([]byte, bool) {
	return nil, false
}

// CommitmentModel implements 256+ trie based on blake2b hashing
type CommitmentModel struct {
	TrustedSetup
}

// Model is a singleton
var Model = New()

func New() *CommitmentModel {
	ret, err := TrustedSetupFromBytes(bn256.NewSuite(), GetTrustedSetupBin())
	if err != nil {
		panic(err)
	}
	return &CommitmentModel{
		TrustedSetup: *ret,
	}
}

func (m *CommitmentModel) PathArity() common.PathArity {
	return common.PathArity256 // only can be used with 256-ary
}

func (m *CommitmentModel) EqualCommitments(c1, c2 common.Serializable) bool {
	return equalCommitments(c1, c2)
}

func equalCommitments(c1, c2 common.Serializable) bool {
	if equals, conclusive := common.CheckNils(c1, c2); conclusive {
		return equals
	}
	// both not nils
	return bytes.Equal(c1.Bytes(), c2.Bytes())
}

func (m *CommitmentModel) Description() string {
	return "trie commitment common implementation based on KZG (Kate) polynomial commitments and bn256 curve frm Dedis.Kyber library. 256-ary keys"
}

func (m *CommitmentModel) ShortName() string {
	return "kzg-bn256"
}

func (m *CommitmentModel) NewVectorCommitment() common.VCommitment {
	return m.newVectorCommitment()
}

func (m *CommitmentModel) ForceStoreTerminalWithNode(_ common.TCommitment) bool {
	return true
}

func (m *CommitmentModel) AlwaysStoreTerminalWithNode() bool {
	return true
}

func (m *CommitmentModel) ExtractDataFromTCommitment(c common.TCommitment) ([]byte, bool) {
	return nil, common.IsNil(c)
}

func (m *CommitmentModel) newVectorCommitment(p ...kyber.Point) *vectorCommitment {
	if len(p) == 0 {
		return &vectorCommitment{Point: m.Suite.G1().Point()}
	}
	return &vectorCommitment{Point: p[0]}
}

func (m *CommitmentModel) NewTerminalCommitment() common.TCommitment {
	return m.newTerminalCommitment()
}

func (m *CommitmentModel) newTerminalCommitment() *terminalCommitment {
	return &terminalCommitment{Scalar: m.Suite.G1().Scalar()}
}

func (m *CommitmentModel) CommitToData(data []byte) common.TCommitment {
	return commitToData(data, m.Suite)
}

func (m *CommitmentModel) UpdateVCommitment(c *common.VCommitment, delta common.VCommitment) {
	if *c == nil {
		*c = m.newVectorCommitment()
	}
	p := (*c).(*vectorCommitment).Point
	p.Add(p, delta.(*vectorCommitment).Point)
}

// UpdateNodeCommitment updates mutated part of node's data and, optionaly, upper
func (m *CommitmentModel) UpdateNodeCommitment(mutate *common.NodeData, childUpdates map[byte]common.VCommitment, terminal common.TCommitment, pathFragment []byte, calcDelta bool) {
	var deltas map[int]kyber.Scalar

	if calcDelta {
		deltas = make(map[int]kyber.Scalar)
	}

	for i, childUpd := range childUpdates {
		if calcDelta {
			prevC, existsPrevC := mutate.ChildCommitments[i]
			if !existsPrevC && childUpd == nil {
				// child didn't exist, no need to delete it
				continue
			}
			var delta kyber.Scalar
			//delta := m.TrustedSetup.Suite.G1().Scalar().Zero() // TODO: the value is not used. Remove the line?
			if childUpd == nil {
				// deleting child
				common.Assert(prevC != nil, "prevC != nil")
				common.Assert(existsPrevC, "par.ChildCommitments[i] != nil")
				delta = scalarFromPoint(m.TrustedSetup.Suite.G1().Scalar(), prevC.(*vectorCommitment).Point)
				delta.Neg(delta)
			} else {
				delta = scalarFromPoint(m.TrustedSetup.Suite.G1().Scalar(), childUpd.(*vectorCommitment).Point)
				if prevC != nil {
					prevS := scalarFromPoint(m.TrustedSetup.Suite.G1().Scalar(), prevC.(*vectorCommitment).Point)
					delta.Sub(delta, prevS)
				}
			}
			deltas[int(i)] = delta
		}
		// update mutated part
		if childUpd == nil {
			delete(mutate.ChildCommitments, i)
		} else {
			mutate.ChildCommitments[i] = childUpd
		}
	}
	if calcDelta && !equalCommitments(mutate.Terminal, terminal) {
		delta := m.TrustedSetup.Suite.G1().Scalar().Zero()
		if terminal == nil {
			if mutate.Terminal != nil {
				delta = mutate.Terminal.(*terminalCommitment).Scalar
				delta.Neg(delta)
			}
		} else {
			delta.Set(terminal.(*terminalCommitment).Scalar)
			if mutate.Terminal != nil {
				delta.Sub(terminal.(*terminalCommitment).Scalar, mutate.Terminal.(*terminalCommitment).Scalar)
			}
		}
		deltas[256] = delta
	}
	mutate.Terminal = terminal
	mutate.PathFragment = pathFragment
	if calcDelta {
		var prevP kyber.Point

		// update upper commitment by adding calculated delta
		if !common.IsNil(mutate.Commitment) {
			prevP = mutate.Commitment.(*vectorCommitment).Point.Clone()
		} else {
			prevP = m.TrustedSetup.Suite.G1().Point().Null()
		}
		elem := m.TrustedSetup.Suite.G1().Point()
		for i, deltaS := range deltas {
			elem.Mul(deltaS, m.TrustedSetup.LagrangeBasis[i])
			prevP.Add(prevP, elem)
		}
		mutate.Commitment = m.newVectorCommitment(prevP)
	} else {
		mutate.Commitment = m.CalcNodeCommitment(mutate)
	}
}

func (m *CommitmentModel) CalcNodeCommitment(data *common.NodeData) common.VCommitment {
	return m.calcNodeCommitment(data)
}

func (m *CommitmentModel) calcNodeCommitment(data *common.NodeData) *vectorCommitment {
	var vect [258]kyber.Scalar
	makeVector(data, &m.TrustedSetup, &vect)
	return &vectorCommitment{Point: m.TrustedSetup.commit(vect[:])}
}

func (m *CommitmentModel) calcProof(data *common.NodeData, index int) kyber.Point {
	var vect [258]kyber.Scalar
	makeVector(data, &m.TrustedSetup, &vect)
	return m.TrustedSetup.prove(vect[:], index)
}

// Vector extracts vector from the node
func makeVector(n *common.NodeData, ts *TrustedSetup, ret *[258]kyber.Scalar) {
	for i, p := range n.ChildCommitments {
		if p == nil {
			continue
		}
		ret[i] = ts.Suite.G1().Scalar()
		scalarFromPoint(ret[i], p.(*vectorCommitment).Point)
	}
	if n.Terminal != nil {
		ret[256] = n.Terminal.(*terminalCommitment).Scalar
	}
	h := blake2b.Sum256(n.PathFragment)
	ret[257] = ts.Suite.G1().Scalar()
	scalarFromBytes(ret[257], h[:])
}

// scalarFromPoint hashes the point and make a scalar from hash
// Note that zero-point does not result in zero scalar
func scalarFromPoint(ret kyber.Scalar, point kyber.Point) kyber.Scalar {
	if point == nil {
		ret.Zero()
		return ret
	}
	pBin, err := point.MarshalBinary()
	if err != nil {
		panic(err)
	}
	scalarFromBytes(ret, pBin)
	return ret
}

func scalarFromBytes(ret kyber.Scalar, data []byte) kyber.Scalar {
	h := blake2b.Sum256(data)
	ret.SetBytes(h[:])
	return ret
}

func commitToData(data []byte, suite *bn256.Suite) common.TCommitment {
	if len(data) == 0 {
		return nil
	}
	h := blake2b.Sum256(data)
	ret := &terminalCommitment{Scalar: suite.G1().Scalar()}
	ret.Scalar.SetBytes(h[:])
	return ret
}
