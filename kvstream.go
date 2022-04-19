package trie_go

import (
	"errors"
	"io"
	"os"
)

// Interfaces for writing/reading persistent streams of key/values

// StreamWriter represents an interface specific to write a sequence of key/value pairs
type StreamWriter interface {
	Write(key, value []byte) error
	Stats() (int, int) // num k/v pairs and num bytes so far
}

// StreamIterator is an interface to iterate stream
type StreamIterator interface {
	Iterate(func(k, v []byte) bool) error
}

// BinaryStreamWriter writes stream of k/v pairs in binary format.
// Keys encoding is 'bytes16' and values is 'bytes32'
type BinaryStreamWriter struct {
	w         io.Writer
	kvCount   int
	byteCount int
}

func NewBinaryStreamWriter(w io.Writer) *BinaryStreamWriter {
	return &BinaryStreamWriter{w: w}
}

// BinaryStreamWriter implements StreamWriter interface
var _ StreamWriter = &BinaryStreamWriter{}

func (b *BinaryStreamWriter) Write(key, value []byte) error {
	if err := WriteBytes16(b.w, key); err != nil {
		return err
	}
	b.byteCount += len(key) + 2
	if err := WriteBytes32(b.w, value); err != nil {
		return err
	}
	b.byteCount += len(value) + 4
	b.kvCount++
	return nil
}

func (b *BinaryStreamWriter) Stats() (int, int) {
	return b.kvCount, b.byteCount
}

type BinaryStreamIterator struct {
	r io.Reader
}

func NewBinaryStreamIterator(r io.Reader) *BinaryStreamIterator {
	return &BinaryStreamIterator{r: r}
}

func (b BinaryStreamIterator) Iterate(fun func(k []byte, v []byte) bool) error {
	for {
		k, err := ReadBytes16(b.r)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		v, err := ReadBytes32(b.r)
		if err != nil {
			return err
		}
		if !fun(k, v) {
			return nil
		}
	}
}

type BinaryStreamFileWriter struct {
	*BinaryStreamWriter
	File *os.File
}

// CreateKVStreamFile create a new k/v file
func CreateKVStreamFile(fname string) (*BinaryStreamFileWriter, error) {
	file, err := os.Create(fname)
	if err != nil {
		return nil, err
	}
	return &BinaryStreamFileWriter{
		BinaryStreamWriter: NewBinaryStreamWriter(file),
		File:               file,
	}, nil
}

type BinaryStreamFileIterator struct {
	*BinaryStreamIterator
	File *os.File
}

// OpenKVStreamFile opens existing file with k/v stream
func OpenKVStreamFile(fname string) (*BinaryStreamFileIterator, error) {
	file, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	return &BinaryStreamFileIterator{
		BinaryStreamIterator: NewBinaryStreamIterator(file),
		File:                 file,
	}, nil
}
