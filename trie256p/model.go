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
	UpdateNodeCommitment(mutate *NodeData, childUpdates map[byte]trie_go.VCommitment, calcDelta bool, terminal trie_go.TCommitment, update *trie_go.VCommitment)
}
