package mutable

import (
	"bytes"
	"fmt"

	"github.com/iotaledger/trie.go/common"
)

// Trie is an updatable trie implemented on top of the unpackedKey/value store. It is virtualized and optimized by caching of the
// trie update operation and keeping consistent trie in the cache
type Trie struct {
	nodeStore *nodeStoreBuffered
}

// TrieReader direct read-only access to trie
type TrieReader struct {
	reader *nodeStore
}

// NodeStore is an interface to TrieReader to the trie as a set of TrieReader represented as unpackedKey/value pairs
// Two implementations:
// - TrieReader is a direct, non-cached TrieReader to unpackedKey/value storage
// - Trie implement a cached TrieReader
type NodeStore interface {
	GetNode(unpackedKey []byte) (Node, bool)
	Model() common.CommitmentModel
	PathArity() common.PathArity
	Info() string
}

// RootCommitment computes root commitment from the root node of the trie represented as a NodeStore
func RootCommitment(tr NodeStore) common.VCommitment {
	n, ok := tr.GetNode(nil)
	if !ok {
		return nil
	}
	return tr.Model().CalcNodeCommitment(&common.NodeData{
		PathFragment:     n.PathFragment(),
		ChildCommitments: n.ChildCommitments(),
		Terminal:         n.Terminal(),
	})
}

// Trie implements NodeStore interface. It buffers (caches) all TrieReader for optimization purposes
var _ NodeStore = &Trie{}

func NewTrie(model common.CommitmentModel, trieStore, valueStore common.KVReader) *Trie {
	ret := &Trie{
		nodeStore: newNodeStoreBuffered(model, trieStore, valueStore, model.PathArity()),
	}
	return ret
}

// Clone is a deep copy of the trie, including its buffered data
func (tr *Trie) Clone() *Trie {
	return &Trie{
		nodeStore: tr.nodeStore.clone(),
	}
}

func (tr *Trie) Model() common.CommitmentModel {
	return tr.nodeStore.reader.m
}

func (tr *Trie) PathArity() common.PathArity {
	return tr.nodeStore.arity
}

// GetNode fetches node from the trie
func (tr *Trie) GetNode(unpackedKey []byte) (Node, bool) {
	return tr.nodeStore.getNode(unpackedKey)
}

func (tr *Trie) Info() string {
	return fmt.Sprintf("Trie( common dscr: '%s')", tr.nodeStore.reader.m.Description())
}

// PersistMutations persists the cache to the unpackedKey/value store
// Does not clear cache
func (tr *Trie) PersistMutations(store common.KVWriter) int {
	return tr.nodeStore.persistMutations(store)
}

// ClearCache clears the node cache
func (tr *Trie) ClearCache() {
	tr.nodeStore.clearCache()
}

// newTerminalNode creates new node in the trie with specified pathFragment and terminal commitment.
// Assumes 'unpackedKey' does not exist in the Trie
func (tr *Trie) newTerminalNode(unpackedKey, unpackedPathFragment []byte, newTerminal common.TCommitment) *bufferedNode {
	tr.nodeStore.unDelete(unpackedKey)
	ret := newBufferedNode(unpackedKey)
	ret.newTerminal = newTerminal
	ret.n.PathFragment = unpackedPathFragment
	ret.pathChanged = true
	tr.nodeStore.insertNewNode(ret)
	return ret
}

// Commit calculates a new root commitment value from the cache and commits all mutations in the cached TrieReader
// It is a re-calculation of the trie. bufferedNode caches are updated accordingly.
func (tr *Trie) Commit() {
	tr.commitNode(nil)
}

// commitNode re-calculates node commitment and, recursively, its children commitments
// Child modification marks in 'modifiedChildren' are updated
// Return update to the upper commitment. nil mean upper commitment is not updated
// It calls implementation-specific function UpdateNodeCommitment and passes parameter
// calcDelta = true if node's commitment can be updated incrementally. The implementation
// of UpdateNodeCommitment may use this parameter to optimize underlying cryptography
func (tr *Trie) commitNode(key []byte) *bufferedNode {
	n, ok := tr.nodeStore.getNode(key)
	if !ok {
		return nil
	}
	isModified := n.pathChanged || len(n.modifiedChildren) > 0 || !tr.Model().EqualCommitments(n.newTerminal, n.n.Terminal)
	if !isModified {
		return n
	}
	mutate := n.n.Clone()
	childUpdates := make(map[byte]common.VCommitment)
	for childIndex := range n.modifiedChildren {
		//curCommitment := mutate.ChildCommitments[childIndex] // may be nil
		child := tr.commitNode(childKey(n, childIndex))
		if child != nil {
			childUpdates[childIndex] = child.n.Commitment
		} else {
			childUpdates[childIndex] = nil
		}
	}

	calcDelta := !n.pathChanged
	tr.Model().UpdateNodeCommitment(mutate, childUpdates, n.newTerminal, n.PathFragment(), calcDelta)

	n.n = *mutate
	if len(n.modifiedChildren) > 0 {
		// clean the modification marks if any
		n.modifiedChildren = make(map[byte]struct{})
	}
	n.pathChanged = false
	return n
}

