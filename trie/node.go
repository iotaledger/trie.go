package trie

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// Node is a read-only interface to the 256+ trie node
type Node interface {
	// Key of the node
	Key() []byte
	// PathFragment of the node (committed)
	PathFragment() []byte
	// Terminal of the node (committed)
	Terminal() TCommitment
	// ChildCommitments can return old commitments if node is not committed
	ChildCommitments() map[byte]VCommitment
}

// Implementations of read-only and buffered/updatable nodes of the 256+ trie

// nodeReadOnly is non-buffered node data
type nodeReadOnly struct {
	// persistent
	n NodeData
	// persisted in the key of the map
	key []byte
}

func (n *nodeReadOnly) PathFragment() []byte {
	return n.n.PathFragment
}

func (n *nodeReadOnly) Terminal() TCommitment {
	return n.n.Terminal
}

func (n *nodeReadOnly) Key() []byte {
	return n.key
}

func (n *nodeReadOnly) ChildCommitments() map[byte]VCommitment {
	Assert(n.IsCommitted(), "trie::nodeReadOnly::ChildCommitments: node is not committed: key: '%s'",
		hex.EncodeToString(n.key))
	return n.n.ChildCommitments
}

func (n *nodeReadOnly) IsCommitted() bool {
	return true
}

func nodeReadOnlyFromBytes(model CommitmentModel, data, unpackedKey []byte, arity PathArity, valueStore KVReader) (*nodeReadOnly, error) {
	ret, err := NodeDataFromBytes(model, data, unpackedKey, arity, valueStore)
	if err != nil {
		return nil, err
	}
	return &nodeReadOnly{
		n:   *ret,
		key: unpackedKey,
	}, nil
}

// bufferedNode is a node of the 256+-ary Trie with cache
type bufferedNode struct {
	// persistent
	n NodeData
	// persisted in the unpackedKey of the map
	unpackedKey []byte
	// non-persistent
	newTerminal      TCommitment       // next value of Terminal
	modifiedChildren map[byte]struct{} // children which has been modified
	pathChanged      bool              // position of the node in trie has been changed duo to modifications
}

func newBufferedNode(key []byte) *bufferedNode {
	return &bufferedNode{
		n:                *NewNodeData(),
		unpackedKey:      key,
		newTerminal:      nil,
		modifiedChildren: make(map[byte]struct{}),
	}
}

func (n *bufferedNode) PathFragment() []byte {
	return n.n.PathFragment
}

func (n *bufferedNode) Terminal() TCommitment {
	return n.newTerminal
}

func (n *bufferedNode) Key() []byte {
	return n.unpackedKey
}

func (n *bufferedNode) ChildCommitments() map[byte]VCommitment {
	return n.n.ChildCommitments
}

func (n *bufferedNode) Clone() *bufferedNode {
	if n == nil {
		return nil
	}
	var newTerminal TCommitment
	if n.newTerminal == nil {
		newTerminal = nil
	} else {
		newTerminal = n.newTerminal.Clone()
	}
	ret := &bufferedNode{
		n:                *n.n.Clone(),
		unpackedKey:      make([]byte, len(n.unpackedKey)),
		newTerminal:      newTerminal,
		modifiedChildren: make(map[byte]struct{}),
		pathChanged:      n.pathChanged,
	}
	copy(ret.unpackedKey, n.unpackedKey)
	for k, v := range n.modifiedChildren {
		ret.modifiedChildren[k] = v
	}
	return ret
}

func (n *bufferedNode) setNewKey(key []byte) {
	n.unpackedKey = key
	n.pathChanged = true
}

func (n *bufferedNode) setNewPathFragment(pf []byte) {
	n.n.PathFragment = pf
	n.pathChanged = true
}

func (n *bufferedNode) setNewTerminal(t TCommitment) {
	n.newTerminal = t
}

func (n *bufferedNode) markChildModified(index byte) {
	n.modifiedChildren[index] = struct{}{}
}

func (n *bufferedNode) Bytes(model CommitmentModel, arity PathArity, optimizeKeyCommitments bool) []byte {
	// Optimization: if terminal commits to unpackedKey, no need to serialize it,
	// because all information is in the key
	isKeyCommitment := false
	if optimizeKeyCommitments && len(n.unpackedKey) > 0 {
		keyCommitment := model.CommitToData(Concat(n.unpackedKey, n.n.PathFragment))
		isKeyCommitment = model.EqualCommitments(n.n.Terminal, keyCommitment)
	}
	var buf bytes.Buffer
	skipStoreTerminal := n.n.Terminal != nil && !model.ForceStoreTerminalWithNode(n.n.Terminal)
	err := n.n.Write(&buf, arity, isKeyCommitment, skipStoreTerminal)
	Assert(err == nil, "trie::bufferedNode::Bytes: %v", err)
	return buf.Bytes()
}

func childKey(n Node, childIndex byte) []byte {
	return Concat(n.Key(), n.PathFragment(), childIndex)
}

func ToString(n Node) string {
	return fmt.Sprintf("nodeData(key: '%s', pathFragment: '%s', term: '%s', numChildren: %d",
		hex.EncodeToString(n.Key()),
		hex.EncodeToString(n.PathFragment()),
		n.Terminal().String(),
		len(n.ChildCommitments()),
	)
}
