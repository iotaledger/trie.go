package main

import (
	"fmt"
	"github.com/iotaledger/hive.go/kvstore/rocksdb"
	trie_go "github.com/iotaledger/trie.go"
	"github.com/iotaledger/trie.go/trie256p"
	"github.com/iotaledger/trie.go/trie_blake2b"
	"golang.org/x/crypto/blake2b"
	"os"
	"strconv"
	"time"
)

const usage = "generate random key/value pairs. USAGE: trie_bench -gen <size> <filename>\n" +
	"generate random key/value pairs with 32 byte random keys. USAGE: trie_bench -genhash <size> <filename>\n" +
	"make rocksdb with trie from file. USAGE: trie_bench -mkdb <filename>\n"

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
		checkErr(err)
		genrnd(size, os.Args[3], false)

	case "-genhash":
		if len(os.Args) != 4 {
			fmt.Printf(usage)
			os.Exit(1)
		}
		size, err := strconv.Atoi(os.Args[2])
		checkErr(err)
		genrnd(size, os.Args[3], true)

	case "-mkdb":
		if len(os.Args) != 3 {
			fmt.Printf(usage)
			os.Exit(1)
		}
		mkdb(os.Args[2])

	default:
		fmt.Printf(usage)
		os.Exit(1)
	}
}

func checkErr(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

const (
	MaxKey   = 64
	MaxValue = 128
)

func genrnd(size int, fname string, hashkeys bool) {
	rndIterator := trie_go.NewRandStreamIterator(trie_go.RandStreamParams{
		Seed:       time.Now().UnixNano(),
		NumKVPairs: size,
		MaxKey:     MaxKey,
		MaxValue:   MaxValue,
	})
	fileWriter, err := trie_go.CreateKVStreamFile(fname)
	checkErr(err)
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
		checkErr(fileWriter.Write(k, v))
		count++
		wrote += len(k) + len(v) + 6
		return true
	})
	checkErr(err)
	fmt.Printf("generated total %d key/value pairs, %f MB\n", count+1, float32(wrote)/(1024*1024))
}

const flushEach = 10_000

func mkdb(fname string) {
	dbDir := fname + ".dbdir"
	if _, err := os.Stat(dbDir); !os.IsNotExist(err) {
		fmt.Printf("directory %s already exists. Can't create new database\n", dbDir)
		os.Exit(1)
	}
	fmt.Printf("creating new database '%s'\n", dbDir)

	db, err := rocksdb.CreateDB(dbDir)
	checkErr(err)
	hiveKVStore := rocksdb.New(db)
	kvPartition := trie_go.NewHiveKVStoreAdaptor(hiveKVStore, []byte{0x01})
	triePartition := trie_go.NewHiveKVStoreAdaptor(hiveKVStore, []byte{0x02})
	trie := trie256p.New(trie_blake2b.New(), triePartition)

	streamIn, err := trie_go.OpenKVStreamFile(fname)
	checkErr(err)
	defer streamIn.Close()

	counter := 1
	err = streamIn.Iterate(func(k []byte, v []byte) bool {
		if counter%flushEach == 0 {
			trie.Commit()
			trie.PersistMutations(triePartition)
			checkErr(db.Flush())
			fmt.Printf("commited %d records\n", counter)
		}
		kvPartition.Set(k, v)
		trie.Update(k, v)
		counter++
		return true
	})
	checkErr(err)
}
