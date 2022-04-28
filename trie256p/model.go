package trie256p

import (
	trie_go "github.com/iotaledger/trie.go"
)

// CommitmentModel abstracts 256+ Trie logic from the commitment logic/cryptography
type CommitmentModel interface {
	// NewVectorCommitment creates empty trie_go.VCommitment
	NewVectorCommitment() trie_go.VCommitment
	// NewTerminalCommitment creates empty trie_go.TCommitment
	NewTerminalCommitment() trie_go.TCommitment
	// CommitToData calculates terminal commitment to an arbitrary data
	CommitToData([]byte) trie_go.TCommitment
	// CalcNodeCommitment calculates commitment of the node data
	CalcNodeCommitment(*NodeData) trie_go.VCommitment
	// UpdateNodeCommitment updates mutable NodeData with the update information.
	// It also (optionally, if 'update' != nil) updates previous commitment to the node
	// If update != nil and *update != nil, parameter calcDelta specifies if commitment is calculated
	// from scratch using CalcNodeCommitment, or it can be calculated by applying additive delta
	// I can be used by implementation to optimize the computation of update. For example KZG implementation
	// can be made dramatically faster this way than strictly computing each time whole expensive vector commitment
	// This interface takes into account different ways how updates are propagated in the trie
	UpdateNodeCommitment(mutate *NodeData, childUpdates map[byte]trie_go.VCommitment, calcDelta bool, terminal trie_go.TCommitment, update *trie_go.VCommitment)
	// GetOptions returns optimization options
	GetOptions() Options
	// Description return description of the implementation
	Description() string
}

type Options struct {
	// is true, key commitments won't be optimized when serializing the trie node
	// Makes sense when key commitments are rare.
	// Default is 'enabled'
	DisableKeyCommitmentOptimization bool
	// if true, provided keys are 'hexarized' with subsequent optimization
	// It makes proofs in the 'blake2b' trie approx 8 times smaller (2 times longer and 16 times narrower)
	// At the expense of some database size overhead.
	// Default is disabled TODO WIP
	UseHexaryPath bool
}
