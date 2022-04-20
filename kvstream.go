package trie_go

import (
	"errors"
	"io"
	"math"
	"math/rand"
	"os"
	"time"
)

// Interfaces and implementations for writing/reading persistent streams of key/values

// StreamWriter represents an interface to write a sequence of key/value pairs
type StreamWriter interface {
	// Write writes key/value pair
	Write(key, value []byte) error
	// Stats return num k/v pairs and num bytes so far
	Stats() (int, int)
}

// StreamIterator is an interface to iterate stream
// In general, order is non-deterministic
type StreamIterator interface {
	Iterate(func(k, v []byte) bool) error
}

// BinaryStreamWriter writes stream of k/v pairs in binary format.
var _ StreamWriter = &BinaryStreamWriter{}

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

// BinaryStreamIterator deserializes stream of key/value pairs from io.Reader
var _ StreamIterator = &BinaryStreamIterator{}

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

// BinaryStreamFileWriter is a BinaryStreamWriter with the file as a backend
var _ StreamWriter = &BinaryStreamFileWriter{}

type BinaryStreamFileWriter struct {
	*BinaryStreamWriter
	File *os.File
}

// CreateKVStreamFile create a new BinaryStreamFileWriter
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

// BinaryStreamFileIterator is a BinaryStreamIterator with the file as a backend
var _ StreamIterator = &BinaryStreamFileIterator{}

type BinaryStreamFileIterator struct {
	*BinaryStreamIterator
	File *os.File
}

// OpenKVStreamFile opens existing file with key/value stream for reading
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

// RandStreamIterator is a stream of random key/value pairs with the given parameters
// Used for testing
var _ StreamIterator = &RandStreamIterator{}

type RandStreamIterator struct {
	rnd   *rand.Rand
	par   RandStreamParams
	count int
}

// RandStreamParams represents parameters of the RandStreamIterator
type RandStreamParams struct {
	// Seed for deterministic randomization
	Seed int64
	// NumKVPairs maximum number of key value pairs to generate. 0 means infinite
	NumKVPairs int
	// MaxKey maximum length of key (randomly generated)
	MaxKey int
	// MaxValue maximum length of value (randomly generated)
	MaxValue int
}

func NewRandStreamIterator(p ...RandStreamParams) *RandStreamIterator {
	ret := &RandStreamIterator{
		par: RandStreamParams{
			Seed:       time.Now().UnixNano(),
			NumKVPairs: 0, // infinite
			MaxKey:     64,
			MaxValue:   128,
		},
	}
	if len(p) > 0 {
		ret.par = p[0]
	}
	ret.rnd = rand.New(rand.NewSource(ret.par.Seed))
	return ret
}

func (r *RandStreamIterator) Iterate(fun func(k []byte, v []byte) bool) error {
	max := r.par.NumKVPairs
	if max <= 0 {
		max = math.MaxInt
	}
	for r.count < max {
		k := make([]byte, r.rnd.Intn(r.par.MaxKey-1)+1)
		r.rnd.Read(k)
		v := make([]byte, r.rnd.Intn(r.par.MaxValue-1)+1)
		r.rnd.Read(v)
		if !fun(k, v) {
			return nil
		}
		r.count++
	}
	return nil
}
