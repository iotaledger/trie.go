package trie

import (
	"fmt"
)

// CommitmentModel abstracts 256+ Trie logic from the commitment logic/cryptography
type CommitmentModel interface {
	// PathArity is used by implementations to optimize operations
	PathArity() PathArity
	// NewVectorCommitment creates empty trie_go.VCommitment
	NewVectorCommitment() VCommitment
	// NewTerminalCommitment creates empty trie_go.TCommitment
	NewTerminalCommitment() TCommitment
	// CommitToData calculates terminal commitment to an arbitrary data
	CommitToData([]byte) TCommitment
	// CalcNodeCommitment calculates commitment of the node data
	CalcNodeCommitment(*NodeData) VCommitment
	// UpdateNodeCommitment updates mutable NodeData with the update information.
	// It also (optionally, if 'update' != nil) updates previous commitment to the node
	// If update != nil and *update != nil, parameter calcDelta specifies if commitment is calculated
	// from scratch using CalcNodeCommitment, or it can be calculated by applying additive delta
	// I can be used by implementation to optimize the computation of update. For examples KZG implementation
	// can be made dramatically faster this way than strictly computing each time whole expensive vector commitment
	// This interface takes into account different ways how updates are propagated in the trie
	UpdateNodeCommitment(mutate *NodeData, childUpdates map[byte]VCommitment, calcDelta bool, terminal TCommitment, update *VCommitment)
	// Description return description of the implementation
	Description() string
	// ShortName short name
	ShortName() string
}
type PathArity byte

const (
	PathArity256 = PathArity(255)
	PathArity16  = PathArity(15)
	PathArity2   = PathArity(1)
)

func (a PathArity) String() string {
	switch a {
	case PathArity256, PathArity16, PathArity2:
		return fmt.Sprintf("PathArity%d", int(a)+1)
	default:
		return "PathArity(wrong)"
	}
}

func (a PathArity) TerminalCommitmentIndex() int {
	switch a {
	case PathArity256:
		return 256
	case PathArity16:
		return 16
	case PathArity2:
		return 2
	}
	panic("wrong path arity")
}

func (a PathArity) PathFragmentCommitmentIndex() int {
	return a.TerminalCommitmentIndex() + 1
}

func (a PathArity) VectorLength() int {
	return int(a) + 3
}

func (a PathArity) IsChildIndex(i int) bool {
	return i <= int(a)
}

func (a PathArity) NumChildren() int {
	return int(a) + 1
}
