// Package trie256p implements functionality of generic verkle trie with 256 child commitment in each node
// + Terminal commitment + commitment to the path fragment: 258 commitments in total.
// It mainly follows the definition from https://hackmd.io/@Evaldas/H13YFOVGt (except commitment to the path fragment)
// The commitment to the path fragment is needed to provide proofs of absence of keys
//
// The specific implementation of the commitment model is presented as a CommitmentModel interface
package trie256p

import (
	"bytes"
	"fmt"
	trie_go "github.com/iotaledger/trie.go"
	"sort"
)

// newTerminalNode creates new node in the trie with specified PathFragment and Terminal commitment.
// Assumes 'key' does not exist in the Trie
func (tr *Trie) newTerminalNode(key, pathFragment []byte, newTerminal trie_go.TCommitment) *bufferedNode {
	tr.unDelete(key)
	ret := newBufferedNode(key)
	ret.newTerminal = newTerminal
	ret.n.PathFragment = pathFragment
	ret.pathChanged = true
	tr.insertNewNode(ret)
	return ret
}

// Commit calculates a new root commitment value from the cache and commits all mutations in the cached NodeStoreReader
// It is a re-calculation of the trie. bufferedNode caches are updated accordingly.
func (tr *Trie) Commit() {
	tr.CommitNode(nil, nil)
}

// CommitNode re-calculates node commitment and, recursively, its children commitments
// Child modification marks in 'modifiedChildren' are updated
// Return update to the upper commitment. nil mean upper commitment is not updated
// It calls implementation-specific function UpdateNodeCommitment and passes parameter
// calcDelta = true if node's commitment can be updated incrementally. The implementation
// of UpdateNodeCommitment may use this parameter to optimize underlying cryptography
func (tr *Trie) CommitNode(key []byte, update *trie_go.VCommitment) {
	n, ok := tr.getNodeIntern(key)
	if !ok {
		if update != nil {
			*update = nil
		}
		return
	}
	if !n.isModified() {
		return
	}
	mutate := NodeData{
		PathFragment:     n.n.PathFragment,
		ChildCommitments: n.n.ChildCommitments,
		Terminal:         n.n.Terminal,
	}
	childUpdates := make(map[byte]trie_go.VCommitment)
	for childIndex := range n.modifiedChildren {
		childKey := childKey(n, childIndex)
		curCommitment := mutate.ChildCommitments[childIndex] // may be nil
		tr.CommitNode(childKey, &curCommitment)
		childUpdates[childIndex] = curCommitment
	}

	calcDelta := !n.pathChanged && update != nil && *update == nil
	tr.Model().UpdateNodeCommitment(&mutate, childUpdates, calcDelta, n.newTerminal, update)

	n.n.Terminal = n.newTerminal
	if len(n.modifiedChildren) > 0 {
		// clean the modification marks if any
		n.modifiedChildren = make(map[byte]struct{})
	}
	n.pathChanged = false
}

// Update updates Trie with the key/value. Reorganizes and re-calculates trie, keeps cache consistent
func (tr *Trie) Update(key []byte, value []byte) {
	c := tr.nodeStoreReader.model.CommitToData(value)
	if c == nil {
		// nil value means deletion
		tr.Delete(key)
		return
	}
	// find path in the trie corresponding to the key
	proof, lastCommonPrefix, ending := proofPath(tr, key)
	if len(proof) == 0 {
		tr.newTerminalNode(nil, key, c)
		return
	}
	lastKey := proof[len(proof)-1]
	switch ending {
	case EndingTerminal:
		tr.mustGetNode(lastKey).setNewTerminal(c)

	case EndingExtend:
		childIndexPosition := len(lastKey) + len(lastCommonPrefix)
		trie_go.Assert(childIndexPosition < len(key), "childPosition < len(key)")
		childIndex := key[childIndexPosition]
		tr.removeKey(key[:childIndexPosition+1])
		tr.newTerminalNode(key[:childIndexPosition+1], key[childIndexPosition+1:], c)
		tr.mustGetNode(lastKey).markChildModified(childIndex)

	case EndingSplit:
		// splitting the node into two path fragments
		tr.splitNode(key, lastKey, lastCommonPrefix, c)

	default:
		panic("inconsistency: unknown path ending code")
	}
	tr.markModifiedCommitmentsBackToRoot(proof)
}