// Update updates Trie with the unpackedKey/value. Reorganizes and re-calculates trie, keeps cache consistent
func (tr *Trie) Update(key []byte, value []byte) {
	term := tr.nodeStore.reader.m.CommitToData(value)
	if common.IsNil(term) {
		// nil value means deletion
		tr.Delete(key)
		return
	}
	// find path in the trie corresponding to the unpackedKey
	unpackedKey := common.UnpackBytes(key, tr.nodeStore.arity)
	proof, lastCommonPrefix, ending := proofPath(tr, unpackedKey)
	if len(proof) == 0 {
		tr.newTerminalNode(nil, unpackedKey, term)
		return
	}
	lastKey := proof[len(proof)-1]
	switch ending {
	case common.EndingTerminal:
		tr.nodeStore.mustGetNode(lastKey).setNewTerminal(term)

	case common.EndingExtend:
		childIndexPosition := len(lastKey) + len(lastCommonPrefix)
		common.Assert(childIndexPosition < len(unpackedKey), "childPosition < len(unpackedKey)")
		childIndex := unpackedKey[childIndexPosition]
		tr.nodeStore.removeKey(unpackedKey[:childIndexPosition+1])
		tr.newTerminalNode(unpackedKey[:childIndexPosition+1], unpackedKey[childIndexPosition+1:], term)
		tr.nodeStore.mustGetNode(lastKey).markChildModified(childIndex)

	case common.EndingSplit:
		// splitting the node into two path fragments
		tr.splitNode(unpackedKey, lastKey, lastCommonPrefix, term)

	default:
		panic("inconsistency: unknown path ending code")
	}
	tr.markModifiedCommitmentsBackToRoot(proof)
}

// InsertKeyCommitment inserts unpackedKey/value pair with equal unpackedKey and value.
// Key must not be empty.
// It leads to optimized serialization of trie nodes because terminal commitment is
// contained in the unpackedKey.
// It saves 33 bytes per trie node for use cases such as ledger state commitment via UTXO IDs:
// each UTXO ID is a commitment to the output, so we only need PoI, not the commitment itself
func (tr *Trie) InsertKeyCommitment(key []byte) {
	if len(key) == 0 {
		panic("InsertKeyCommitment: unpackedKey can't be empty")
	}
	tr.Update(key, key)
}

func (tr *Trie) splitNode(fullKey, lastKey, commonPrefix []byte, newTerminal common.TCommitment) {
	splitIndex := len(commonPrefix)
	childPosition := len(lastKey) + splitIndex
	common.Assert(childPosition <= len(fullKey), "childPosition <= len(fullKey)")

	n := tr.nodeStore.mustGetNode(lastKey)

	keyNewNode := make([]byte, childPosition+1)
	copy(keyNewNode, fullKey)
	common.Assert(splitIndex < len(n.n.PathFragment), "splitIndex < len(n.newPathFragment)")
	childContinue := n.n.PathFragment[splitIndex]
	keyNewNode[len(keyNewNode)-1] = childContinue

	// create new node with keyNewNode, move everything from old to the new node
	// Only path fragment and unpackedKey changes
	newNode := n.Clone() // children and terminal remains the same, pathFragment changes
	newNode.setNewKey(keyNewNode)
	newNode.setNewPathFragment(n.PathFragment()[splitIndex+1:])
	tr.nodeStore.insertNewNode(newNode)

	// modify the node under the old unpackedKey
	n.setNewPathFragment(commonPrefix)
	n.n.ChildCommitments = make(map[uint8]common.VCommitment)
	n.modifiedChildren = make(map[uint8]struct{})
	n.markChildModified(childContinue)
	n.n.Terminal = nil
	n.newTerminal = nil

	// insert terminal
	if childPosition == len(fullKey) {
		// no need for the new node
		n.newTerminal = newTerminal
	} else {
		// create a new node
		keyFork := fullKey[:len(keyNewNode)]
		childForkIndex := keyFork[len(keyFork)-1]
		common.Assert(childForkIndex != childContinue, "childForkIndex != childContinue")
		tr.newTerminalNode(keyFork, fullKey[len(keyFork):], newTerminal)
		n.markChildModified(childForkIndex)
	}
}

