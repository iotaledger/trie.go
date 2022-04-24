package hive_adaptor

import (
	"errors"
	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/trie.go"
	"github.com/iotaledger/trie.go/trie256p"
)

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

type batchPartition struct {
	prefix []byte
	batch  kvstore.BatchedMutations
}

func newBatchPartition(prefix []byte, batch kvstore.BatchedMutations) *batchPartition {
	return &batchPartition{
		prefix: prefix,
		batch:  batch,
	}
}

func (k *batchPartition) Set(key, value []byte) {
	if err := k.batch.Set(makeKey(k.prefix, key), value); err != nil {
		panic(err)
	}
}

func (k *batchPartition) Del(key []byte) {
	if err := k.batch.Delete(makeKey(k.prefix, key)); err != nil {
		panic(err)
	}
}

type HiveBatchedUpdater struct {
	kvs             kvstore.KVStore
	batch           kvstore.BatchedMutations
	trieBatch       *batchPartition
	valueStoreBatch *batchPartition
	trie            *trie256p.Trie
}

func NewHiveBatchedUpdater(kvs kvstore.KVStore, trie *trie256p.Trie, triePrefix, storePrefix []byte) (*HiveBatchedUpdater, error) {
	b, err := kvs.Batched()
	if err != nil {
		return nil, err
	}
	ret := &HiveBatchedUpdater{
		kvs:             kvs,
		trieBatch:       newBatchPartition(triePrefix, b),
		valueStoreBatch: newBatchPartition(storePrefix, b),
		trie:            trie,
		batch:           b,
	}
	return ret, nil
}

func (a *HiveBatchedUpdater) Update(key []byte, value []byte) {
	if len(value) > 0 {
		a.valueStoreBatch.Set(key, value)
	} else {
		a.valueStoreBatch.Del(key)
	}
	a.trie.Update(key, value)
}

func (a *HiveBatchedUpdater) Commit() error {
	a.trie.PersistMutations(a.trieBatch)
	a.trie.ClearCache()
	return a.batch.Commit()
}
