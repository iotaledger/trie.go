package hive_adaptor

import (
	"errors"
	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/trie.go"
	"github.com/iotaledger/trie.go/trie256p"
)

// HiveKVStoreAdaptor maps a partition of the Hive KVStore to trie_go.KVStore
type HiveKVStoreAdaptor struct {
	kvs    kvstore.KVStore
	prefix []byte
}

func NewHiveKVStoreAdaptor(kvs kvstore.KVStore, prefix []byte) *HiveKVStoreAdaptor {
	return &HiveKVStoreAdaptor{kvs: kvs, prefix: prefix}
}

func mustNoErr(err error) {
	if err != nil {
		panic(err)
	}
}

func makeKey(prefix, k []byte) []byte {
	if len(prefix) == 0 {
		return k
	}
	return trie_go.Concat(prefix, k)
}

func (kvs *HiveKVStoreAdaptor) Get(key []byte) []byte {
	v, err := kvs.kvs.Get(makeKey(kvs.prefix, key))
	if errors.Is(err, kvstore.ErrKeyNotFound) {
		return nil
	}
	mustNoErr(err)
	return v
}

func (kvs *HiveKVStoreAdaptor) Has(key []byte) bool {
	v, err := kvs.kvs.Has(makeKey(kvs.prefix, key))
	mustNoErr(err)
	return v
}

func (kvs *HiveKVStoreAdaptor) Set(key, value []byte) {
	var err error
	if len(value) == 0 {
		err = kvs.kvs.Delete(makeKey(kvs.prefix, key))
	} else {
		err = kvs.kvs.Set(makeKey(kvs.prefix, key), value)
	}
	mustNoErr(err)
}

func (kvs *HiveKVStoreAdaptor) Iterate(fun func(k []byte, v []byte) bool) {
	err := kvs.kvs.Iterate(kvs.prefix, func(key kvstore.Key, value kvstore.Value) bool {
		return fun(key[len(kvs.prefix):], value)
	})
	mustNoErr(err)
}

// HiveBatchedUpdater implements buffering and flush updates in batches, both k/v pairs and trie.
// Dramatically improves speed
type HiveBatchedUpdater struct {
	kvs              kvstore.KVStore
	batch            kvstore.BatchedMutations
	wTrie            batchWriter
	wValue           batchWriter
	triePrefix       []byte
	valueStorePrefix []byte
	trie             *trie256p.Trie
}

func NewHiveBatchedUpdater(kvs kvstore.KVStore, model trie256p.CommitmentModel, triePrefix, storePrefix []byte) (*HiveBatchedUpdater, error) {
	ret := &HiveBatchedUpdater{
		kvs:              kvs,
		trie:             trie256p.New(model, NewHiveKVStoreAdaptor(kvs, triePrefix)),
		triePrefix:       triePrefix,
		valueStorePrefix: storePrefix,
	}
	return ret, nil
}

func (a *HiveBatchedUpdater) Update(key []byte, value []byte) {
	var err error
	if a.batch == nil {
		a.batch, err = a.kvs.Batched()
		mustNoErr(err)
		a.wTrie = newBatchWriter(a.batch, a.triePrefix)
		a.wValue = newBatchWriter(a.batch, a.valueStorePrefix)
	}
	a.wValue.Set(key, value)
	a.trie.Update(key, value)
}

// batchWriter implements KVWriter interface over the Hive batch
type batchWriter struct {
	prefix []byte
	batch  kvstore.BatchedMutations
}

func newBatchWriter(b kvstore.BatchedMutations, prefix []byte) batchWriter {
	return batchWriter{
		prefix: prefix,
		batch:  b,
	}
}

func (b batchWriter) Set(key, value []byte) {
	var err error
	if len(value) > 0 {
		err = b.batch.Set(makeKey(b.prefix, key), value)
	} else {
		err = b.batch.Delete(makeKey(b.prefix, key))
	}
	mustNoErr(err)
}

func (a *HiveBatchedUpdater) Commit() error {
	if a.batch == nil {
		return nil
	}
	a.trie.Commit()
	a.trie.PersistMutations(a.wTrie)
	if err := a.batch.Commit(); err != nil {
		return err
	}
	a.trie.ClearCache()
	a.batch = nil
	return nil
}