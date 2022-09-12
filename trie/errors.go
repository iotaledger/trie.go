package trie

import "golang.org/x/xerrors"

var (
	ErrNotAllBytesConsumed = xerrors.New("serialization error: not all bytes were consumed")
)
