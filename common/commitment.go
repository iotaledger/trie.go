package common

import (
	"io"
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
