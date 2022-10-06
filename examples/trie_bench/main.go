package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/kvstore/badger"
	"github.com/iotaledger/hive.go/core/kvstore/mapdb"
	model2 "github.com/iotaledger/trie.go/common"
	"github.com/iotaledger/trie.go/hive_adaptor"
	"github.com/iotaledger/trie.go/models/trie_blake2b"
	"github.com/iotaledger/trie.go/models/trie_blake2b/trie_blake2b_verify"
	"github.com/iotaledger/trie.go/mutable"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/xerrors"
)

const usage = "USAGE: trie_bench [-n=<num kv pairs>] [-blake2b=20|32]" +
	"[-arity=2|16|26] [-valuethr=<terminal optimization threshold>]" +
	"[maxkey=<max key size>] [maxvalue=<max value size>]" +
	"<gen|mkdbbadger|mkdbmem|scandbbadger|mkdbbadgernotrie> <name>\n"

var (
	model    *trie_blake2b.CommitmentModel
	hashsize = flag.Int("blake2b", 20, "must be 20 or 32")
	arityPar = flag.Int("arity", 16, "must be 2, 16 or 256")
	num      = flag.Int("n", 1000, "number of k/v pairs")
	hashkv   = flag.Bool("hashkv", false, "hash keys and values")
	optterm  = flag.Int("valuethr", 0, "commitments to values longer that parameter won't be saved in the try")
	maxKey   = flag.Int("maxkey", MaxKey, "maximum size of the generated key")
	maxValue = flag.Int("maxvalue", MaxValue, "maximum size of the generated value")
	cmd      string
	name     string
	fname    string
	dbdir    string
)

