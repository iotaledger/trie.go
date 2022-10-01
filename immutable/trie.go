package immutable

import (
	"bytes"
	"fmt"

	"github.com/iotaledger/trie.go/common"
)

// Trie is an updatable trie implemented on top of the unpackedKey/value store. It is virtualized and optimized by caching of the
// trie update operation and keeping consistent trie in the cache
type Trie struct {
	TrieReader
	mutatedRoot *bufferedNode
}

// TrieReader direct read-only access to trie
type TrieReader struct {
	nodeStore      *immutableNodeStore
	persistentRoot common.VCommitment
}

func New(nodeStore *immutableNodeStore, root common.VCommitment) (*Trie, error) {
	rootNodeData, ok := nodeStore.FetchNodeData(root, nil)
	if !ok {
		return nil, fmt.Errorf("mutatedRoot commitment '%s', dbKey '%s' does not exist",
			root.String(), root.String())
	}
	ret := &Trie{
		TrieReader: TrieReader{
			persistentRoot: root,
			nodeStore:      nodeStore,
		},
		mutatedRoot: newBufferedNode(rootNodeData, nil),
	}
	return ret, nil
}

func (tr *TrieReader) RootCommitment() common.VCommitment {
	return tr.persistentRoot
}

func (tr *TrieReader) Model() common.CommitmentModel {
	return tr.nodeStore.m
}

func (tr *TrieReader) PathArity() common.PathArity {
	return tr.nodeStore.arity
}

// Commit calculates a new mutatedRoot commitment value from the cache and commits all mutations in the cached TrieReader
// It is a re-calculation of the trie. bufferedNode caches are updated accordingly.
func (tr *Trie) Commit() {
	tr.commitNode(tr.mutatedRoot)
}

// commitNode re-calculates node commitment and, recursively, its children commitments
// Child modification marks in 'uncommittedChildren' are updated
// Return update to the upper commitment. nil mean upper commitment is not updated
// It calls implementation-specific function UpdateNodeCommitment and passes parameter
// calcDelta = true if node's commitment can be updated incrementally. The implementation
// of UpdateNodeCommitment may use this parameter to optimize underlying cryptography
func (tr *Trie) commitNode(node *bufferedNode) {
	if node.isCommitted() {
		return
	}

	childUpdates := make(map[byte]common.VCommitment)
	for idx, child := range node.uncommittedChildren {
		if child == nil {
			childUpdates[idx] = nil
		} else {
			tr.commitNode(child)
			childUpdates[idx] = child.commitment()
		}
	}
	mutate := node.nodeModified.Clone()
	c := node.nodeFetched.Commitment.Clone()
	tr.Model().UpdateNodeCommitment(mutate, childUpdates, !common.IsNil(c), node.nodeModified.Terminal, &c)
	node.nodeModified.Commitment = c
	node.uncommittedChildren = make(map[byte]*bufferedNode)
}

// Update updates Trie with the unpackedKey/value. Reorganizes and re-calculates trie, keeps cache consistent
func (tr *Trie) Update(triePath []byte, value []byte) {
	var c common.TCommitment
	c = tr.Model().CommitToData(value)
	unpackedTriePath := common.UnpackBytes(triePath, tr.PathArity())
	if common.IsNil(c) {
		tr.mutatedRoot, _ = tr.delete(tr.mutatedRoot, unpackedTriePath)
	} else {
		tr.mutatedRoot = tr.update(tr.mutatedRoot, unpackedTriePath, c)
	}
}

// Delete deletes Key/value from the Trie, reorganizes the trie
func (tr *Trie) Delete(key []byte) {
	tr.Update(key, nil)
}

// PersistMutations persists/append the cache to the store.
// Returns deleted part for possible use in the mutable state implementation
// Does not clear cache
func (tr *Trie) PersistMutations(store common.KVWriter) (int, map[string]struct{}) {
	panic("implement me")
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

func (tr *Trie) newTerminalNode(triePath, pathFragment []byte, newTerminal common.TCommitment) *bufferedNode {
	ret := newBufferedNode(nil, triePath)
	ret.setPathFragment(pathFragment)
	ret.setTerminal(newTerminal, tr.Model())
	return ret
}

func (tr *Trie) VectorCommitmentFromBytes(data []byte) (common.VCommitment, error) {
	ret := tr.nodeStore.m.NewVectorCommitment()
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
	panic("implement me")
	//ret := make([][]byte, 0)
	//store.Iterate(func(k, v []byte) bool {
	//	p, _, ending := proofPath(tr, UnpackBytes(k, tr.PathArity()))
	//	if ending == EndingTerminal {
	//		lastKey := p[len(p)-1]
	//		n, ok := tr.GetNode(lastKey)
	//		if !ok {
	//			ret = append(ret, k)
	//		} else {
	//			if !tr.Model().EqualCommitments(tr.trieBuffer.nodeStore.m.CommitToData(v), n.terminal()) {
	//				ret = append(ret, k)
	//			}
	//		}
	//	} else {
	//		ret = append(ret, k)
	//	}
	//	return true
	//})
	//return ret
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
	panic("implement me")
	//return tr.trieBuffer.dangerouslyDumpCacheToString()
}
