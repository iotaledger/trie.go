package common

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"

	"golang.org/x/crypto/blake2b"
)

// CheckNils returns (conclusive comparison result, true) if at least one is nil
// return (false, false) if both are not nil and can both be safely dereferenced
func CheckNils(i1, i2 interface{}) (bool, bool) {
	// TODO better suggestion? The problem: type(nil) != nil
	i1Nil := i1 == nil || (reflect.ValueOf(i1).Kind() == reflect.Ptr && reflect.ValueOf(i1).IsNil())
	i2Nil := i2 == nil || (reflect.ValueOf(i2).Kind() == reflect.Ptr && reflect.ValueOf(i2).IsNil())
	if i1Nil && i2Nil {
		return true, true
	}
	if i1Nil || i2Nil {
		return false, true
	}
	return false, false
}

// MustBytes most common way of serialization
func MustBytes(o interface{ Write(w io.Writer) error }) []byte {
	var buf bytes.Buffer
	if err := o.Write(&buf); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// byteCounter simple byte counter as io.Writer
type byteCounter int

func (b *byteCounter) Write(p []byte) (n int, err error) {
	*b = byteCounter(int(*b) + len(p))
	return 0, nil
}

// Size calculates byte size of the serializable object
func Size(o interface{ Write(w io.Writer) error }) (int, error) {
	var ret byteCounter
	if err := o.Write(&ret); err != nil {
		return 0, err
	}
	return int(ret), nil
}

// MustSize calculates byte size of the serializable object
func MustSize(o interface{ Write(w io.Writer) error }) int {
	ret, err := Size(o)
	if err != nil {
		panic(err)
	}
	return ret
}

// Assert simple assertion with message formatting
func Assert(cond bool, format string, p ...interface{}) {
	if !cond {
		panic(fmt.Sprintf(format, p...))
	}
}

// Concat concatenates bytes of byte-able objects
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

// ByteSize computes byte size of the serialized key/value iterator
// assumes 2 bytes per key length and 4 bytes per value length
func ByteSize(s KVIterator) int {
	accLen := 0
	s.Iterate(func(k, v []byte) bool {
		accLen += len(k) + 2 + len(v) + 4
		return true
	})
	return accLen
}

// NumEntries calculates number of key/value pair in the iterator
func NumEntries(s KVIterator) int {
	ret := 0
	s.Iterate(func(_, _ []byte) bool {
		ret++
		return true
	})
	return ret
}

// DumpToFile serializes iterator to the file in binary form.
// The content of the file in general is non-deterministic due to the random order of iteration
func DumpToFile(r KVIterator, fname string) (int, error) {
	file, err := os.Create(fname)
	if err != nil {
		return 0, err
	}
	defer func() { _ = file.Close() }()

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

func DangerouslyDumpToConsole(title string, r KVIterator) {
	counter := 0
	fmt.Printf("%s\n", title)
	r.Iterate(func(k, v []byte) bool {
		fmt.Printf("%d: '%x' ::: '%x'\n", counter, k, v)
		counter++
		return true
	})
}

// UnDumpFromFile restores dumped set of key/value pairs into the key/value writer
func UnDumpFromFile(w KVWriter, fname string) (int, error) {
	file, err := os.Open(fname)
	if err != nil {
		return 0, err
	}
	defer func() { _ = file.Close() }()

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

// writeKV serializes key/value pair into the io.Writer. 2 and 4 little endian bytes for respectively key length and value length
func writeKV(w io.Writer, k, v []byte) (int, error) {
	if err := WriteBytes16(w, k); err != nil {
		return 0, err
	}
	if err := WriteBytes32(w, v); err != nil {
		return len(k) + 2, err
	}
	return len(k) + len(v) + 6, nil
}

// readKV deserializes key/value pair from io.Reader. Returns key/value pair and an error flag if not enough data
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

// ---------------------------------------------------------------------------
// r/w utility functions
// TODO rewrite with generics when switch to Go 1.18

func ReadBytes8(r io.Reader) ([]byte, error) {
	length, err := ReadByte(r)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return []byte{}, nil
	}
	ret := make([]byte, length)
	_, err = r.Read(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func WriteBytes8(w io.Writer, data []byte) error {
	if len(data) > 256 {
		panic(fmt.Sprintf("WriteBytes8: too long data (%v)", len(data)))
	}
	err := WriteByte(w, byte(len(data)))
	if err != nil {
		return err
	}
	if len(data) != 0 {
		_, err = w.Write(data)
	}
	return err
}

func ReadBytes16(r io.Reader) ([]byte, error) {
	var length uint16
	err := ReadUint16(r, &length)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return []byte{}, nil
	}
	ret := make([]byte, length)
	_, err = r.Read(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func WriteBytes16(w io.Writer, data []byte) error {
	if len(data) > math.MaxUint16 {
		panic(fmt.Sprintf("WriteBytes16: too long data (%v)", len(data)))
	}
	err := WriteUint16(w, uint16(len(data)))
	if err != nil {
		return err
	}
	if len(data) != 0 {
		_, err = w.Write(data)
	}
	return err
}

func ReadUint16(r io.Reader, pval *uint16) error {
	var tmp2 [2]byte
	_, err := r.Read(tmp2[:])
	if err != nil {
		return err
	}
	*pval = binary.LittleEndian.Uint16(tmp2[:])
	return nil
}

func WriteUint16(w io.Writer, val uint16) error {
	_, err := w.Write(Uint16To2Bytes(val))
	return err
}

func Uint16To2Bytes(val uint16) []byte {
	var tmp2 [2]byte
	binary.LittleEndian.PutUint16(tmp2[:], val)
	return tmp2[:]
}

func Uint16From2Bytes(b []byte) (uint16, error) {
	if len(b) != 2 {
		return 0, errors.New("len(b) != 2")
	}
	return binary.LittleEndian.Uint16(b), nil
}

func ReadByte(r io.Reader) (byte, error) {
	var b [1]byte
	_, err := r.Read(b[:])
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func WriteByte(w io.Writer, val byte) error {
	b := []byte{val}
	_, err := w.Write(b)
	return err
}

func ReadBytes32(r io.Reader) ([]byte, error) {
	var length uint32
	err := ReadUint32(r, &length)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return []byte{}, nil
	}
	ret := make([]byte, length)
	_, err = r.Read(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func WriteBytes32(w io.Writer, data []byte) error {
	if len(data) > math.MaxUint32 {
		panic(fmt.Sprintf("WriteBytes32: too long data (%v)", len(data)))
	}
	err := WriteUint32(w, uint32(len(data)))
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func Uint32To4Bytes(val uint32) []byte {
	var tmp4 [4]byte
	binary.LittleEndian.PutUint32(tmp4[:], val)
	return tmp4[:]
}

func Uint32From4Bytes(b []byte) (uint32, error) {
	if len(b) != 4 {
		return 0, errors.New("len(b) != 4")
	}
	return binary.LittleEndian.Uint32(b), nil
}

func MustUint32From4Bytes(b []byte) uint32 {
	ret, err := Uint32From4Bytes(b)
	if err != nil {
		panic(err)
	}
	return ret
}

func ReadUint32(r io.Reader, pval *uint32) error {
	var tmp4 [4]byte
	_, err := r.Read(tmp4[:])
	if err != nil {
		return err
	}
	*pval = MustUint32From4Bytes(tmp4[:])
	return nil
}

func WriteUint32(w io.Writer, val uint32) error {
	_, err := w.Write(Uint32To4Bytes(val))
	return err
}

func Blake2b160(data []byte) (ret [20]byte) {
	hash, _ := blake2b.New(20, nil)
	if _, err := hash.Write(data); err != nil {
		panic(err)
	}
	copy(ret[:], hash.Sum(nil))
	return
}
