package trie_go

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
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

// VCommitment represents interface to vector commitment. It can be hash, or it can be a curve element
type VCommitment interface {
	Clone() VCommitment
	Serializable
}

// TCommitment represents commitment to the terminal data. Usually it is a hash of field element
type TCommitment interface {
	Clone() TCommitment
	Serializable
}

// MustBytes most common way of serialization
func MustBytes(o interface{ Write(w io.Writer) error }) []byte {
	var buf bytes.Buffer
	if err := o.Write(&buf); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func EqualCommitments(c1, c2 Serializable) bool {
	if c1 == c2 {
		return true
	}
	// TODO better suggestion ? The problem: type(nil) != nil
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

// abstraction of key/value interfaces

type KVReader interface {
	Get(key []byte) []byte // nil mean absence
	Has(key []byte) bool   // for performance
}

type KVWriter interface {
	Set(key, value []byte) // value == nil means deletion
}

type KVIterator interface {
	Iterate(func(k, v []byte) bool)
}

type KVStore interface {
	KVReader
	KVWriter
	KVIterator
}

// in memory KVStore implementations. Mostly used for testing

type inMemoryKVStore map[string][]byte

func NewInMemoryKVStore() KVStore {
	return make(inMemoryKVStore)
}

func (i inMemoryKVStore) Get(k []byte) []byte {
	return i[string(k)]
}

func (i inMemoryKVStore) Has(k []byte) bool {
	_, ok := i[string(k)]
	return ok
}

func (i inMemoryKVStore) Iterate(f func(k []byte, v []byte) bool) {
	for k, v := range i {
		if !f([]byte(k), v) {
			return
		}
	}
}

func (i inMemoryKVStore) Set(k, v []byte) {
	if len(v) != 0 {
		i[string(k)] = v
	} else {
		delete(i, string(k))
	}
}

// r/w utility functions

func ReadBytes16(r io.Reader) ([]byte, error) {
	var length uint16
	err := ReadUint16(r, &length)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return []byte{}, nil
	}
	ret := make([]byte, length)
	_, err = r.Read(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func WriteBytes16(w io.Writer, data []byte) error {
	if len(data) > math.MaxUint16 {
		panic(fmt.Sprintf("WriteBytes16: too long data (%v)", len(data)))
	}
	err := WriteUint16(w, uint16(len(data)))
	if err != nil {
		return err
	}
	if len(data) != 0 {
		_, err = w.Write(data)
	}
	return err
}

func ReadUint16(r io.Reader, pval *uint16) error {
	var tmp2 [2]byte
	_, err := r.Read(tmp2[:])
	if err != nil {
		return err
	}
	*pval = binary.LittleEndian.Uint16(tmp2[:])
	return nil
}

func WriteUint16(w io.Writer, val uint16) error {
	_, err := w.Write(Uint16To2Bytes(val))
	return err
}

func Uint16To2Bytes(val uint16) []byte {
	var tmp2 [2]byte
	binary.LittleEndian.PutUint16(tmp2[:], val)
	return tmp2[:]
}

func Uint16From2Bytes(b []byte) (uint16, error) {
	if len(b) != 2 {
		return 0, errors.New("len(b) != 2")
	}
	return binary.LittleEndian.Uint16(b), nil
}

func ReadByte(r io.Reader) (byte, error) {
	var b [1]byte
	_, err := r.Read(b[:])
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func WriteByte(w io.Writer, val byte) error {
	b := []byte{val}
	_, err := w.Write(b)
	return err
}

func ReadBytes32(r io.Reader) ([]byte, error) {
	var length uint32
	err := ReadUint32(r, &length)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return []byte{}, nil
	}
	ret := make([]byte, length)
	_, err = r.Read(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func WriteBytes32(w io.Writer, data []byte) error {
	if len(data) > math.MaxUint32 {
		panic(fmt.Sprintf("WriteBytes32: too long data (%v)", len(data)))
	}
	err := WriteUint32(w, uint32(len(data)))
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func Uint32To4Bytes(val uint32) []byte {
	var tmp4 [4]byte
	binary.LittleEndian.PutUint32(tmp4[:], val)
	return tmp4[:]
}

func Uint32From4Bytes(b []byte) (uint32, error) {
	if len(b) != 4 {
		return 0, errors.New("len(b) != 4")
	}
	return binary.LittleEndian.Uint32(b), nil
}

func MustUint32From4Bytes(b []byte) uint32 {
	ret, err := Uint32From4Bytes(b)
	if err != nil {
		panic(err)
	}
	return ret
}

func ReadUint32(r io.Reader, pval *uint32) error {
	var tmp4 [4]byte
	_, err := r.Read(tmp4[:])
	if err != nil {
		return err
	}
	*pval = MustUint32From4Bytes(tmp4[:])
	return nil
}

func WriteUint32(w io.Writer, val uint32) error {
	_, err := w.Write(Uint32To4Bytes(val))
	return err
}

type byteCounter int

func (b *byteCounter) Write(p []byte) (n int, err error) {
	*b = byteCounter(int(*b) + len(p))
	return 0, nil
}
