package common

import (
	"bytes"
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
	AsKey() []byte
	Serializable
}

// TCommitment represents commitment to the terminal data. Usually it is a hash of the data of a scalar field element
type TCommitment interface {
	Clone() TCommitment
	Serializable
}

func ReadVectorCommitment(m CommitmentModel, r io.Reader) (VCommitment, error) {
	ret := m.NewVectorCommitment()
	if err := ret.Read(r); err != nil {
		return nil, err
	}
	return ret, nil
}

func ReadTerminalCommitment(m CommitmentModel, r io.Reader) (TCommitment, error) {
	ret := m.NewTerminalCommitment()
	if err := ret.Read(r); err != nil {
		return nil, err
	}
	return ret, nil
}

func VectorCommitmentFromBytes(m CommitmentModel, data []byte) (VCommitment, error) {
	rdr := bytes.NewReader(data)
	ret, err := ReadVectorCommitment(m, rdr)
	if err != nil {
		return nil, err
	}
	if rdr.Len() > 0 {
		return nil, ErrNotAllBytesConsumed
	}
	return ret, nil
}

func TerminalCommitmentFromBytes(m CommitmentModel, data []byte) (TCommitment, error) {
	rdr := bytes.NewReader(data)
	ret, err := ReadTerminalCommitment(m, rdr)
	if err != nil {
		return nil, err
	}
	if rdr.Len() > 0 {
		return nil, ErrNotAllBytesConsumed
	}
	return ret, nil
}
