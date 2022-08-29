package trie_mimc

import (
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
)

// api frontend.API, h mimc.MiMC, data frontend.Variable
// Proof mimc 32 byte model-specific proof of inclusion
type FrontendProof struct {
	PathArity frontend.Variable      //trie.PathArity
	HashSize  frontend.Variable      //HashSize
	Key       []frontend.Variable    //[]byte
	Path      []FrontendProofElement //[]*ProofElement
}

type FrontendProofElement struct {
	// PathFragment []frontend.Variable //[]byte
	Children   []frontend.Variable //[]byte only for binary tree
	Terminal   []frontend.Variable //[]byte
	ChildIndex frontend.Variable   //int
}

// Error Types
const (
	WrongCasting         int = 1
	UnexpectedCommitment int = 2
)

// Return the result of right shift by 1, input size = 32
// Input: {42, 194, 10, 96, 133, 113, 88, 31, 86, 136, 60, 65, 11, 106, 226, 218, 169, 220, 186, 36, 114, 230, 53, 147, 171, 202, 12, 106, 45, 89, 231, 132}
// Output: {42, 194, 10, 96, 133, 113, 88, 31, 86, 136, 60, 65, 11, 106, 226, 218, 169, 220, 186, 36, 114, 230, 53, 147, 171, 202, 12, 106, 45, 89, 231}
func rightShift1(api frontend.API, input frontend.Variable) frontend.Variable {
	var lsb8 frontend.Variable = 0
	var multiplier frontend.Variable = 1
	inputBinary := api.ToBinary(input)
	for i := 0; i < 8; i++ {
		lsb8 = api.Add(lsb8, api.Mul(inputBinary[i], multiplier))
		multiplier = api.Mul(multiplier, 2)
	}
	return api.DivUnchecked(api.Sub(input, lsb8), 256)
}

func leastNBytes(api frontend.API, input frontend.Variable, N int) frontend.Variable {
	var lsb frontend.Variable = 0
	var multiplier frontend.Variable = 1
	inputBinary := api.ToBinary(input)
	for i := 0; i < 8*N; i++ {
		lsb = api.Add(lsb, api.Mul(inputBinary[i], multiplier))
		multiplier = api.Mul(multiplier, 2)
	}
	return lsb
}

// Right shift N Bytes
func rightShiftNBytes(api frontend.API, input frontend.Variable, N int) frontend.Variable {
	var lsb frontend.Variable = 0
	var multiplier frontend.Variable = 1
	inputBinary := api.ToBinary(input)
	for i := 0; i < N*8; i++ {
		lsb = api.Add(lsb, api.Mul(inputBinary[i], multiplier))
		multiplier = api.Mul(multiplier, 2)
	}

	var divider frontend.Variable = 1
	for i := 0; i < N; i++ {
		divider = api.Mul(divider, 256)
	}
	return api.DivUnchecked(api.Sub(input, lsb), divider)
}

func hashVectors(api frontend.API, hFunc mimc.MiMC, hashes ...frontend.Variable) frontend.Variable {
	hFunc.Write(hashes[0])
	hFunc.Write(rightShift1(api, hashes[1]))
	for n, h := range hashes[2:] {
		// hs = append(hs, api.Add(NBytesLeftShift(api, hashes[n+1], n+1), rightShiftNBytes(api, h, n+2)))
		hFunc.Write(api.Add(NBytesLeftShift(api, hashes[n+1], n+1), rightShiftNBytes(api, h, n+2)))
	}
	hFunc.Write(leftShift1Byte(api, leastNBytes(api, hashes[len(hashes)-1], len(hashes)-1)))
	return hFunc.Sum()
}

// Shift lsb N Bytes to left, with 0 tails
func NBytesLeftShift(api frontend.API, input frontend.Variable, N int) frontend.Variable {
	var lsb frontend.Variable = 0
	var multiplier frontend.Variable = 1
	inputBinary := api.ToBinary(input)
	for i := 0; i < 8*N; i++ {
		lsb = api.Add(lsb, api.Mul(inputBinary[i], multiplier))
		multiplier = api.Mul(multiplier, 2)
	}
	multiplier = 1
	for i := 0; i < 32-N; i++ {
		multiplier = api.Mul(multiplier, 256)
	}
	return api.Mul(lsb, multiplier)
}

func Lsb8LeftShift31(api frontend.API, input frontend.Variable) frontend.Variable {
	var lsb8 frontend.Variable = 0
	var multiplier frontend.Variable = 1
	inputBinary := api.ToBinary(input)
	for i := 0; i < 8; i++ {
		lsb8 = api.Add(lsb8, api.Mul(inputBinary[i], multiplier))
		multiplier = api.Mul(multiplier, 2)
	}
	return api.Mul(lsb8, []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
}

// Return the result of right shift by 1, input size = 32
// Input: {42, 194, 10, 96, 133, 113, 88, 31, 86, 136, 60, 65, 11, 106, 226, 218, 169, 220, 186, 36, 114, 230, 53, 147, 171, 202, 12, 106, 45, 89, 231}
// Output: {42, 194, 10, 96, 133, 113, 88, 31, 86, 136, 60, 65, 11, 106, 226, 218, 169, 220, 186, 36, 114, 230, 53, 147, 171, 202, 12, 106, 45, 89, 231, 0}
func leftShift1Byte(api frontend.API, input frontend.Variable) frontend.Variable {
	return api.Mul(input, 256)
}

// Validate check the proof against the provided root commitments
// proofs is the hash from the leaf to the root
// paths indicate the children location through the path from the leaf to the root
func Validate(api frontend.API, hFunc mimc.MiMC, root frontend.Variable, ps0, ps1, ps2, ps3 []frontend.Variable, paths []frontend.Variable) {
	h := hashVectors(api, hFunc, ps0[0], ps1[0], ps2[0], ps3[0])
	for i := 1; i < len(ps0); i++ {
		s0 := api.Select(paths[i-1], ps0[i], h)
		s1 := api.Select(paths[i-1], h, ps1[i])
		tmp := hashVectors(api, hFunc, s0, s1, ps2[i], ps3[i])
		h = api.Select(api.Cmp(api.Add(ps0[i], ps1[i], ps2[i], ps3[i]), 0), tmp, h)
	}
	api.AssertIsEqual(h, root)
}
