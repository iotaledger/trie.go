package trie_kzg_bn256

import (
	"bytes"
	"fmt"
	"github.com/iotaledger/trie.go/trie"
	"go.dedis.ch/kyber/v3"
	"golang.org/x/xerrors"
	"io"
)

type ProofElement struct {
	// commitment to the vector (node)
	C kyber.Point
	// index of the vector element. 256 mean terminal, 257 means path fragment
	VectorIndex uint16
	// proof that the committed value is at the position VectorIndex of the committed vector
	// If 0 <= VectorIndex <= 255 the value will be of the commitment to the next vector in the path
	// If VectorIndex == 256, the value is the commitment to the terminal value of the key. It is valid only in the
	// last element of the proof path
	// values >=257 are not correct for the proof of inclusion
	Proof kyber.Point
}

// ProofOfInclusion is valid only if the key is present in the trie.
type ProofOfInclusion struct {
	// key of the proof
	Key []byte
	// commitment to the terminal value
	Terminal kyber.Scalar
	// path of proof elements
	Path []*ProofElement
}

// ProofOfPath is a proof of some existing path in the state, which also proves absence of some key
type ProofOfPath struct {
	// TODO not implemented
}

func ProofOfInclusionFromBytes(data []byte) (*ProofOfInclusion, error) {
	ret := &ProofOfInclusion{}
	if err := ret.Read(bytes.NewReader(data)); err != nil {
		return nil, err
	}
	return ret, nil
}

// ProofOfInclusion converts generic proof path of existing key to the verifiable proof path
// Returns nil, false if path does not exist
func (m *CommitmentModel) ProofOfInclusion(key []byte, tr trie.NodeStore) (*ProofOfInclusion, bool) {
	trie.Assert(tr.PathArity() == trie.PathArity256, "for KZG commitment model only 256-ary trie is supported")

	proofGeneric := trie.GetProofGeneric(tr, key)
	if proofGeneric == nil || len(proofGeneric.Path) == 0 || proofGeneric.Ending != trie.EndingTerminal {
		// key is not present in the state
		return nil, false
	}
	// key is present in the state
	ret := &ProofOfInclusion{
		Key:      proofGeneric.Key,
		Terminal: m.TrustedSetup.Suite.G1().Scalar(),
		Path:     make([]*ProofElement, len(proofGeneric.Path)),
	}

	proofLength := len(proofGeneric.Path)
	nodes := make([]*trie.NodeData, proofLength)

	for i, k := range proofGeneric.Path {
		n, ok := tr.GetNode(k)
		trie.Assert(ok, "can't find node with key '%x'", k)

		nodes[i] = &trie.NodeData{
			PathFragment:     n.PathFragment(),
			ChildCommitments: n.ChildCommitments(),
			Terminal:         n.Terminal(),
		}
		ret.Path[i] = &ProofElement{}
		if i == proofLength-1 {
			ret.Path[i].VectorIndex = 256
		} else {
			nextKey := proofGeneric.Path[i+1]
			ret.Path[i].VectorIndex = uint16(nextKey[len(nextKey)-1])
		}
	}

	for i, n := range nodes {
		ret.Path[i].C = m.calcNodeCommitment(n).Point
		//if i == 0 || i == proofLength-1 {
		//	ret.Path[i].C = m.calcNodeCommitment(n).Point
		//} else {
		//	nextC, ok := nodes[i-1].ChildCommitments[byte(ret.Path[i].VectorIndex)]
		//	trie_go.Assert(ok, "can't find commitment at path index %d and child index %d", i-1, ret.Path[i].VectorIndex)
		//	ret.Path[i].C = nextC.(*vectorCommitment).Point
		//}
		ret.Path[i].Proof = m.calcProof(nodes[i], int(ret.Path[i].VectorIndex))
	}

	ret.Terminal.Set(nodes[proofLength-1].Terminal.(*terminalCommitment).Scalar)
	return ret, true
}

