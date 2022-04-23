package trie_go

import (
	"github.com/iotaledger/hive.go/kvstore"
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

func (kvs *HiveKVStoreAdaptor) makeKey(k []byte) []byte {
	if len(kvs.prefix) == 0 {
		return k
	}
	return Concat(kvs.prefix, k)
}
func (kvs *HiveKVStoreAdaptor) Get(key []byte) []byte {
	v, err := kvs.kvs.Get(kvs.makeKey(key))
	mustNoErr(err)
	return v
}

func (kvs *HiveKVStoreAdaptor) Has(key []byte) bool {
	v, err := kvs.kvs.Has(kvs.makeKey(key))
	mustNoErr(err)
	return v
}

func (kvs *HiveKVStoreAdaptor) Set(key, value []byte) {
	var err error
	if len(value) == 0 {
		err = kvs.kvs.Delete(kvs.makeKey(key))
	} else {
		err = kvs.kvs.Set(kvs.makeKey(key), value)
	}
	mustNoErr(err)
}

func (kvs *HiveKVStoreAdaptor) Iterate(fun func(k []byte, v []byte) bool) {
	err := kvs.kvs.Iterate(kvs.prefix, func(key kvstore.Key, value kvstore.Value) bool {
		return fun(key[len(kvs.prefix):], value)
	})
	mustNoErr(err)
}
