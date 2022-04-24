package main

import (
	"fmt"
	"github.com/iotaledger/hive.go/kvstore/badger"
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
	"check consistency of the DB. USAGE: trie_bench -checkdb <name>\n"

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

	case "-checkdb":
		if len(os.Args) != 3 {
			fmt.Printf(usage)
			os.Exit(1)
		}
		checkdb(os.Args[2])

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
	MaxKey   = 64
	MaxValue = 128
)

func genrnd(size int, name string, hashkeys bool) {
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
		if hashkeys {
			t := blake2b.Sum256(k)
			k = t[:]
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

func mkdbmem(name string) {
	triePartition := trie_go.NewInMemoryKVStore()
	kvPartition := trie_go.NewInMemoryKVStore()
	trie := trie256p.New(trie_blake2b.New(), triePartition)

	fname := name + ".bin"
	streamIn, err := trie_go.OpenKVStreamFile(fname)
	must(err)
	defer streamIn.Close()

	tm := NewTimer()
	counterRec := 0
	err = streamIn.Iterate(func(k []byte, v []byte) bool {
		kvPartition.Set(k, v)
		trie.Update(k, v)
		counterRec++
		return true
	})
	trie.Commit()
	counterMut := trie.PersistMutations(triePartition)
	trie.ClearCache()
	must(err)

	fmt.Printf("total records: %d\ntotal trie nodes: %d\n", counterRec, counterMut)
	fmt.Printf("Speed: %f records/sec\n", float64(counterRec)/tm.Duration().Seconds())
}

var (
	triePrefix       = []byte{0x01}
	valueStorePrefix = []byte{0x02}
)

const flushEach = 100_000

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
	triePartition := hive_adaptor.NewHiveKVStoreAdaptor(kvs, triePrefix)
	trie := trie256p.New(trie_blake2b.New(), triePartition)
	var updater *hive_adaptor.HiveBatchedUpdater
	must(err)

	streamIn, err := trie_go.OpenKVStreamFile(fname)
	must(err)
	defer streamIn.Close()

	tm := NewTimer()
	counterRec := 1
	err = streamIn.Iterate(func(k []byte, v []byte) bool {
		if updater == nil {
			updater, err = hive_adaptor.NewHiveBatchedUpdater(kvs, trie, triePrefix, valueStorePrefix)
			must(err)
		}
		updater.Update(k, v)

		if counterRec%flushEach == 0 {
			must(updater.Commit())
			updater = nil
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
}

func checkdb(name string) {
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
	model := trie_blake2b.New()
	trie := trie256p.NewNodeStoreReader(trieKVS, model)
	values := hive_adaptor.NewHiveKVStoreAdaptor(kvs, valueStorePrefix)

	rootC := trie256p.RootCommitment(trie)
	fmt.Printf("root commitment: %s\n", rootC)

	counter := 0
	values.Iterate(func(k []byte, v []byte) bool {
		p := model.Proof(k, trie)
		err = p.Validate(rootC)
		must(err)
		if counter%flushEach == 0 {
			fmt.Printf("validate %d records\n")
		}
		counter++
		return true
	})
}

type timer time.Time

func NewTimer() timer {
	return timer(time.Now())
}

func (t timer) Duration() time.Duration {
	return time.Now().Sub(time.Time(t))
}