// Delete deletes Key/value from the Trie, reorganizes the trie
func (tr *Trie) Delete(key []byte) {
	unpackedKey := common.UnpackBytes(key, tr.nodeStore.arity)
	proof, _, ending := proofPath(tr, unpackedKey)
	if len(proof) == 0 || ending != common.EndingTerminal {
		return
	}
	lastKey := proof[len(proof)-1]
	lastNode, ok := tr.nodeStore.getNode(lastKey)
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
		tr.nodeStore.removeKey(lastKey)
		if len(proof) >= 2 {
			tr.markModifiedCommitmentsBackToRoot(proof)
			prevKey := proof[len(proof)-2]
			prevNode := tr.nodeStore.mustGetNode(prevKey)
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

// mergeNode merges nodes when it is possible, i.e. first node does not contain terminal commitment and has only one
// child commitment. In this case pathFragments can be merged in one resulting node
func (tr *Trie) mergeNode(key []byte, n *bufferedNode, childIndex byte) {
	nextKey := childKey(n, childIndex)
	nextNode := tr.nodeStore.mustGetNode(nextKey)

	tr.nodeStore.unDelete(key)
	ret := nextNode.Clone()
	ret.setNewKey(key)
	ret.setNewPathFragment(common.Concat(n.PathFragment(), childIndex, nextNode.PathFragment()))
	tr.nodeStore.replaceNode(ret)
	tr.nodeStore.removeKey(nextKey)
}

// markModifiedCommitmentsBackToRoot updates 'modifiedChildren' marks along tha path from the updated node to the root
func (tr *Trie) markModifiedCommitmentsBackToRoot(proof [][]byte) {
	for i := len(proof) - 1; i > 0; i-- {
		k := proof[i]
		kPrev := proof[i-1]
		childIndex := k[len(k)-1]
		n := tr.nodeStore.mustGetNode(kPrev)
		n.markChildModified(childIndex)
	}
}

// hasCommitment returns if trie will contain commitment to the unpackedKey in the (future) committed state
func (tr *Trie) hasCommitment(key []byte) bool {
	n, ok := tr.nodeStore.getNode(key)
	if !ok {
		return false
	}
	if n.newTerminal != nil {
		// commits to terminal
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

// UpdateStr updates unpackedKey/value pair in the trie
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

func (tr *Trie) VectorCommitmentFromBytes(data []byte) (common.VCommitment, error) {
	ret := tr.nodeStore.reader.m.NewVectorCommitment()
	rdr := bytes.NewReader(data)
	if err := ret.Read(rdr); err != nil {
		return nil, err
	}
	if rdr.Len() != 0 {
		return nil, common.ErrNotAllBytesConsumed
	}
	return ret, nil
}

// Reconcile returns a list of keys in the store which cannot be proven in the trie
// Trie is consistent if empty slice is returned
// May be an expensive operation
func (tr *Trie) Reconcile(store common.KVIterator) [][]byte {
	ret := make([][]byte, 0)
	store.Iterate(func(k, v []byte) bool {
		p, _, ending := proofPath(tr, common.UnpackBytes(k, tr.PathArity()))
		if ending == common.EndingTerminal {
			lastKey := p[len(p)-1]
			n, ok := tr.GetNode(lastKey)
			if !ok {
				ret = append(ret, k)
			} else {
				if !tr.Model().EqualCommitments(tr.nodeStore.reader.m.CommitToData(v), n.Terminal()) {
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

// UpdateAll mass-updates trie from the unpackedKey/value store.
// To be used to build trie for arbitrary unpackedKey/value data sets
func (tr *Trie) UpdateAll(store common.KVIterator) {
	store.Iterate(func(k, v []byte) bool {
		tr.Update(k, v)
		return true
	})
}

func (tr *Trie) DangerouslyDumpCacheToString() string {
	return tr.nodeStore.dangerouslyDumpCacheToString()
}

// TrieReader implements NodeStore
var _ NodeStore = &TrieReader{}

func NewTrieReader(model common.CommitmentModel, trieStore, valueStore common.KVReader) *TrieReader {
	return &TrieReader{
		reader: newNodeStore(trieStore, valueStore, model, model.PathArity()),
	}
}

func (tr *TrieReader) GetNode(unpackedKey []byte) (Node, bool) {
	return tr.reader.getNode(unpackedKey)
}

func (tr *TrieReader) Model() common.CommitmentModel {
	return tr.reader.m
}

func (tr *TrieReader) PathArity() common.PathArity {
	return tr.reader.arity
}

func (tr *TrieReader) Info() string {
	return fmt.Sprintf("TrieReader ( common: %s, path arity: %s )",
		tr.reader.m.Description(), tr.reader.arity,
	)
}
