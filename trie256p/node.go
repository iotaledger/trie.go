package trie256p

import (
	"bytes"
	trie_go "github.com/iotaledger/trie.go"
)

// Node is a read-only interface to the 256+ trie node
type Node interface {
	// Key of the node
	Key() []byte
	// PathFragment of the node (committed)
	PathFragment() []byte
	// Terminal of the node (committed)
	Terminal() trie_go.TCommitment
	// ChildCommitments can return old commitments if node is not committed
	ChildCommitments() map[byte]trie_go.VCommitment
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

func (n *nodeReadOnly) Terminal() trie_go.TCommitment {
	return n.n.Terminal
}

func (n *nodeReadOnly) Key() []byte {
	return n.key
}

func (n *nodeReadOnly) ChildCommitments() map[byte]trie_go.VCommitment {
	trie_go.Assert(n.IsCommitted(), "ChildCommitments: node is not committed")
	return n.n.ChildCommitments
}

func (n *nodeReadOnly) IsCommitted() bool {
	return true
}

func nodeReadOnlyFromBytes(model CommitmentModel, data, unpackedKey []byte, arity PathArity) (*nodeReadOnly, error) {
	ret, err := NodeDataFromBytes(model, data, unpackedKey, arity)
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
	// persisted in the key of the map
	key []byte
	// non-persistent
	newTerminal      trie_go.TCommitment // next value of Terminal
	modifiedChildren map[byte]struct{}   // children which has been modified
	pathChanged      bool                // position of the node in trie has been changed duo to modifications
}

func newBufferedNode(key []byte) *bufferedNode {
	return &bufferedNode{
		n:                *NewNodeData(),
		key:              key,
		newTerminal:      nil,
		modifiedChildren: make(map[byte]struct{}),
	}
}

func (n *bufferedNode) PathFragment() []byte {
	return n.n.PathFragment
}

func (n *bufferedNode) Terminal() trie_go.TCommitment {
	return n.newTerminal
}

func (n *bufferedNode) Key() []byte {
	return n.key
}

func (n *bufferedNode) ChildCommitments() map[byte]trie_go.VCommitment {
	return n.n.ChildCommitments
}

func (n *bufferedNode) Clone() *bufferedNode {
	if n == nil {
		return nil
	}
	var newTerminal trie_go.TCommitment
	if n.newTerminal == nil {
		newTerminal = nil
	} else {
		newTerminal = n.newTerminal.Clone()
	}
	ret := &bufferedNode{
		n:                *n.n.Clone(),
		key:              make([]byte, len(n.key)),
		newTerminal:      newTerminal,
		modifiedChildren: make(map[byte]struct{}),
		pathChanged:      n.pathChanged,
	}
	copy(ret.key, n.key)
	for k, v := range n.modifiedChildren {
		ret.modifiedChildren[k] = v
	}
	return ret
}

func (n *bufferedNode) setNewKey(key []byte) {
	n.key = key
	n.pathChanged = true
}

func (n *bufferedNode) setNewPathFragment(pf []byte) {
	n.n.PathFragment = pf
	n.pathChanged = true
}

func (n *bufferedNode) setNewTerminal(t trie_go.TCommitment) {
	n.newTerminal = t
}

func (n *bufferedNode) markChildModified(index byte) {
	n.modifiedChildren[index] = struct{}{}
}

func (n *bufferedNode) isModified() bool {
	return n.pathChanged || len(n.modifiedChildren) > 0 || !trie_go.EqualCommitments(n.newTerminal, n.n.Terminal)
}

func (n *bufferedNode) Bytes(model CommitmentModel, arity PathArity, optimizeKeyCommitments bool) []byte {
	// Optimization: if terminal commits to key, no need to serialize it
	isKeyCommitment := false
	if optimizeKeyCommitments && len(n.key) > 0 {
		keyCommitment := model.CommitToData(trie_go.Concat(n.key, n.n.PathFragment))
		isKeyCommitment = trie_go.EqualCommitments(n.n.Terminal, keyCommitment)
	}
	var buf bytes.Buffer
	err := n.n.Write(&buf, arity, isKeyCommitment)
	trie_go.Assert(err == nil, "%v", err)
	return buf.Bytes()
}

func childKey(n Node, childIndex byte) []byte {
	return trie_go.Concat(n.Key(), n.PathFragment(), childIndex)
}
