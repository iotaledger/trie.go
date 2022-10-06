package immutable

import (
	"bytes"
	"encoding/hex"

	"github.com/iotaledger/trie.go/common"
)

// Update updates Trie with the unpackedKey/value. Reorganizes and re-calculates trie, keeps cache consistent
func (tr *Trie) Update(key []byte, value []byte) {
	unpackedTriePath := common.UnpackBytes(key, tr.PathArity())
	if len(value) == 0 {
		tr.delete(unpackedTriePath)
	} else {
		tr.update(unpackedTriePath, value)
	}
}

// Delete deletes Key/value from the Trie
func (tr *Trie) Delete(key []byte) {
	tr.Update(key, nil)
}

func (tr *TrieReader) Get(key []byte) []byte {
	//fmt.Printf("**** Get key: %s\n", string(key))
	unpackedTriePath := common.UnpackBytes(key, tr.PathArity())
	found := false
	var terminal common.TCommitment
	tr.traverseImmutablePath(unpackedTriePath, func(n *common.NodeData, _ []byte, ending ProofEndingCode) {
		//fmt.Printf("          --- traverse commitment: %s\n", n.Commitment)
		if ending == EndingTerminal {
			if !common.IsNil(n.Terminal) {
				found = true
				terminal = n.Terminal
			}
		}
	})
	if !found {
		return nil
	}
	value, valueInCommitment := common.ExtractValue(terminal)
	if valueInCommitment {
		common.Assert(len(value) > 0, "value in commitment must be not nil. Unpacked key: '%s'",
			hex.EncodeToString(unpackedTriePath))
		return value
	}
	value = tr.nodeStore.valueStore.Get(common.AsKey(terminal))
	common.Assert(len(value) > 0, "value in the value store must be not nil. Unpacked key: '%s'",
		hex.EncodeToString(unpackedTriePath))
	return value
}

func (tr *TrieReader) Has(key []byte) bool {
	unpackedTriePath := common.UnpackBytes(key, tr.PathArity())
	found := false
	tr.traverseImmutablePath(unpackedTriePath, func(n *common.NodeData, _ []byte, ending ProofEndingCode) {
		if ending == EndingTerminal {
			if !common.IsNil(n.Terminal) {
				found = true
			}
		}
	})
	return found
}

// Iterate iterates whole trie
func (tr *TrieReader) Iterate(f func(k []byte, v []byte) bool) {
	tr.iteratePrefix(f, nil)
}

// iteratePrefix iterates the key/value with keys with prefix.
// The order of the iteration will be deterministic
func (tr *TrieReader) iteratePrefix(f func(k []byte, v []byte) bool, prefix []byte) {
	var root common.VCommitment
	var triePath []byte
	unpackedPrefix := common.UnpackBytes(prefix, tr.Model().PathArity())
	tr.traverseImmutablePath(unpackedPrefix, func(n *common.NodeData, trieKey []byte, ending ProofEndingCode) {
		if bytes.HasPrefix(common.Concat(trieKey, n.PathFragment), unpackedPrefix) {
			root = n.Commitment
			triePath = trieKey
		}
	})
	if !common.IsNil(root) {
		tr.iterate(root, triePath, f)
	}
}

func (tr *TrieReader) iterate(root common.VCommitment, triePath []byte, fun func(k []byte, v []byte) bool) bool {
	n, found := tr.nodeStore.FetchNodeData(root)
	common.Assert(found, "can't fetch node. triePath: '%s', node commitment: %s", hex.EncodeToString(triePath), root)

	if !common.IsNil(n.Terminal) {
		key, err := common.PackUnpackedBytes(common.Concat(triePath, n.PathFragment), tr.Model().PathArity())
		value, inTheCommitment := n.Terminal.ExtractValue()
		if !inTheCommitment {
			value = tr.nodeStore.valueStore.Get(common.AsKey(n.Terminal))
			common.Assert(len(value) > 0, "can't fetch value. triePath: '%s', data commitment: %s", hex.EncodeToString(key), n.Terminal)
		}
		common.AssertNoError(err)
		if !fun(key, value) {
			return false
		}
	}
	for childIndex, childCommitment := range n.ChildCommitments {
		if !tr.iterate(childCommitment, common.Concat(triePath, n.PathFragment, childIndex), fun) {
			return false
		}
	}
	return true
}

// TrieIterator implements common.KVIterator interface for keys in the trie with givem prefix
type TrieIterator struct {
	prefix []byte
	tr     *TrieReader
}

func (ti *TrieIterator) Iterate(fun func(k []byte, v []byte) bool) {
	ti.tr.iteratePrefix(fun, ti.prefix)
}

// Iterator returns iterator for a sub-trie
func (tr *TrieReader) Iterator(prefix []byte) *TrieIterator {
	return &TrieIterator{
		prefix: prefix,
		tr:     tr,
	}
}

func (tr *TrieReader) GetStr(key string) string {
	return string(tr.Get([]byte(key)))
}

func (tr *TrieReader) HasStr(key string) bool {
	return tr.Has([]byte(key))
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

// DeleteStr removes key from trie
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
