package immutable

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// Trie is an updatable trie implemented on top of the unpackedKey/value store. It is virtualized and optimized by caching of the
// trie update operation and keeping consistent trie in the cache
type Trie struct {
	nodeStore      *NodeStore
	persistentRoot VCommitment
	root           *bufferedNode
}

// TrieReader direct read-only access to trie
type TrieReader struct {
	nodeStore      *NodeStore
	persistentRoot VCommitment
}

func New(nodeStore *NodeStore, root VCommitment) (*Trie, error) {
	rootNodeData, ok := nodeStore.FetchNodeData(AsKey(root), nil)
	if !ok {
		return nil, fmt.Errorf("root commitment '%s', dbKey '%s' does not exist",
			root.String(), hex.EncodeToString(AsKey(root)))
	}
	ret := &Trie{
		persistentRoot: root,
		nodeStore:      nodeStore,
		root:           newBufferedNode(rootNodeData, nil),
	}
	return ret, nil
}

func (tr *Trie) Root() VCommitment {
	return tr.root.Commitment()
}

func (tr *Trie) Model() CommitmentModel {
	return tr.nodeStore.m
}

func (tr *Trie) PathArity() PathArity {
	return tr.nodeStore.arity
}

// PersistMutations persists/append the cache to the store.
// Returns deleted part for possible use in the mutable state implementation
// Does not clear cache
func (tr *Trie) PersistMutations(store KVWriter) (int, map[string]struct{}) {
	panic("implement me")
}

// ClearCache clears the node cache
func (tr *Trie) ClearCache() {
	panic("implement me")
}

func (tr *Trie) newTerminalNode(triePath, pathFragment []byte, newTerminal TCommitment) *bufferedNode {
	ret := newBufferedNode(nil, triePath)
	ret.setPathFragment(pathFragment)
	ret.setTerminal(newTerminal, tr.Model())
	return ret
}

// Commit calculates a new root commitment value from the cache and commits all mutations in the cached TrieReader
// It is a re-calculation of the trie. bufferedNode caches are updated accordingly.
func (tr *Trie) Commit() {
	tr.commitNode(tr.root)
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

	childUpdates := make(map[byte]VCommitment)
	for idx, child := range node.uncommittedChildren {
		if child == nil {
			childUpdates[idx] = nil
		} else {
			tr.commitNode(child)
			childUpdates[idx] = child.Commitment()
		}
	}
	mutate := node.nodeModified.Clone()
	c := node.nodeFetched.Commitment.Clone()
	tr.Model().UpdateNodeCommitment(mutate, childUpdates, !IsNil(c), node.nodeModified.Terminal, &c)
	node.nodeModified.Commitment = c
	node.uncommittedChildren = make(map[byte]*bufferedNode)
}

// Update updates Trie with the unpackedKey/value. Reorganizes and re-calculates trie, keeps cache consistent
func (tr *Trie) Update(triePath []byte, value []byte) {
	var c TCommitment
	c = tr.Model().CommitToData(value)
	unpackedTriePath := UnpackBytes(triePath, tr.PathArity())
	if IsNil(c) {
		tr.root, _ = tr.delete(tr.root, unpackedTriePath)
	} else {
		tr.root = tr.update(tr.root, unpackedTriePath, c)
	}
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

// Delete deletes Key/value from the Trie, reorganizes the trie
func (tr *Trie) Delete(key []byte) {
	tr.Update(key, nil)
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

func (tr *Trie) VectorCommitmentFromBytes(data []byte) (VCommitment, error) {
	ret := tr.nodeStore.m.NewVectorCommitment()
	rdr := bytes.NewReader(data)
	if err := ret.Read(rdr); err != nil {
		return nil, err
	}
	if rdr.Len() != 0 {
		return nil, ErrNotAllBytesConsumed
	}
	return ret, nil
}

// Reconcile returns a list of keys in the store which cannot be proven in the trie
// Trie is consistent if empty slice is returned
// May be an expensive operation
func (tr *Trie) Reconcile(store KVIterator) [][]byte {
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
	//			if !tr.Model().EqualCommitments(tr.trieBuffer.nodeStore.m.CommitToData(v), n.Terminal()) {
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
func (tr *Trie) UpdateAll(store KVIterator) {
	store.Iterate(func(k, v []byte) bool {
		tr.Update(k, v)
		return true
	})
}

func (tr *Trie) DangerouslyDumpCacheToString() string {
	panic("implement me")
	//return tr.trieBuffer.dangerouslyDumpCacheToString()
}

func NewTrieReader(model CommitmentModel, trieStore, valueStore KVReader) *TrieReader {
	return &TrieReader{
		reader: NewNodeStore(trieStore, valueStore, model, model.PathArity()),
	}
}

func (tr *TrieReader) RootKey() []byte {
	return tr.rootKey
}

func (tr *TrieReader) GetNode(dbKey, triePath []byte) (Node, bool) {
	return tr.reader.getNode(dbKey, triePath)
}

func (tr *TrieReader) Model() CommitmentModel {
	return tr.reader.m
}

func (tr *TrieReader) PathArity() PathArity {
	return tr.reader.arity
}

func (tr *TrieReader) Info() string {
	return fmt.Sprintf("TrieReader ( model: %s, path arity: %s )",
		tr.reader.m.Description(), tr.reader.arity,
	)
}

func (tr *TrieReader) ValueStore() KVReader {
	return tr.reader.valueStore
}

func (tr *TrieReader) ClearCache() {
	// does nothing for the pure trieBuffer
}
