package trie

import (
	"bytes"
	"io"
	"reflect"
)

// abstraction of commitment data

// Serializable is a common interface for serialization of commitment data
type Serializable interface {
	Read(r io.Reader) error
	Write(w io.Writer) error
	Bytes() []byte
	String() string
}

// VCommitment represents interface to the vector commitment. It can be hash, or it can be a curve element
type VCommitment interface {
	Clone() VCommitment
	Serializable
}

// TCommitment represents commitment to the terminal data. Usually it is a hash of the data of a scalar field element
type TCommitment interface {
	Clone() TCommitment
	Serializable
}

// EqualCommitments a generic way to compare 2 commitments
func EqualCommitments(c1, c2 Serializable) bool {
	// TODO better suggestion? The problem: type(nil) != nil
	c1Nil := c1 == nil || (reflect.ValueOf(c1).Kind() == reflect.Ptr && reflect.ValueOf(c1).IsNil())
	c2Nil := c2 == nil || (reflect.ValueOf(c2).Kind() == reflect.Ptr && reflect.ValueOf(c2).IsNil())
	if c1Nil && c2Nil {
		return true
	}
	if c1Nil || c2Nil {
		return false
	}
	return bytes.Equal(c1.Bytes(), c2.Bytes())
}
