package common

//----------------------------------------------------------------------------
// generic abstraction interfaces of key/value storage

// KVReader is a key/value reader
type KVReader interface {
	// Get retrieves value by key. Returned nil means absence of the key
	Get(key []byte) []byte
	// Has checks presence of the key in the key/value store
	Has(key []byte) bool // for performance
}

// KVWriter is a key/value writer
type KVWriter interface {
	// Set writes new or updates existing key with the value.
	// value == nil means deletion of the key from the store
	Set(key, value []byte)
}

// KVIterator is an interface to iterate through a set of key/value pairs.
// Order of iteration is NON-DETERMINISTIC in general
type KVIterator interface {
	Iterate(func(k, v []byte) bool)
	IterateKeys(func(k []byte) bool)
}

// KVBatchedWriter collects mutations in the buffer via Set-s to KVWriter and then flushes (applies) it atomically to DB with Commit
// KVBatchedWriter implementation should be deterministic: the sequence of Set-s to KWWriter exactly determines
// the sequence, how key/value pairs in the database are updated or deleted (with value == nil)
type KVBatchedWriter interface {
	KVWriter
	Commit() error
}

// KVStore is a compound interface
type KVStore interface {
	KVReader
	KVWriter
	KVIterator
}

// BatchedUpdatable is a KVStore equipped with the batched update capability. You can only update
// BatchedUpdatable in atomic batches
type BatchedUpdatable interface {
	BatchedWriter() KVBatchedWriter
}

// Traversable is an interface which provides with partial iterators
type Traversable interface {
	Iterator(prefix []byte) KVIterator
}

// CopyAll flushes KVIterator to KVWriter. It is up to the iterator correctly stop iterating
func CopyAll(dst KVWriter, src KVIterator) {
	src.Iterate(func(k, v []byte) bool {
		dst.Set(k, v)
		return true
	})
}

type readerPartition struct {
	prefix byte
	r      KVReader
}

func (p *readerPartition) Get(key []byte) []byte {
	return p.r.Get(Concat(p.prefix, key))
}

func (p *readerPartition) Has(key []byte) bool {
	return p.r.Has(Concat(p.prefix, key))
}

func MakeReaderPartition(r KVReader, prefix byte) KVReader {
	return &readerPartition{
		prefix: prefix,
		r:      r,
	}
}

type writerPartition struct {
	prefix byte
	w      KVWriter
}

func (w *writerPartition) Set(key, value []byte) {
	w.w.Set(Concat(w.prefix, key), value)
}

func MakeWriterPartition(w KVWriter, prefix byte) KVWriter {
	return &writerPartition{
		prefix: prefix,
		w:      w,
	}
}
