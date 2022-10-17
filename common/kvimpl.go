package common

import (
	"bytes"
	"errors"
	"io"
	"math"
	"math/rand"
	"os"
	"time"
)

//----------------------------------------------------------------------------
// InMemoryKVStore is a KVStore implementation. Mostly used for testing
var (
	_ KVStore          = InMemoryKVStore{}
	_ BatchedUpdatable = InMemoryKVStore{}
	_ Traversable      = InMemoryKVStore{}
	_ KVBatchedWriter  = &simpleBatchedMemoryWriter{}
	_ KVIterator       = &simpleInMemoryIterator{}
)

type (
	InMemoryKVStore map[string][]byte

	mutation struct {
		key   []byte
		value []byte
	}

	simpleBatchedMemoryWriter struct {
		store     InMemoryKVStore
		mutations []mutation
	}

	simpleInMemoryIterator struct {
		store  InMemoryKVStore
		prefix []byte
	}
)

func NewInMemoryKVStore() InMemoryKVStore {
	return make(InMemoryKVStore)
}

func (im InMemoryKVStore) Get(k []byte) []byte {
	return im[string(k)]
}

func (im InMemoryKVStore) Has(k []byte) bool {
	_, ok := im[string(k)]
	return ok
}

func (im InMemoryKVStore) Iterate(f func(k []byte, v []byte) bool) {
	for k, v := range im {
		if !f([]byte(k), v) {
			return
		}
	}
}

func (im InMemoryKVStore) Set(k, v []byte) {
	if len(v) != 0 {
		im[string(k)] = v
	} else {
		delete(im, string(k))
	}
}

func (bw *simpleBatchedMemoryWriter) Set(key, value []byte) {
	bw.mutations = append(bw.mutations, mutation{
		key:   key,
		value: value,
	})
}

func (bw *simpleBatchedMemoryWriter) Commit() error {
	for _, m := range bw.mutations {
		if len(m.value) > 0 {
			bw.store[string(m.key)] = m.value
		} else {
			delete(bw.store, string(m.key))
		}
	}
	bw.mutations = make([]mutation, 0)
	return nil
}

func (im InMemoryKVStore) BatchedWriter() KVBatchedWriter {
	return &simpleBatchedMemoryWriter{
		store:     im,
		mutations: make([]mutation, 0),
	}
}

func (im InMemoryKVStore) Iterator(prefix []byte) KVIterator {
	return &simpleInMemoryIterator{
		store:  im,
		prefix: prefix,
	}
}

func (si *simpleInMemoryIterator) Iterate(f func(k []byte, v []byte) bool) {
	var key []byte
	for k, v := range si.store {
		key = []byte(k)
		if bytes.HasPrefix(key, si.prefix) {
			if !f(key, v) {
				return
			}
		}
	}
}

//----------------------------------------------------------------------------
// interfaces for writing/reading persistent streams of key/value pairs

// KVStreamWriter represents an interface to write a sequence of key/value pairs
type KVStreamWriter interface {
	// Write writes key/value pair
	Write(key, value []byte) error
	// Stats return num k/v pairs and num bytes so far
	Stats() (int, int)
}

// KVStreamIterator is an interface to iterate stream
// In general, order is non-deterministic
type KVStreamIterator interface {
	Iterate(func(k, v []byte) bool) error
}

//----------------------------------------------------------------------------
// implementations of writing/reading persistent streams of key/value pairs

// BinaryStreamWriter writes stream of k/v pairs in binary format
// Each key is prefixed with 2 bytes (little-endian uint16) of size,
// each value with 4 bytes of size (little-endian uint32)
var _ KVStreamWriter = &BinaryStreamWriter{}

type BinaryStreamWriter struct {
	w         io.Writer
	kvCount   int
	byteCount int
}

func NewBinaryStreamWriter(w io.Writer) *BinaryStreamWriter {
	return &BinaryStreamWriter{w: w}
}

// BinaryStreamWriter implements KVStreamWriter interface
var _ KVStreamWriter = &BinaryStreamWriter{}

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
var _ KVStreamIterator = &BinaryStreamIterator{}

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
var _ KVStreamWriter = &BinaryStreamFileWriter{}

type BinaryStreamFileWriter struct {
	*BinaryStreamWriter
	file *os.File
}

// CreateKVStreamFile create a new BinaryStreamFileWriter
func CreateKVStreamFile(fname string) (*BinaryStreamFileWriter, error) {
	file, err := os.Create(fname)
	if err != nil {
		return nil, err
	}
	return &BinaryStreamFileWriter{
		BinaryStreamWriter: NewBinaryStreamWriter(file),
		file:               file,
	}, nil
}

func (fw *BinaryStreamFileWriter) Close() error {
	return fw.file.Close()
}

// BinaryStreamFileIterator is a BinaryStreamIterator with the file as a backend
var _ KVStreamIterator = &BinaryStreamFileIterator{}

type BinaryStreamFileIterator struct {
	*BinaryStreamIterator
	file *os.File
}

// OpenKVStreamFile opens existing file with key/value stream for reading
func OpenKVStreamFile(fname string) (*BinaryStreamFileIterator, error) {
	file, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	return &BinaryStreamFileIterator{
		BinaryStreamIterator: NewBinaryStreamIterator(file),
		file:                 file,
	}, nil
}

func (fs *BinaryStreamFileIterator) Close() error {
	return fs.file.Close()
}

// RandStreamIterator is a stream of random key/value pairs with the given parameters
// Used for testing
var _ KVStreamIterator = &RandStreamIterator{}

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
