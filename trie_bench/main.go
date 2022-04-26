package main

import (
	"fmt"
	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/kvstore/badger"
	"github.com/iotaledger/hive.go/kvstore/mapdb"
	trie_go "github.com/iotaledger/trie.go"
	"github.com/iotaledger/trie.go/hive_adaptor"
	"github.com/iotaledger/trie.go/trie256p"
	"github.com/iotaledger/trie.go/trie_blake2b"
	"golang.org/x/crypto/blake2b"
	"os"
	"strconv"
	"time"
)

const usage = "generate random key/value pairs. USAGE: trie_bench -gen <size> <name>\n" +
	"generate random key/value pairs with 32 byte random keys. USAGE: trie_bench -genhash <size> <name>\n" +
	"make badger DB with trie from file. USAGE: trie_bench -mkdbbadger <name>\n" +
	"make in-memory DB with trie from file. USAGE: trie_bench -mkdbmem <name>\n" +
	"check consistency of the DB. USAGE: trie_bench -scandbbadger <name>\n"

func main() {
	if len(os.Args) < 2 {
		fmt.Printf(usage)
		os.Exit(1)
	}
	switch os.Args[1] {
	case "-gen":
		if len(os.Args) != 4 {
			fmt.Printf(usage)
			os.Exit(1)
		}
		size, err := strconv.Atoi(os.Args[2])
		must(err)
		genrnd(size, os.Args[3], false)

	case "-genhash":
		if len(os.Args) != 4 {
			fmt.Printf(usage)
			os.Exit(1)
		}
		size, err := strconv.Atoi(os.Args[2])
		must(err)
		genrnd(size, os.Args[3], true)

	case "-mkdbbadger":
		if len(os.Args) != 3 {
			fmt.Printf(usage)
			os.Exit(1)
		}
		mkdbbadger(os.Args[2])

	case "-mkdbmem":
		if len(os.Args) != 3 {
			fmt.Printf(usage)
			os.Exit(1)
		}
		mkdbmem(os.Args[2])

	case "-scandbbadger":
		if len(os.Args) != 3 {
			fmt.Printf(usage)
			os.Exit(1)
		}
		scandbbadger(os.Args[2])

	default:
		fmt.Printf(usage)
		os.Exit(1)
	}
}