func (tr *Trie) splitNode(fullKey, lastKey, commonPrefix []byte, newTerminal trie_go.TCommitment) {
	splitIndex := len(commonPrefix)
	childPosition := len(lastKey) + splitIndex
	trie_go.Assert(childPosition <= len(fullKey), "childPosition <= len(fullKey)")

	n := tr.mustGetNode(lastKey)

	keyNewNode := make([]byte, childPosition+1)
	copy(keyNewNode, fullKey)
	trie_go.Assert(splitIndex < len(n.n.PathFragment), "splitIndex < len(n.newPathFragment)")
	childContinue := n.n.PathFragment[splitIndex]
	keyNewNode[len(keyNewNode)-1] = childContinue

	// create new node with keyNewNode, move everything from old to the new node
	// Only path fragment and key changes
	newNode := n.Clone() // children and Terminal remains the same, PathFragment changes
	newNode.setNewKey(keyNewNode)
	newNode.setNewPathFragment(n.PathFragment()[splitIndex+1:])
	tr.insertNewNode(newNode)

	// modify the node under the old key
	n.setNewPathFragment(commonPrefix)
	n.n.ChildCommitments = make(map[uint8]trie_go.VCommitment)
	n.modifiedChildren = make(map[uint8]struct{})
	n.markChildModified(childContinue)
	n.n.Terminal = nil
	n.newTerminal = nil

	// insert Terminal
	if childPosition == len(fullKey) {
		// no need for the new node
		n.newTerminal = newTerminal
	} else {
		// create a new node
		keyFork := fullKey[:len(keyNewNode)]
		childForkIndex := keyFork[len(keyFork)-1]
		trie_go.Assert(childForkIndex != childContinue, "childForkIndex != childContinue")
		tr.newTerminalNode(keyFork, fullKey[len(keyFork):], newTerminal)
		n.markChildModified(childForkIndex)
	}
}

// Delete deletes Key/value from the Trie, reorganizes the trie
func (tr *Trie) Delete(key []byte) {
	proof, _, ending := proofPath(tr, key)
	if len(proof) == 0 || ending != EndingTerminal {
		return
	}
	lastKey := proof[len(proof)-1]
	lastNode, ok := tr.getNodeIntern(lastKey)
	if !ok {
		return
	}
	lastNode.setNewTerminal(nil)
	reorg, mergeChildIndex := tr.checkReorg(lastNode)
	switch reorg {
	case nodeReorgNOP:
		// do nothing
		tr.markModifiedCommitmentsBackToRoot(proof)
	case nodeReorgRemove:
		// last node does not commit to anything, should be removed
		tr.removeKey(lastKey)
		if len(proof) >= 2 {
			tr.markModifiedCommitmentsBackToRoot(proof)
			prevKey := proof[len(proof)-2]
			prevNode := tr.mustGetNode(prevKey)
			reorg, mergeChildIndex = tr.checkReorg(prevNode)
			if reorg == nodeReorgMerge {
				tr.mergeNode(prevKey, prevNode, mergeChildIndex)
			}
		}
	case nodeReorgMerge:
		tr.mergeNode(lastKey, lastNode, mergeChildIndex)
		tr.markModifiedCommitmentsBackToRoot(proof)
	}
}

// mergeNode merges nodes when it is possible, i.e. first node does not contain Terminal commitment and has only one
// child commitment. In this case pathFragments can be merged in one resulting node
func (tr *Trie) mergeNode(key []byte, n *bufferedNode, childIndex byte) {
	nextKey := childKey(n, childIndex)
	nextNode := tr.mustGetNode(nextKey)

	tr.unDelete(key)
	ret := nextNode.Clone()
	ret.setNewKey(key)
	ret.setNewPathFragment(trie_go.Concat(n.PathFragment(), childIndex, nextNode.PathFragment()))
	tr.replaceNode(ret)
	tr.removeKey(nextKey)
}

// markModifiedCommitmentsBackToRoot updates 'modifiedChildren' marks along tha path from the updated node to the root
func (tr *Trie) markModifiedCommitmentsBackToRoot(proof [][]byte) {
	for i := len(proof) - 1; i > 0; i-- {
		k := proof[i]
		kPrev := proof[i-1]
		childIndex := k[len(k)-1]
		n := tr.mustGetNode(kPrev)
		n.markChildModified(childIndex)
	}
}

