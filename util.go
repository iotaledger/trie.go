// Package trie_go contains key/value related functions used for testing (otherwise of general nature)
package trie_go

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"time"
)

func Size(o interface{ Write(w io.Writer) error }) (int, error) {
	var ret byteCounter
	if err := o.Write(&ret); err != nil {
		return 0, err
	}
	return int(ret), nil
}

func MustSize(o interface{ Write(w io.Writer) error }) int {
	ret, err := Size(o)
	if err != nil {
		panic(err)
	}
	return ret
}

func Assert(cond bool, format string, p ...interface{}) {
	if !cond {
		panic(fmt.Sprintf(format, p...))
	}
}

func Concat(par ...interface{}) []byte {
	var buf bytes.Buffer
	for _, p := range par {
		switch p := p.(type) {
		case []byte:
			buf.Write(p)
		case byte:
			buf.WriteByte(p)
		case string:
			buf.Write([]byte(p))
		case interface{ Bytes() []byte }:
			buf.Write(p.Bytes())
		default:
			Assert(false, "Concat: unsupported type %T", p)
		}
	}
	return buf.Bytes()
}

func ByteSize(s KVIterator) int {
	accLen := 0
	s.Iterate(func(k, v []byte) bool {
		accLen += len(k) + len(v)
		return true
	})
	return accLen
}

func NumEntries(s KVIterator) int {
	ret := 0
	s.Iterate(func(_, _ []byte) bool {
		ret++
		return true
	})
	return ret
}

func DumpToFile(r KVIterator, fname string) (int, error) {
	file, err := os.Create(fname)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var bytesTotal int
	r.Iterate(func(k, v []byte) bool {
		n, errw := writeKV(file, k, v)
		if errw != nil {
			err = errw
			return false
		}
		bytesTotal += n
		return true
	})
	return bytesTotal, err
}

func UnDumpFromFile(w KVWriter, fname string) (int, error) {
	file, err := os.Open(fname)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var k, v []byte
	var exit bool
	n := 0
	for {
		if k, v, exit = readKV(file); exit {
			break
		}
		n += len(k) + len(v) + 6
		w.Set(k, v)
	}
	return n, nil
}

func writeKV(w io.Writer, k, v []byte) (int, error) {
	if err := WriteBytes16(w, k); err != nil {
		return 0, err
	}
	if err := WriteBytes32(w, v); err != nil {
		return len(k) + 2, err
	}
	return len(k) + len(v) + 6, nil
}

func readKV(r io.Reader) ([]byte, []byte, bool) {
	k, err := ReadBytes16(r)
	if errors.Is(err, io.EOF) {
		return nil, nil, true
	}
	v, err := ReadBytes32(r)
	if err != nil {
		panic(err)
	}
	return k, v, false
}

type randStreamIterator struct {
	rnd   *rand.Rand
	par   RandStreamParams
	count int
}

type RandStreamParams struct {
	Seed       int64
	NumKVPairs int // 0 means infinite
	MaxKey     int // max length of key
	MaxValue   int // max length of value
}

func NewRandStreamIterator(p ...RandStreamParams) *randStreamIterator {
	ret := &randStreamIterator{
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

func (r *randStreamIterator) Iterate(fun func(k []byte, v []byte) bool) error {
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
