package tests

import (
	"math/rand"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"golang.org/x/crypto/blake2b"
)

func BenchmarkHashing(b *testing.B) {
	var short [32]byte
	var medium [2000]byte
	var long [8000]byte
	rand.Read(short[:])
	rand.Read(medium[:])
	rand.Read(long[:])

	b.Run("short-blake2b", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			blake2b.Sum256(short[:])
		}
	})
	b.Run("medium-blake2b", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			blake2b.Sum256(medium[:])
		}
	})
	b.Run("long-blake2b", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			blake2b.Sum256(long[:])
		}
	})
	b.Run("short_mimc", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			mimcIt(short[:])
		}
	})
	b.Run("medium_mimc", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			mimcIt(medium[:])
		}
	})
	b.Run("long_mimc", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			mimcIt(long[:])
		}
	})
}

func mimcIt(data []byte) []byte {
	h := mimc.NewMiMC()
	h.Write(data)
	ret := h.Sum(nil)
	return ret[:]
}