// ProofOfPath returns proof of path along the key, if key is absent. If key is present, it returns nil, false,
// The proof of path can be used as a proof of absence of the key in the state, i.e. to prove that something else is
// committed in the state instead of what should be committed if the key would be present
func (m *CommitmentModel) ProofOfPath(key []byte, tr trie.NodeStore) (*ProofOfPath, bool) {
	panic("implement me")
}

func (p *ProofOfInclusion) Bytes() []byte {
	return trie.MustBytes(p)
}

// Validate check the proof against the provided root commitments
// if 'value' is specified, checks if commitment to that value is the terminal of the last element in path
func (p *ProofOfInclusion) Validate(root trie.VCommitment, value ...[]byte) error {
	if len(value) > 0 {
		ct := commitToData(value[0], Model.Suite)
		if !trie.EqualCommitments(ct, &terminalCommitment{Scalar: p.Terminal}) {
			return xerrors.New("terminal commitment not equal to the provided value")
		}
	}
	if len(p.Path) == 0 {
		return xerrors.New("proof path is empty")
	}
	if !trie.EqualCommitments(root, &vectorCommitment{Point: p.Path[0].C}) {
		return xerrors.New("provided commitment and commitment to the first element are not equal")
	}
	var val kyber.Scalar

	for i := range p.Path {
		if p.Path[i].VectorIndex < 256 {
			val = scalarFromPoint(Model.Suite.G1().Scalar(), p.Path[i+1].C)
		} else {
			val = p.Terminal
		}
		if !Model.verify(p.Path[i].C, p.Path[i].Proof, val, int(p.Path[i].VectorIndex)) {
			return xerrors.New(fmt.Sprintf("proof is invalid at path position %d", i))
		}
	}
	return nil
}

func (p *ProofOfInclusion) Write(w io.Writer) error {
	if err := trie.WriteBytes16(w, p.Key); err != nil {
		return err
	}
	if _, err := p.Terminal.MarshalTo(w); err != nil {
		return err
	}
	if err := trie.WriteUint16(w, uint16(len(p.Path))); err != nil {
		return err
	}
	for _, e := range p.Path {
		if err := e.Write(w); err != nil {
			return err
		}
	}
	return nil
}

func (p *ProofOfInclusion) Read(r io.Reader) error {
	var err error
	if p.Key, err = trie.ReadBytes16(r); err != nil {
		return err
	}
	p.Terminal = Model.Suite.G1().Scalar()
	if _, err = p.Terminal.UnmarshalFrom(r); err != nil {
		return err
	}
	var size uint16
	if err = trie.ReadUint16(r, &size); err != nil {
		return err
	}
	p.Path = make([]*ProofElement, size)
	for i := range p.Path {
		p.Path[i] = &ProofElement{}
		if err = p.Path[i].Read(r); err != nil {
			return err
		}
	}
	return nil
}

func (e *ProofElement) Write(w io.Writer) error {
	if _, err := e.C.MarshalTo(w); err != nil {
		return err
	}
	if err := trie.WriteUint16(w, e.VectorIndex); err != nil {
		return err
	}
	if _, err := e.Proof.MarshalTo(w); err != nil {
		return err
	}
	return nil
}

func (e *ProofElement) Read(r io.Reader) error {
	e.C = Model.Suite.G1().Point()
	if _, err := e.C.UnmarshalFrom(r); err != nil {
		return err
	}
	if err := trie.ReadUint16(r, &e.VectorIndex); err != nil {
		return err
	}
	e.Proof = Model.Suite.G1().Point()
	if _, err := e.Proof.UnmarshalFrom(r); err != nil {
		return err
	}
	return nil
}

func (p *ProofOfInclusion) String() string {
	ret := fmt.Sprintf("KZG PROOF: key: %s, term: %s\n", string(p.Key), p.Terminal)
	for i, e := range p.Path {
		ret += fmt.Sprintf("%d:\n%s\n", i, e.String())
	}
	return ret
}

func (e *ProofElement) String() string {
	return fmt.Sprintf("     C: %s\n     idx: %d\n     P: %s", e.C, e.VectorIndex, e.Proof)
}