func main() {
	flag.Parse()
	tail := flag.Args()
	if len(tail) < 2 {
		fmt.Printf(usage)
		os.Exit(1)
	}
	cmd = tail[0]

	switch cmd {
	case "gen", "mkdbbadger", "mkdbmem", "scandbbadger", "mkdbbadgernotrie":
	default:
		fmt.Printf(usage)
		os.Exit(1)
	}
	name = tail[1]
	var arity model2.PathArity
	switch *arityPar {
	case 2:
		arity = model2.PathArity2
	case 16:
		arity = model2.PathArity16
	case 256:
		arity = model2.PathArity256
	default:
		fmt.Printf(usage)
		os.Exit(1)
	}

	switch *hashsize {
	case 20:
		model = trie_blake2b.New(arity, trie_blake2b.HashSize160, *optterm)
	case 32:
		model = trie_blake2b.New(arity, trie_blake2b.HashSize256, *optterm)
	default:
		fmt.Printf(usage)
		os.Exit(1)
	}
	fmt.Printf("Commitment common: '%s'\n", model.Description())
	fmt.Printf("Terminal optimization threshold: %d\n", *optterm)
	fname = name + ".bin"
	dbdir = fmt.Sprintf("%s.%d.%d.%d.dbdir", name, *hashsize, *arityPar, *optterm)

	switch cmd {
	case "gen":
		fmt.Printf("number of key/value pairs to generate: %d\n", *num)
		fmt.Printf("maximum key length: %d\n", *maxKey)
		fmt.Printf("maximum value length: %d\n", *maxValue)
		if *hashkv {
			fmt.Printf("generated keys and values will be hashed into 32 bytes\n")
		}
		fmt.Printf("generating file '%s'\n", fname)
		genrnd()

	case "mkdbbadgernotrie":
		dbdir += ".notrie"
		mkdbbadgerNoTrie()

	case "mkdbbadger":
		mkdbbadger()

	case "mkdbmem":
		mkdbmem()

	case "scandbbadger":
		scandbbadger()

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

func genrnd() {
	rndIterator := model2.NewRandStreamIterator(model2.RandStreamParams{
		Seed:       time.Now().UnixNano(),
		NumKVPairs: *num,
		MaxKey:     *maxKey,
		MaxValue:   *maxValue,
	})
	fileWriter, err := model2.CreateKVStreamFile(fname)
	must(err)
	defer func() { _ = fileWriter.Close() }()

	count := 0
	wrote := 0
	err = rndIterator.Iterate(func(k []byte, v []byte) bool {
		if (count+1)%100000 == 0 {
			fmt.Printf("writing key/value pair %d. Wrote %d bytes\n", count+1, wrote)
		}
		if *hashkv {
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

func mkdbmem() {
	kvs := mapdb.NewMapDB()
	file2kvs(kvs)
}

// all value and trie in badger db. Flushes every 100_000 records

func mkdbbadger() {
	if _, err := os.Stat(dbdir); !os.IsNotExist(err) {
		fmt.Printf("directory %s already exists. Can't create new database\n", dbdir)
		os.Exit(1)
	}
	fmt.Printf("creating new Badger database '%s'\n", dbdir)

	db, err := badger.CreateDB(dbdir)
	must(err)
	defer func() { _ = db.Close() }()

	kvs := badger.New(db)
	must(err)

	file2kvs(kvs)
}

func mkdbbadgerNoTrie() {
	if _, err := os.Stat(dbdir); !os.IsNotExist(err) {
		fmt.Printf("directory %s already exists. Can't create new database\n", dbdir)
		os.Exit(1)
	}
	fmt.Printf("creating new Badger database. No trie '%s'\n", dbdir)

	db, err := badger.CreateDB(dbdir)
	must(err)
	defer func() { _ = db.Close() }()

	kvs := badger.New(db)
	must(err)

	file2kvsNoTrie(kvs)
}

func scandbbadger() {
	if _, err := os.Stat(dbdir); os.IsNotExist(err) {
		fmt.Printf("directory %s does not exist\n", dbdir)
		os.Exit(1)
	}
	fmt.Printf("opening database '%s'\n", dbdir)

	db, err := badger.CreateDB(dbdir)
	must(err)
	defer func() { _ = db.Close() }()

	kvs := badger.New(db)
	trieKVS := hive_adaptor.NewHiveKVStoreAdaptor(kvs, triePrefix)
	valueKVS := hive_adaptor.NewHiveKVStoreAdaptor(kvs, valueStorePrefix)

	//trie.DangerouslyDumpToConsole("----- VALUES ------", valueKVS)
	//trie.DangerouslyDumpToConsole("----- TRIE ------", trieKVS)

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

	tr := mutable.NewTrieReader(model, trieKVS, valueKVS)
	root := mutable.RootCommitment(tr)
	fmt.Printf("root commitment: %s\n", root)

	recCounter = 1
	proofBytes := 0
	proofLen := 0
	tm := newTimer()
	valueKVS.Iterate(func(k []byte, v []byte) bool {
		proof := model.Proof(k, tr)
		proofBytes += len(proof.Bytes())
		proofLen += len(proof.Path)
		err = trie_blake2b_verify.Validate(proof, root.Bytes())
		must(err)

		tc := proof.Path[len(proof.Path)-1].Terminal
		tc1, _ := trie_blake2b.CommitToDataRaw(v, model.HashSize())
		if !bytes.Equal(tc1, tc) {
			err = xerrors.New("invalid proof: terminal commitment and terminal proof are not equal")
		}
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

func file2kvs(kvs kvstore.KVStore) {
	streamIn, err := model2.OpenKVStreamFile(fname)
	must(err)
	defer func() { _ = streamIn.Close() }()

	tm := newTimer()
	counterRec := 1
	tr := mutable.NewTrieReader(model, hive_adaptor.NewHiveKVStoreAdaptor(kvs, triePrefix), nil)
	updater, err := hive_adaptor.NewHiveBatchedUpdater(kvs, model, triePrefix, valueStorePrefix)
	must(err)
	var mem runtime.MemStats
	err = streamIn.Iterate(func(k []byte, v []byte) bool {
		updater.Set(k, v)
		if counterRec%flushEach == 0 {
			must(updater.Commit())
			runtime.ReadMemStats(&mem)

			fmt.Printf("commited %d records. rec/sec: %v, mem alloc: %f MB\n",
				counterRec, counterRec/int(tm.Duration().Seconds()),
				float32(mem.Alloc)/(1024*1024),
			)
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

	fmt.Printf("root commitment: %s\n", mutable.RootCommitment(tr))
}

func file2kvsNoTrie(kvs kvstore.KVStore) {
	streamIn, err := model2.OpenKVStreamFile(fname)
	must(err)
	defer func() { _ = streamIn.Close() }()

	tm := newTimer()
	counterRec := 1
	must(err)

	var batched kvstore.BatchedMutations
	var mem runtime.MemStats
	err = streamIn.Iterate(func(k []byte, v []byte) bool {
		if batched == nil {
			batched, err = kvs.Batched()
			must(err)
		}
		must(batched.Set(k, v))
		if counterRec%10_000 == 0 {
			must(batched.Commit())
			batched = nil
			must(kvs.Flush())
			runtime.ReadMemStats(&mem)

			sec := int(tm.Duration().Seconds())
			if sec == 0 {
				sec = 1
			}
			fmt.Printf("wrote %d records. rec/sec: %v, mem alloc: %f MB\n",
				counterRec, counterRec/sec,
				float32(mem.Alloc)/(1024*1024),
			)
		}
		counterRec++
		return true
	})
	must(err)
	fmt.Printf("Speed: %f records/sec\n", float64(counterRec)/tm.Duration().Seconds())
}

func newTimer() timer {
	return timer(time.Now())
}

func (t timer) Duration() time.Duration {
	return time.Now().Sub(time.Time(t))
}