// hasCommitment returns if trie will contain commitment to the key in the (future) committed state
func (tr *Trie) hasCommitment(key []byte) bool {
	n, ok := tr.getNodeIntern(key)
	if !ok {
		return false
	}
	if n.newTerminal != nil {
		// commits to Terminal
		return true
	}
	for childIndex := range n.modifiedChildren {
		if tr.hasCommitment(childKey(n, childIndex)) {
			// modified child commits to something
			return true
		}
	}
	// new commitments do not come from children
	if len(n.n.ChildCommitments) > 0 {
		// existing children commit
		return true
	}
	// node does not commit to anything
	return false
}

type reorgStatus int

const (
	nodeReorgRemove = reorgStatus(iota)
	nodeReorgMerge
	nodeReorgNOP
)

// checkReorg check what has to be done with the node after deletion: either nothing, node must be removed or merged
func (tr *Trie) checkReorg(n *bufferedNode) (reorgStatus, byte) {
	if n.newTerminal != nil {
		return nodeReorgNOP, 0
	}
	toCheck := make(map[byte]struct{})
	for c := range n.ChildCommitments() {
		toCheck[c] = struct{}{}
	}
	for c := range n.modifiedChildren {
		if tr.hasCommitment(childKey(n, c)) {
			toCheck[c] = struct{}{}
		} else {
			delete(toCheck, c)
		}
	}
	switch len(toCheck) {
	case 0:
		return nodeReorgRemove, 0
	case 1:
		for ret := range toCheck {
			return nodeReorgMerge, ret
		}
	}
	return nodeReorgNOP, 0
}

// UpdateStr updates key/value pair in the trie
func (tr *Trie) UpdateStr(key interface{}, value interface{}) {
	var k, v []byte
	if key != nil {
		switch kt := key.(type) {
		case []byte:
			k = kt
		case string:
			k = []byte(kt)
		default:
			panic("[]byte or string expected")
		}
	}
	if value != nil {
		switch vt := value.(type) {
		case []byte:
			v = vt
		case string:
			v = []byte(vt)
		default:
			panic("[]byte or string expected")
		}
	}
	tr.Update(k, v)
}

// DeleteStr removes node from trie
func (tr *Trie) DeleteStr(key interface{}) {
	var k []byte
	if key != nil {
		switch kt := key.(type) {
		case []byte:
			k = kt
		case string:
			k = []byte(kt)
		default:
			panic("[]byte or string expected")
		}
	}
	tr.Delete(k)
}

func (tr *Trie) VectorCommitmentFromBytes(data []byte) (trie_go.VCommitment, error) {
	ret := tr.nodeStoreReader.model.NewVectorCommitment()
	if err := ret.Read(bytes.NewReader(data)); err != nil {
		return nil, err
	}
	return ret, nil
}

// Reconcile returns a list of keys in the store which cannot be proven in the trie
// Trie is consistent if empty slice is returned
// May be an expensive operation
func (tr *Trie) Reconcile(store trie_go.KVIterator) [][]byte {
	ret := make([][]byte, 0)
	store.Iterate(func(k, v []byte) bool {
		p, _, ending := proofPath(tr, []byte(k))
		if ending == EndingTerminal {
			lastKey := p[len(p)-1]
			n, ok := tr.GetNode(lastKey)
			if !ok {
				ret = append(ret, k)
			} else {
				if !trie_go.EqualCommitments(tr.nodeStoreReader.model.CommitToData(v), n.Terminal()) {
					ret = append(ret, k)
				}
			}
		} else {
			ret = append(ret, k)
		}
		return true
	})
	return ret
}

// UpdateAll mass-updates trie from the key/value store.
// To be used to build trie for arbitrary key/value data sets
func (tr *Trie) UpdateAll(store trie_go.KVIterator) {
	store.Iterate(func(k, v []byte) bool {
		tr.Update(k, v)
		return true
	})
}

func (tr *Trie) DangerouslyDumpCacheToString() string {
	ret := ""
	keys := make([]string, 0)
	for k := range tr.nodeCache {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ret += fmt.Sprintf("'%s': C = %s\n%s\n", k, tr.Model().CalcNodeCommitment(&tr.nodeCache[k].n), tr.nodeCache[k].n.String())
	}
	return ret
}