func must(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

const (
	MaxKey   = 100
	MaxValue = 32
)

func genrnd(size int, name string, hashKV bool) {
	rndIterator := trie_go.NewRandStreamIterator(trie_go.RandStreamParams{
		Seed:       time.Now().UnixNano(),
		NumKVPairs: size,
		MaxKey:     MaxKey,
		MaxValue:   MaxValue,
	})
	fname := name + ".bin"
	fileWriter, err := trie_go.CreateKVStreamFile(fname)
	must(err)
	defer fileWriter.Close()

	count := 0
	wrote := 0
	err = rndIterator.Iterate(func(k []byte, v []byte) bool {
		if (count+1)%100000 == 0 {
			fmt.Printf("writing key/value pair %d. Wrote %d bytes\n", count+1, wrote)
		}
		if hashKV {
			t := blake2b.Sum256(k)
			k = t[:]
			if len(v) > 0 {
				t = blake2b.Sum256(v)
				v = t[:]
			}
		}
		must(fileWriter.Write(k, v))
		count++
		wrote += len(k) + len(v) + 6
		return true
	})
	must(err)
	fmt.Printf("generated total %d key/value pairs, %f MB\n", count+1, float32(wrote)/(1024*1024))
}

// all values loads in memory
const flushEach = 100_000

func mkdbmem(name string) {
	fname := name + ".bin"
	kvs := mapdb.NewMapDB()
	file2kvs(fname, kvs)
}

// all value and trie in badger db. Flushes every 100_000 records

func mkdbbadger(name string) {
	dbDir := name + ".dbdir"
	fname := name + ".bin"
	if _, err := os.Stat(dbDir); !os.IsNotExist(err) {
		fmt.Printf("directory %s already exists. Can't create new database\n", dbDir)
		os.Exit(1)
	}
	fmt.Printf("creating new database '%s'\n", dbDir)

	db, err := badger.CreateDB(dbDir)
	must(err)
	kvs := badger.New(db)
	must(err)

	file2kvs(fname, kvs)
}

func scandbbadger(name string) {
	dbDir := name + ".dbdir"
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		fmt.Printf("directory %s does not exist\n", dbDir)
		os.Exit(1)
	}
	fmt.Printf("opening database '%s'\n", dbDir)

	db, err := badger.CreateDB(dbDir)
	must(err)
	kvs := badger.New(db)
	trieKVS := hive_adaptor.NewHiveKVStoreAdaptor(kvs, triePrefix)
	valueKVS := hive_adaptor.NewHiveKVStoreAdaptor(kvs, valueStorePrefix)

	recCounter := 0
	keyByteCounter := 0
	valueKVS.Iterate(func(k []byte, v []byte) bool {
		recCounter++
		keyByteCounter += len(k)
		return true
	})
	fmt.Printf("K/V STORAGE: number of key/value pairs: %d, avg key len: %d\n",
		recCounter, keyByteCounter/recCounter)

	recCounter = 0
	keyByteCounter = 0
	valueByteCounter := 0
	trieKVS.Iterate(func(k []byte, v []byte) bool {
		recCounter++
		keyByteCounter += len(k)
		valueByteCounter += len(v)
		return true
	})
	fmt.Printf("TRIE: number of nodes: %d, avg key len: %d, avg node size: %d\n",
		recCounter, keyByteCounter/recCounter, valueByteCounter/recCounter)

	model := trie_blake2b.New()
	trie := trie256p.NewNodeStoreReader(trieKVS, model)
	root := trie256p.RootCommitment(trie)
	fmt.Printf("root commitment: %s\n", root)

	recCounter = 1
	proofBytes := 0
	proofLen := 0
	tm := newTimer()
	valueKVS.Iterate(func(k []byte, v []byte) bool {
		proof := model.Proof(k, trie)
		proofBytes += len(proof.Bytes())
		proofLen += len(proof.Path)
		err = proof.Validate(root, v)
		must(err)
		if recCounter%flushEach == 0 {
			fmt.Printf("validated %d records in %v, %f proof/sec, avg proof bytes %d, avg proof len %f\n",
				recCounter, tm.Duration(), float64(recCounter)/tm.Duration().Seconds(),
				proofBytes/recCounter, float32(proofLen)/float32(recCounter))
		}
		recCounter++
		return true
	})
}

type timer time.Time

var (
	triePrefix       = []byte{0x01}
	valueStorePrefix = []byte{0x02}
)

func file2kvs(fname string, kvs kvstore.KVStore) {
	streamIn, err := trie_go.OpenKVStreamFile(fname)
	must(err)
	defer streamIn.Close()

	tm := newTimer()
	counterRec := 1
	model := trie_blake2b.New()
	trie := trie256p.NewNodeStoreReader(hive_adaptor.NewHiveKVStoreAdaptor(kvs, triePrefix), model)
	updater, err := hive_adaptor.NewHiveBatchedUpdater(kvs, model, triePrefix, valueStorePrefix)
	must(err)
	err = streamIn.Iterate(func(k []byte, v []byte) bool {
		updater.Update(k, v)
		if counterRec%flushEach == 0 {
			must(updater.Commit())
			fmt.Printf("commited %d records. Duration: %v\n", counterRec, tm.Duration())
		}
		counterRec++
		return true
	})
	must(err)
	if updater != nil {
		must(updater.Commit())
		fmt.Printf("commited %d records. Duration: %v\n", counterRec, tm.Duration())
	}
	fmt.Printf("Speed: %f records/sec\n", float64(counterRec)/tm.Duration().Seconds())

	fmt.Printf("root commitment: %s\n", trie256p.RootCommitment(trie))
}

func newTimer() timer {
	return timer(time.Now())
}

func (t timer) Duration() time.Duration {
	return time.Now().Sub(time.Time(t))
}
