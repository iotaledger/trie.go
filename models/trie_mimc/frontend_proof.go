package trie_mimc

import (
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
)

// Return the result of right shift by 1 byte, input size = 32 bytes (fixed)
// Input: {42, 194, .. X other variables .., 231, 132} (Vector with length)
// Output: {42, 194, .. X other variables .., 231} (Vector with length-1)
func rightShift1Byte(api frontend.API, input frontend.Variable) frontend.Variable {
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
	if N >= 32 {
		return api.Add(input, 0)
	}
	for i := 0; i < 8*N; i++ {
		lsb = api.Add(lsb, api.Mul(inputBinary[i], multiplier))
		multiplier = api.Mul(multiplier, 2)
	}
	return lsb
}

// Right shift by N Bytes
// As DivUnchecked does not work with remainder we subtract N lsb, before dividing
func rightShiftNBytes(api frontend.API, input frontend.Variable, N int) frontend.Variable {
	var lsb frontend.Variable = 0
	var multiplier frontend.Variable = 1
	inputBinary := api.ToBinary(input)
	if N >= 32 {
		return 0
	}
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

// Hash the proof sets, including all children, terminal, and path fragment.
func hashVectors(api frontend.API, hFunc mimc.MiMC,
	hashes ...frontend.Variable) frontend.Variable {

	hFunc.Write(hashes[0])
	hFunc.Write(rightShift1Byte(api, hashes[1]))
	for n, h := range hashes[2:] {
		hFunc.Write(api.Add(NBytesLeftShift(api, hashes[n+1], n+1),
			rightShiftNBytes(api, h, n+2)))
	}
	hFunc.Write(leftShift1Byte(api, leastNBytes(api, hashes[len(hashes)-1], len(hashes)-1)))
	return hFunc.Sum()
}

// Shift lsb N Bytes to the left, with tailing 0s.
func NBytesLeftShift(api frontend.API, input frontend.Variable, N int) frontend.Variable {
	var lsb frontend.Variable = 0
	var multiplier frontend.Variable = 1
	inputBinary := api.ToBinary(input)

	// Note: here we make sure all the calculations are within 32 bytes.
	if N >= 32 {
		return 0
	}
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

// Return the result of left shift by 1 byte, input size = 32 bytes (fixed)
// Note: The MSB should be 0, or overflow occurs (the overflow behavior is not simply ``mod'' by 2^256)
// Input: {0, 42, 194, .. X other variables .., 231}
// Output: {42, 194, , .. X other variables .., 231, 0}
func leftShift1Byte(api frontend.API, input frontend.Variable) frontend.Variable {
	return api.Mul(input, 256)
}

// Validate2 checks the proof against the provided root commitments in a binary trie
// ps0-3 are the proof sets. The proof sets consist of children, terminal, and path fragments along the path.
// For example, for binary trie, there are two children (ps0, ps1), terminal (ps2), and path fragment (ps3).
// We name them as proof sets according to the original gnark example by using a binary complete tree.
// paths indicate the children location through the path from the leaf to the root
func Validate2(api frontend.API, hFunc mimc.MiMC, root frontend.Variable,
	ps0, ps1, ps2, ps3 []frontend.Variable, paths []frontend.Variable) {
	h := hashVectors(api, hFunc, ps0[0], ps1[0], ps2[0], ps3[0])
	for i := 1; i < len(ps0); i++ {
		s0 := api.Select(paths[i-1], ps0[i], h)
		s1 := api.Select(paths[i-1], h, ps1[i])
		tmp := hashVectors(api, hFunc, s0, s1, ps2[i], ps3[i])
		// Note that all the proof set are 0 if there is no elements for that path step,
		// thus h from the previous step is carried forward.
		h = api.Select(api.Cmp(api.Add(ps0[i], ps1[i], ps2[i], ps3[i]), 0), tmp, h)
	}
	api.AssertIsEqual(h, root)
}

// Validate16 check the proof against the provided root commitments in a hexadecimal trie
// ps0-17 are the proof sets. The proof sets consist of children, terminal, and path fragments along the path.
// For example, for binary trie, there are two children (ps0-15), terminal (ps16), and path fragment (ps17).
// We name them as proof sets according to the original gnark example by using a binary complete tree.
// paths indicate the children location through the path from the leaf to the root
func Validate16(api frontend.API, hFunc mimc.MiMC, root frontend.Variable, ps0, ps1, ps2, ps3, ps4, ps5,
	ps6, ps7, ps8, ps9, ps10, ps11, ps12, ps13, ps14, ps15, ps16, ps17 []frontend.Variable,
	paths []frontend.Variable) {
	h := hashVectors(api, hFunc, ps0[0], ps1[0], ps2[0], ps3[0], ps4[0], ps5[0], ps6[0], ps7[0], ps8[0],
		ps9[0], ps10[0], ps11[0], ps12[0], ps13[0], ps14[0], ps15[0], ps16[0], ps17[0])
	for i := 1; i < len(ps0); i++ {
		// Note: cannot use api.IsZero(api.Cmp(...))
		s0 := api.Select(api.IsZero(api.Sub(paths[i-1], 0)), h, ps0[i])
		s1 := api.Select(api.IsZero(api.Sub(paths[i-1], 1)), h, ps1[i])
		s2 := api.Select(api.IsZero(api.Sub(paths[i-1], 2)), h, ps2[i])
		s3 := api.Select(api.IsZero(api.Sub(paths[i-1], 3)), h, ps3[i])
		s4 := api.Select(api.IsZero(api.Sub(paths[i-1], 4)), h, ps4[i])
		s5 := api.Select(api.IsZero(api.Sub(paths[i-1], 5)), h, ps5[i])
		s6 := api.Select(api.IsZero(api.Sub(paths[i-1], 6)), h, ps6[i])
		s7 := api.Select(api.IsZero(api.Sub(paths[i-1], 7)), h, ps7[i])
		s8 := api.Select(api.IsZero(api.Sub(paths[i-1], 8)), h, ps8[i])
		s9 := api.Select(api.IsZero(api.Sub(paths[i-1], 9)), h, ps9[i])
		s10 := api.Select(api.IsZero(api.Sub(paths[i-1], 10)), h, ps10[i])
		s11 := api.Select(api.IsZero(api.Sub(paths[i-1], 11)), h, ps11[i])
		s12 := api.Select(api.IsZero(api.Sub(paths[i-1], 12)), h, ps12[i])
		s13 := api.Select(api.IsZero(api.Sub(paths[i-1], 13)), h, ps13[i])
		s14 := api.Select(api.IsZero(api.Sub(paths[i-1], 14)), h, ps14[i])
		s15 := api.Select(api.IsZero(api.Sub(paths[i-1], 15)), h, ps15[i])
		tmp := hashVectors(api, hFunc, s0, s1, s2, s3, s4, s5, s6, s7, s8,
			s9, s10, s11, s12, s13, s14, s15, ps16[i], ps17[i])
		// Note that all the proof set are 0 if there is no elements for that path step,
		// thus h from the previous step is carried forward.
		h = api.Select(api.Cmp(api.Add(ps0[i], ps1[i], ps2[i], ps3[i], ps4[i], ps5[i], ps6[i], ps7[i], ps8[i],
			ps9[i], ps10[i], ps11[i], ps12[i], ps13[i], ps14[i], ps15[i], ps16[i], ps17[i]), 0), tmp, h)
	}
	api.AssertIsEqual(h, root)
}

// Validate256 check the proof against the provided root commitments in a 256 trie
// ps0-257 are the proof sets. The proof sets consist of children, terminal, and path fragments along the path.
// For example, for binary trie, there are two children (ps0-255), terminal (ps256), and path fragment (ps257).
// We name them as proof sets according to the original gnark example by using a binary complete tree.
// paths indicate the children location through the path from the leaf to the root
func Validate256(api frontend.API, hFunc mimc.MiMC, root frontend.Variable,
	ps0, ps1, ps2, ps3, ps4, ps5, ps6, ps7, ps8, ps9, ps10, ps11, ps12, ps13, ps14, ps15,
	ps16, ps17, ps18, ps19, ps20, ps21, ps22, ps23, ps24, ps25, ps26, ps27, ps28, ps29, ps30, ps31,
	ps32, ps33, ps34, ps35, ps36, ps37, ps38, ps39, ps40, ps41, ps42, ps43, ps44, ps45, ps46, ps47,
	ps48, ps49, ps50, ps51, ps52, ps53, ps54, ps55, ps56, ps57, ps58, ps59, ps60, ps61, ps62, ps63,
	ps64, ps65, ps66, ps67, ps68, ps69, ps70, ps71, ps72, ps73, ps74, ps75, ps76, ps77, ps78, ps79,
	ps80, ps81, ps82, ps83, ps84, ps85, ps86, ps87, ps88, ps89, ps90, ps91, ps92, ps93, ps94, ps95,
	ps96, ps97, ps98, ps99, ps100, ps101, ps102, ps103, ps104, ps105, ps106, ps107, ps108, ps109, ps110, ps111,
	ps112, ps113, ps114, ps115, ps116, ps117, ps118, ps119, ps120, ps121, ps122, ps123, ps124, ps125, ps126, ps127,
	ps128, ps129, ps130, ps131, ps132, ps133, ps134, ps135, ps136, ps137, ps138, ps139, ps140, ps141, ps142, ps143,
	ps144, ps145, ps146, ps147, ps148, ps149, ps150, ps151, ps152, ps153, ps154, ps155, ps156, ps157, ps158, ps159,
	ps160, ps161, ps162, ps163, ps164, ps165, ps166, ps167, ps168, ps169, ps170, ps171, ps172, ps173, ps174, ps175,
	ps176, ps177, ps178, ps179, ps180, ps181, ps182, ps183, ps184, ps185, ps186, ps187, ps188, ps189, ps190, ps191,
	ps192, ps193, ps194, ps195, ps196, ps197, ps198, ps199, ps200, ps201, ps202, ps203, ps204, ps205, ps206, ps207,
	ps208, ps209, ps210, ps211, ps212, ps213, ps214, ps215, ps216, ps217, ps218, ps219, ps220, ps221, ps222, ps223,
	ps224, ps225, ps226, ps227, ps228, ps229, ps230, ps231, ps232, ps233, ps234, ps235, ps236, ps237, ps238, ps239,
	ps240, ps241, ps242, ps243, ps244, ps245, ps246, ps247, ps248, ps249, ps250, ps251, ps252, ps253, ps254, ps255,
	ps256, ps257 []frontend.Variable, paths []frontend.Variable) {
	h := hashVectors(api, hFunc, ps0[0], ps1[0], ps2[0], ps3[0], ps4[0], ps5[0], ps6[0], ps7[0], ps8[0], ps9[0], ps10[0], ps11[0], ps12[0], ps13[0], ps14[0], ps15[0],
		ps16[0], ps17[0], ps18[0], ps19[0], ps20[0], ps21[0], ps22[0], ps23[0], ps24[0], ps25[0], ps26[0], ps27[0], ps28[0], ps29[0], ps30[0], ps31[0],
		ps32[0], ps33[0], ps34[0], ps35[0], ps36[0], ps37[0], ps38[0], ps39[0], ps40[0], ps41[0], ps42[0], ps43[0], ps44[0], ps45[0], ps46[0], ps47[0],
		ps48[0], ps49[0], ps50[0], ps51[0], ps52[0], ps53[0], ps54[0], ps55[0], ps56[0], ps57[0], ps58[0], ps59[0], ps60[0], ps61[0], ps62[0], ps63[0],
		ps64[0], ps65[0], ps66[0], ps67[0], ps68[0], ps69[0], ps70[0], ps71[0], ps72[0], ps73[0], ps74[0], ps75[0], ps76[0], ps77[0], ps78[0], ps79[0],
		ps80[0], ps81[0], ps82[0], ps83[0], ps84[0], ps85[0], ps86[0], ps87[0], ps88[0], ps89[0], ps90[0], ps91[0], ps92[0], ps93[0], ps94[0], ps95[0],
		ps96[0], ps97[0], ps98[0], ps99[0], ps100[0], ps101[0], ps102[0], ps103[0], ps104[0], ps105[0], ps106[0], ps107[0], ps108[0], ps109[0], ps110[0], ps111[0],
		ps112[0], ps113[0], ps114[0], ps115[0], ps116[0], ps117[0], ps118[0], ps119[0], ps120[0], ps121[0], ps122[0], ps123[0], ps124[0], ps125[0], ps126[0], ps127[0],
		ps128[0], ps129[0], ps130[0], ps131[0], ps132[0], ps133[0], ps134[0], ps135[0], ps136[0], ps137[0], ps138[0], ps139[0], ps140[0], ps141[0], ps142[0], ps143[0],
		ps144[0], ps145[0], ps146[0], ps147[0], ps148[0], ps149[0], ps150[0], ps151[0], ps152[0], ps153[0], ps154[0], ps155[0], ps156[0], ps157[0], ps158[0], ps159[0],
		ps160[0], ps161[0], ps162[0], ps163[0], ps164[0], ps165[0], ps166[0], ps167[0], ps168[0], ps169[0], ps170[0], ps171[0], ps172[0], ps173[0], ps174[0], ps175[0],
		ps176[0], ps177[0], ps178[0], ps179[0], ps180[0], ps181[0], ps182[0], ps183[0], ps184[0], ps185[0], ps186[0], ps187[0], ps188[0], ps189[0], ps190[0], ps191[0],
		ps192[0], ps193[0], ps194[0], ps195[0], ps196[0], ps197[0], ps198[0], ps199[0], ps200[0], ps201[0], ps202[0], ps203[0], ps204[0], ps205[0], ps206[0], ps207[0],
		ps208[0], ps209[0], ps210[0], ps211[0], ps212[0], ps213[0], ps214[0], ps215[0], ps216[0], ps217[0], ps218[0], ps219[0], ps220[0], ps221[0], ps222[0], ps223[0],
		ps224[0], ps225[0], ps226[0], ps227[0], ps228[0], ps229[0], ps230[0], ps231[0], ps232[0], ps233[0], ps234[0], ps235[0], ps236[0], ps237[0], ps238[0], ps239[0],
		ps240[0], ps241[0], ps242[0], ps243[0], ps244[0], ps245[0], ps246[0], ps247[0], ps248[0], ps249[0], ps250[0], ps251[0], ps252[0], ps253[0], ps254[0], ps255[0],
		ps256[0], ps257[0])
	for i := 1; i < len(ps0); i++ {
		// Note: cannot use api.IsZero(api.Cmp(...))
		s0 := api.Select(api.IsZero(api.Sub(paths[i-1], 0)), h, ps0[i])
		s1 := api.Select(api.IsZero(api.Sub(paths[i-1], 1)), h, ps1[i])
		s2 := api.Select(api.IsZero(api.Sub(paths[i-1], 2)), h, ps2[i])
		s3 := api.Select(api.IsZero(api.Sub(paths[i-1], 3)), h, ps3[i])
		s4 := api.Select(api.IsZero(api.Sub(paths[i-1], 4)), h, ps4[i])
		s5 := api.Select(api.IsZero(api.Sub(paths[i-1], 5)), h, ps5[i])
		s6 := api.Select(api.IsZero(api.Sub(paths[i-1], 6)), h, ps6[i])
		s7 := api.Select(api.IsZero(api.Sub(paths[i-1], 7)), h, ps7[i])
		s8 := api.Select(api.IsZero(api.Sub(paths[i-1], 8)), h, ps8[i])
		s9 := api.Select(api.IsZero(api.Sub(paths[i-1], 9)), h, ps9[i])
		s10 := api.Select(api.IsZero(api.Sub(paths[i-1], 10)), h, ps10[i])
		s11 := api.Select(api.IsZero(api.Sub(paths[i-1], 11)), h, ps11[i])
		s12 := api.Select(api.IsZero(api.Sub(paths[i-1], 12)), h, ps12[i])
		s13 := api.Select(api.IsZero(api.Sub(paths[i-1], 13)), h, ps13[i])
		s14 := api.Select(api.IsZero(api.Sub(paths[i-1], 14)), h, ps14[i])
		s15 := api.Select(api.IsZero(api.Sub(paths[i-1], 15)), h, ps15[i])
		s16 := api.Select(api.IsZero(api.Sub(paths[i-1], 16)), h, ps16[i])
		s17 := api.Select(api.IsZero(api.Sub(paths[i-1], 17)), h, ps17[i])
		s18 := api.Select(api.IsZero(api.Sub(paths[i-1], 18)), h, ps18[i])
		s19 := api.Select(api.IsZero(api.Sub(paths[i-1], 19)), h, ps19[i])
		s20 := api.Select(api.IsZero(api.Sub(paths[i-1], 20)), h, ps20[i])
		s21 := api.Select(api.IsZero(api.Sub(paths[i-1], 21)), h, ps21[i])
		s22 := api.Select(api.IsZero(api.Sub(paths[i-1], 22)), h, ps22[i])
		s23 := api.Select(api.IsZero(api.Sub(paths[i-1], 23)), h, ps23[i])
		s24 := api.Select(api.IsZero(api.Sub(paths[i-1], 24)), h, ps24[i])
		s25 := api.Select(api.IsZero(api.Sub(paths[i-1], 25)), h, ps25[i])
		s26 := api.Select(api.IsZero(api.Sub(paths[i-1], 26)), h, ps26[i])
		s27 := api.Select(api.IsZero(api.Sub(paths[i-1], 27)), h, ps27[i])
		s28 := api.Select(api.IsZero(api.Sub(paths[i-1], 28)), h, ps28[i])
		s29 := api.Select(api.IsZero(api.Sub(paths[i-1], 29)), h, ps29[i])
		s30 := api.Select(api.IsZero(api.Sub(paths[i-1], 30)), h, ps30[i])
		s31 := api.Select(api.IsZero(api.Sub(paths[i-1], 31)), h, ps31[i])
		s32 := api.Select(api.IsZero(api.Sub(paths[i-1], 32)), h, ps32[i])
		s33 := api.Select(api.IsZero(api.Sub(paths[i-1], 33)), h, ps33[i])
		s34 := api.Select(api.IsZero(api.Sub(paths[i-1], 34)), h, ps34[i])
		s35 := api.Select(api.IsZero(api.Sub(paths[i-1], 35)), h, ps35[i])
		s36 := api.Select(api.IsZero(api.Sub(paths[i-1], 36)), h, ps36[i])
		s37 := api.Select(api.IsZero(api.Sub(paths[i-1], 37)), h, ps37[i])
		s38 := api.Select(api.IsZero(api.Sub(paths[i-1], 38)), h, ps38[i])
		s39 := api.Select(api.IsZero(api.Sub(paths[i-1], 39)), h, ps39[i])
		s40 := api.Select(api.IsZero(api.Sub(paths[i-1], 40)), h, ps40[i])
		s41 := api.Select(api.IsZero(api.Sub(paths[i-1], 41)), h, ps41[i])
		s42 := api.Select(api.IsZero(api.Sub(paths[i-1], 42)), h, ps42[i])
		s43 := api.Select(api.IsZero(api.Sub(paths[i-1], 43)), h, ps43[i])
		s44 := api.Select(api.IsZero(api.Sub(paths[i-1], 44)), h, ps44[i])
		s45 := api.Select(api.IsZero(api.Sub(paths[i-1], 45)), h, ps45[i])
		s46 := api.Select(api.IsZero(api.Sub(paths[i-1], 46)), h, ps46[i])
		s47 := api.Select(api.IsZero(api.Sub(paths[i-1], 47)), h, ps47[i])
		s48 := api.Select(api.IsZero(api.Sub(paths[i-1], 48)), h, ps48[i])
		s49 := api.Select(api.IsZero(api.Sub(paths[i-1], 49)), h, ps49[i])
		s50 := api.Select(api.IsZero(api.Sub(paths[i-1], 50)), h, ps50[i])
		s51 := api.Select(api.IsZero(api.Sub(paths[i-1], 51)), h, ps51[i])
		s52 := api.Select(api.IsZero(api.Sub(paths[i-1], 52)), h, ps52[i])
		s53 := api.Select(api.IsZero(api.Sub(paths[i-1], 53)), h, ps53[i])
		s54 := api.Select(api.IsZero(api.Sub(paths[i-1], 54)), h, ps54[i])
		s55 := api.Select(api.IsZero(api.Sub(paths[i-1], 55)), h, ps55[i])
		s56 := api.Select(api.IsZero(api.Sub(paths[i-1], 56)), h, ps56[i])
		s57 := api.Select(api.IsZero(api.Sub(paths[i-1], 57)), h, ps57[i])
		s58 := api.Select(api.IsZero(api.Sub(paths[i-1], 58)), h, ps58[i])
		s59 := api.Select(api.IsZero(api.Sub(paths[i-1], 59)), h, ps59[i])
		s60 := api.Select(api.IsZero(api.Sub(paths[i-1], 60)), h, ps60[i])
		s61 := api.Select(api.IsZero(api.Sub(paths[i-1], 61)), h, ps61[i])
		s62 := api.Select(api.IsZero(api.Sub(paths[i-1], 62)), h, ps62[i])
		s63 := api.Select(api.IsZero(api.Sub(paths[i-1], 63)), h, ps63[i])
		s64 := api.Select(api.IsZero(api.Sub(paths[i-1], 64)), h, ps64[i])
		s65 := api.Select(api.IsZero(api.Sub(paths[i-1], 65)), h, ps65[i])
		s66 := api.Select(api.IsZero(api.Sub(paths[i-1], 66)), h, ps66[i])
		s67 := api.Select(api.IsZero(api.Sub(paths[i-1], 67)), h, ps67[i])
		s68 := api.Select(api.IsZero(api.Sub(paths[i-1], 68)), h, ps68[i])
		s69 := api.Select(api.IsZero(api.Sub(paths[i-1], 69)), h, ps69[i])
		s70 := api.Select(api.IsZero(api.Sub(paths[i-1], 70)), h, ps70[i])
		s71 := api.Select(api.IsZero(api.Sub(paths[i-1], 71)), h, ps71[i])
		s72 := api.Select(api.IsZero(api.Sub(paths[i-1], 72)), h, ps72[i])
		s73 := api.Select(api.IsZero(api.Sub(paths[i-1], 73)), h, ps73[i])
		s74 := api.Select(api.IsZero(api.Sub(paths[i-1], 74)), h, ps74[i])
		s75 := api.Select(api.IsZero(api.Sub(paths[i-1], 75)), h, ps75[i])
		s76 := api.Select(api.IsZero(api.Sub(paths[i-1], 76)), h, ps76[i])
		s77 := api.Select(api.IsZero(api.Sub(paths[i-1], 77)), h, ps77[i])
		s78 := api.Select(api.IsZero(api.Sub(paths[i-1], 78)), h, ps78[i])
		s79 := api.Select(api.IsZero(api.Sub(paths[i-1], 79)), h, ps79[i])
		s80 := api.Select(api.IsZero(api.Sub(paths[i-1], 80)), h, ps80[i])
		s81 := api.Select(api.IsZero(api.Sub(paths[i-1], 81)), h, ps81[i])
		s82 := api.Select(api.IsZero(api.Sub(paths[i-1], 82)), h, ps82[i])
		s83 := api.Select(api.IsZero(api.Sub(paths[i-1], 83)), h, ps83[i])
		s84 := api.Select(api.IsZero(api.Sub(paths[i-1], 84)), h, ps84[i])
		s85 := api.Select(api.IsZero(api.Sub(paths[i-1], 85)), h, ps85[i])
		s86 := api.Select(api.IsZero(api.Sub(paths[i-1], 86)), h, ps86[i])
		s87 := api.Select(api.IsZero(api.Sub(paths[i-1], 87)), h, ps87[i])
		s88 := api.Select(api.IsZero(api.Sub(paths[i-1], 88)), h, ps88[i])
		s89 := api.Select(api.IsZero(api.Sub(paths[i-1], 89)), h, ps89[i])
		s90 := api.Select(api.IsZero(api.Sub(paths[i-1], 90)), h, ps90[i])
		s91 := api.Select(api.IsZero(api.Sub(paths[i-1], 91)), h, ps91[i])
		s92 := api.Select(api.IsZero(api.Sub(paths[i-1], 92)), h, ps92[i])
		s93 := api.Select(api.IsZero(api.Sub(paths[i-1], 93)), h, ps93[i])
		s94 := api.Select(api.IsZero(api.Sub(paths[i-1], 94)), h, ps94[i])
		s95 := api.Select(api.IsZero(api.Sub(paths[i-1], 95)), h, ps95[i])
		s96 := api.Select(api.IsZero(api.Sub(paths[i-1], 96)), h, ps96[i])
		s97 := api.Select(api.IsZero(api.Sub(paths[i-1], 97)), h, ps97[i])
		s98 := api.Select(api.IsZero(api.Sub(paths[i-1], 98)), h, ps98[i])
		s99 := api.Select(api.IsZero(api.Sub(paths[i-1], 99)), h, ps99[i])
		s100 := api.Select(api.IsZero(api.Sub(paths[i-1], 100)), h, ps100[i])
		s101 := api.Select(api.IsZero(api.Sub(paths[i-1], 101)), h, ps101[i])
		s102 := api.Select(api.IsZero(api.Sub(paths[i-1], 102)), h, ps102[i])
		s103 := api.Select(api.IsZero(api.Sub(paths[i-1], 103)), h, ps103[i])
		s104 := api.Select(api.IsZero(api.Sub(paths[i-1], 104)), h, ps104[i])
		s105 := api.Select(api.IsZero(api.Sub(paths[i-1], 105)), h, ps105[i])
		s106 := api.Select(api.IsZero(api.Sub(paths[i-1], 106)), h, ps106[i])
		s107 := api.Select(api.IsZero(api.Sub(paths[i-1], 107)), h, ps107[i])
		s108 := api.Select(api.IsZero(api.Sub(paths[i-1], 108)), h, ps108[i])
		s109 := api.Select(api.IsZero(api.Sub(paths[i-1], 109)), h, ps109[i])
		s110 := api.Select(api.IsZero(api.Sub(paths[i-1], 110)), h, ps110[i])
		s111 := api.Select(api.IsZero(api.Sub(paths[i-1], 111)), h, ps111[i])
		s112 := api.Select(api.IsZero(api.Sub(paths[i-1], 112)), h, ps112[i])
		s113 := api.Select(api.IsZero(api.Sub(paths[i-1], 113)), h, ps113[i])
		s114 := api.Select(api.IsZero(api.Sub(paths[i-1], 114)), h, ps114[i])
		s115 := api.Select(api.IsZero(api.Sub(paths[i-1], 115)), h, ps115[i])
		s116 := api.Select(api.IsZero(api.Sub(paths[i-1], 116)), h, ps116[i])
		s117 := api.Select(api.IsZero(api.Sub(paths[i-1], 117)), h, ps117[i])
		s118 := api.Select(api.IsZero(api.Sub(paths[i-1], 118)), h, ps118[i])
		s119 := api.Select(api.IsZero(api.Sub(paths[i-1], 119)), h, ps119[i])
		s120 := api.Select(api.IsZero(api.Sub(paths[i-1], 120)), h, ps120[i])
		s121 := api.Select(api.IsZero(api.Sub(paths[i-1], 121)), h, ps121[i])
		s122 := api.Select(api.IsZero(api.Sub(paths[i-1], 122)), h, ps122[i])
		s123 := api.Select(api.IsZero(api.Sub(paths[i-1], 123)), h, ps123[i])
		s124 := api.Select(api.IsZero(api.Sub(paths[i-1], 124)), h, ps124[i])
		s125 := api.Select(api.IsZero(api.Sub(paths[i-1], 125)), h, ps125[i])
		s126 := api.Select(api.IsZero(api.Sub(paths[i-1], 126)), h, ps126[i])
		s127 := api.Select(api.IsZero(api.Sub(paths[i-1], 127)), h, ps127[i])
		s128 := api.Select(api.IsZero(api.Sub(paths[i-1], 128)), h, ps128[i])
		s129 := api.Select(api.IsZero(api.Sub(paths[i-1], 129)), h, ps129[i])
		s130 := api.Select(api.IsZero(api.Sub(paths[i-1], 130)), h, ps130[i])
		s131 := api.Select(api.IsZero(api.Sub(paths[i-1], 131)), h, ps131[i])
		s132 := api.Select(api.IsZero(api.Sub(paths[i-1], 132)), h, ps132[i])
		s133 := api.Select(api.IsZero(api.Sub(paths[i-1], 133)), h, ps133[i])
		s134 := api.Select(api.IsZero(api.Sub(paths[i-1], 134)), h, ps134[i])
		s135 := api.Select(api.IsZero(api.Sub(paths[i-1], 135)), h, ps135[i])
		s136 := api.Select(api.IsZero(api.Sub(paths[i-1], 136)), h, ps136[i])
		s137 := api.Select(api.IsZero(api.Sub(paths[i-1], 137)), h, ps137[i])
		s138 := api.Select(api.IsZero(api.Sub(paths[i-1], 138)), h, ps138[i])
		s139 := api.Select(api.IsZero(api.Sub(paths[i-1], 139)), h, ps139[i])
		s140 := api.Select(api.IsZero(api.Sub(paths[i-1], 140)), h, ps140[i])
		s141 := api.Select(api.IsZero(api.Sub(paths[i-1], 141)), h, ps141[i])
		s142 := api.Select(api.IsZero(api.Sub(paths[i-1], 142)), h, ps142[i])
		s143 := api.Select(api.IsZero(api.Sub(paths[i-1], 143)), h, ps143[i])
		s144 := api.Select(api.IsZero(api.Sub(paths[i-1], 144)), h, ps144[i])
		s145 := api.Select(api.IsZero(api.Sub(paths[i-1], 145)), h, ps145[i])
		s146 := api.Select(api.IsZero(api.Sub(paths[i-1], 146)), h, ps146[i])
		s147 := api.Select(api.IsZero(api.Sub(paths[i-1], 147)), h, ps147[i])
		s148 := api.Select(api.IsZero(api.Sub(paths[i-1], 148)), h, ps148[i])
		s149 := api.Select(api.IsZero(api.Sub(paths[i-1], 149)), h, ps149[i])
		s150 := api.Select(api.IsZero(api.Sub(paths[i-1], 150)), h, ps150[i])
		s151 := api.Select(api.IsZero(api.Sub(paths[i-1], 151)), h, ps151[i])
		s152 := api.Select(api.IsZero(api.Sub(paths[i-1], 152)), h, ps152[i])
		s153 := api.Select(api.IsZero(api.Sub(paths[i-1], 153)), h, ps153[i])
		s154 := api.Select(api.IsZero(api.Sub(paths[i-1], 154)), h, ps154[i])
		s155 := api.Select(api.IsZero(api.Sub(paths[i-1], 155)), h, ps155[i])
		s156 := api.Select(api.IsZero(api.Sub(paths[i-1], 156)), h, ps156[i])
		s157 := api.Select(api.IsZero(api.Sub(paths[i-1], 157)), h, ps157[i])
		s158 := api.Select(api.IsZero(api.Sub(paths[i-1], 158)), h, ps158[i])
		s159 := api.Select(api.IsZero(api.Sub(paths[i-1], 159)), h, ps159[i])
		s160 := api.Select(api.IsZero(api.Sub(paths[i-1], 160)), h, ps160[i])
		s161 := api.Select(api.IsZero(api.Sub(paths[i-1], 161)), h, ps161[i])
		s162 := api.Select(api.IsZero(api.Sub(paths[i-1], 162)), h, ps162[i])
		s163 := api.Select(api.IsZero(api.Sub(paths[i-1], 163)), h, ps163[i])
		s164 := api.Select(api.IsZero(api.Sub(paths[i-1], 164)), h, ps164[i])
		s165 := api.Select(api.IsZero(api.Sub(paths[i-1], 165)), h, ps165[i])
		s166 := api.Select(api.IsZero(api.Sub(paths[i-1], 166)), h, ps166[i])
		s167 := api.Select(api.IsZero(api.Sub(paths[i-1], 167)), h, ps167[i])
		s168 := api.Select(api.IsZero(api.Sub(paths[i-1], 168)), h, ps168[i])
		s169 := api.Select(api.IsZero(api.Sub(paths[i-1], 169)), h, ps169[i])
		s170 := api.Select(api.IsZero(api.Sub(paths[i-1], 170)), h, ps170[i])
		s171 := api.Select(api.IsZero(api.Sub(paths[i-1], 171)), h, ps171[i])
		s172 := api.Select(api.IsZero(api.Sub(paths[i-1], 172)), h, ps172[i])
		s173 := api.Select(api.IsZero(api.Sub(paths[i-1], 173)), h, ps173[i])
		s174 := api.Select(api.IsZero(api.Sub(paths[i-1], 174)), h, ps174[i])
		s175 := api.Select(api.IsZero(api.Sub(paths[i-1], 175)), h, ps175[i])
		s176 := api.Select(api.IsZero(api.Sub(paths[i-1], 176)), h, ps176[i])
		s177 := api.Select(api.IsZero(api.Sub(paths[i-1], 177)), h, ps177[i])
		s178 := api.Select(api.IsZero(api.Sub(paths[i-1], 178)), h, ps178[i])
		s179 := api.Select(api.IsZero(api.Sub(paths[i-1], 179)), h, ps179[i])
		s180 := api.Select(api.IsZero(api.Sub(paths[i-1], 180)), h, ps180[i])
		s181 := api.Select(api.IsZero(api.Sub(paths[i-1], 181)), h, ps181[i])
		s182 := api.Select(api.IsZero(api.Sub(paths[i-1], 182)), h, ps182[i])
		s183 := api.Select(api.IsZero(api.Sub(paths[i-1], 183)), h, ps183[i])
		s184 := api.Select(api.IsZero(api.Sub(paths[i-1], 184)), h, ps184[i])
		s185 := api.Select(api.IsZero(api.Sub(paths[i-1], 185)), h, ps185[i])
		s186 := api.Select(api.IsZero(api.Sub(paths[i-1], 186)), h, ps186[i])
		s187 := api.Select(api.IsZero(api.Sub(paths[i-1], 187)), h, ps187[i])
		s188 := api.Select(api.IsZero(api.Sub(paths[i-1], 188)), h, ps188[i])
		s189 := api.Select(api.IsZero(api.Sub(paths[i-1], 189)), h, ps189[i])
		s190 := api.Select(api.IsZero(api.Sub(paths[i-1], 190)), h, ps190[i])
		s191 := api.Select(api.IsZero(api.Sub(paths[i-1], 191)), h, ps191[i])
		s192 := api.Select(api.IsZero(api.Sub(paths[i-1], 192)), h, ps192[i])
		s193 := api.Select(api.IsZero(api.Sub(paths[i-1], 193)), h, ps193[i])
		s194 := api.Select(api.IsZero(api.Sub(paths[i-1], 194)), h, ps194[i])
		s195 := api.Select(api.IsZero(api.Sub(paths[i-1], 195)), h, ps195[i])
		s196 := api.Select(api.IsZero(api.Sub(paths[i-1], 196)), h, ps196[i])
		s197 := api.Select(api.IsZero(api.Sub(paths[i-1], 197)), h, ps197[i])
		s198 := api.Select(api.IsZero(api.Sub(paths[i-1], 198)), h, ps198[i])
		s199 := api.Select(api.IsZero(api.Sub(paths[i-1], 199)), h, ps199[i])
		s200 := api.Select(api.IsZero(api.Sub(paths[i-1], 200)), h, ps200[i])
		s201 := api.Select(api.IsZero(api.Sub(paths[i-1], 201)), h, ps201[i])
		s202 := api.Select(api.IsZero(api.Sub(paths[i-1], 202)), h, ps202[i])
		s203 := api.Select(api.IsZero(api.Sub(paths[i-1], 203)), h, ps203[i])
		s204 := api.Select(api.IsZero(api.Sub(paths[i-1], 204)), h, ps204[i])
		s205 := api.Select(api.IsZero(api.Sub(paths[i-1], 205)), h, ps205[i])
		s206 := api.Select(api.IsZero(api.Sub(paths[i-1], 206)), h, ps206[i])
		s207 := api.Select(api.IsZero(api.Sub(paths[i-1], 207)), h, ps207[i])
		s208 := api.Select(api.IsZero(api.Sub(paths[i-1], 208)), h, ps208[i])
		s209 := api.Select(api.IsZero(api.Sub(paths[i-1], 209)), h, ps209[i])
		s210 := api.Select(api.IsZero(api.Sub(paths[i-1], 210)), h, ps210[i])
		s211 := api.Select(api.IsZero(api.Sub(paths[i-1], 211)), h, ps211[i])
		s212 := api.Select(api.IsZero(api.Sub(paths[i-1], 212)), h, ps212[i])
		s213 := api.Select(api.IsZero(api.Sub(paths[i-1], 213)), h, ps213[i])
		s214 := api.Select(api.IsZero(api.Sub(paths[i-1], 214)), h, ps214[i])
		s215 := api.Select(api.IsZero(api.Sub(paths[i-1], 215)), h, ps215[i])
		s216 := api.Select(api.IsZero(api.Sub(paths[i-1], 216)), h, ps216[i])
		s217 := api.Select(api.IsZero(api.Sub(paths[i-1], 217)), h, ps217[i])
		s218 := api.Select(api.IsZero(api.Sub(paths[i-1], 218)), h, ps218[i])
		s219 := api.Select(api.IsZero(api.Sub(paths[i-1], 219)), h, ps219[i])
		s220 := api.Select(api.IsZero(api.Sub(paths[i-1], 220)), h, ps220[i])
		s221 := api.Select(api.IsZero(api.Sub(paths[i-1], 221)), h, ps221[i])
		s222 := api.Select(api.IsZero(api.Sub(paths[i-1], 222)), h, ps222[i])
		s223 := api.Select(api.IsZero(api.Sub(paths[i-1], 223)), h, ps223[i])
		s224 := api.Select(api.IsZero(api.Sub(paths[i-1], 224)), h, ps224[i])
		s225 := api.Select(api.IsZero(api.Sub(paths[i-1], 225)), h, ps225[i])
		s226 := api.Select(api.IsZero(api.Sub(paths[i-1], 226)), h, ps226[i])
		s227 := api.Select(api.IsZero(api.Sub(paths[i-1], 227)), h, ps227[i])
		s228 := api.Select(api.IsZero(api.Sub(paths[i-1], 228)), h, ps228[i])
		s229 := api.Select(api.IsZero(api.Sub(paths[i-1], 229)), h, ps229[i])
		s230 := api.Select(api.IsZero(api.Sub(paths[i-1], 230)), h, ps230[i])
		s231 := api.Select(api.IsZero(api.Sub(paths[i-1], 231)), h, ps231[i])
		s232 := api.Select(api.IsZero(api.Sub(paths[i-1], 232)), h, ps232[i])
		s233 := api.Select(api.IsZero(api.Sub(paths[i-1], 233)), h, ps233[i])
		s234 := api.Select(api.IsZero(api.Sub(paths[i-1], 234)), h, ps234[i])
		s235 := api.Select(api.IsZero(api.Sub(paths[i-1], 235)), h, ps235[i])
		s236 := api.Select(api.IsZero(api.Sub(paths[i-1], 236)), h, ps236[i])
		s237 := api.Select(api.IsZero(api.Sub(paths[i-1], 237)), h, ps237[i])
		s238 := api.Select(api.IsZero(api.Sub(paths[i-1], 238)), h, ps238[i])
		s239 := api.Select(api.IsZero(api.Sub(paths[i-1], 239)), h, ps239[i])
		s240 := api.Select(api.IsZero(api.Sub(paths[i-1], 240)), h, ps240[i])
		s241 := api.Select(api.IsZero(api.Sub(paths[i-1], 241)), h, ps241[i])
		s242 := api.Select(api.IsZero(api.Sub(paths[i-1], 242)), h, ps242[i])
		s243 := api.Select(api.IsZero(api.Sub(paths[i-1], 243)), h, ps243[i])
		s244 := api.Select(api.IsZero(api.Sub(paths[i-1], 244)), h, ps244[i])
		s245 := api.Select(api.IsZero(api.Sub(paths[i-1], 245)), h, ps245[i])
		s246 := api.Select(api.IsZero(api.Sub(paths[i-1], 246)), h, ps246[i])
		s247 := api.Select(api.IsZero(api.Sub(paths[i-1], 247)), h, ps247[i])
		s248 := api.Select(api.IsZero(api.Sub(paths[i-1], 248)), h, ps248[i])
		s249 := api.Select(api.IsZero(api.Sub(paths[i-1], 249)), h, ps249[i])
		s250 := api.Select(api.IsZero(api.Sub(paths[i-1], 250)), h, ps250[i])
		s251 := api.Select(api.IsZero(api.Sub(paths[i-1], 251)), h, ps251[i])
		s252 := api.Select(api.IsZero(api.Sub(paths[i-1], 252)), h, ps252[i])
		s253 := api.Select(api.IsZero(api.Sub(paths[i-1], 253)), h, ps253[i])
		s254 := api.Select(api.IsZero(api.Sub(paths[i-1], 254)), h, ps254[i])
		s255 := api.Select(api.IsZero(api.Sub(paths[i-1], 255)), h, ps255[i])
		tmp := hashVectors(api, hFunc,
			s0, s1, s2, s3, s4, s5, s6, s7, s8, s9, s10, s11, s12, s13, s14, s15,
			s16, s17, s18, s19, s20, s21, s22, s23, s24, s25, s26, s27, s28, s29, s30, s31,
			s32, s33, s34, s35, s36, s37, s38, s39, s40, s41, s42, s43, s44, s45, s46, s47,
			s48, s49, s50, s51, s52, s53, s54, s55, s56, s57, s58, s59, s60, s61, s62, s63,
			s64, s65, s66, s67, s68, s69, s70, s71, s72, s73, s74, s75, s76, s77, s78, s79,
			s80, s81, s82, s83, s84, s85, s86, s87, s88, s89, s90, s91, s92, s93, s94, s95,
			s96, s97, s98, s99, s100, s101, s102, s103, s104, s105, s106, s107, s108, s109, s110, s111,
			s112, s113, s114, s115, s116, s117, s118, s119, s120, s121, s122, s123, s124, s125, s126, s127,
			s128, s129, s130, s131, s132, s133, s134, s135, s136, s137, s138, s139, s140, s141, s142, s143,
			s144, s145, s146, s147, s148, s149, s150, s151, s152, s153, s154, s155, s156, s157, s158, s159,
			s160, s161, s162, s163, s164, s165, s166, s167, s168, s169, s170, s171, s172, s173, s174, s175,
			s176, s177, s178, s179, s180, s181, s182, s183, s184, s185, s186, s187, s188, s189, s190, s191,
			s192, s193, s194, s195, s196, s197, s198, s199, s200, s201, s202, s203, s204, s205, s206, s207,
			s208, s209, s210, s211, s212, s213, s214, s215, s216, s217, s218, s219, s220, s221, s222, s223,
			s224, s225, s226, s227, s228, s229, s230, s231, s232, s233, s234, s235, s236, s237, s238, s239,
			s240, s241, s242, s243, s244, s245, s246, s247, s248, s249, s250, s251, s252, s253, s254, s255,
			ps256[i], ps257[i])
		// Note that all the proof set are 0 if there is no elements for that path step,
		// thus h from the previous step is carried forward.
		h = api.Select(api.Cmp(api.Add(ps0[i], ps1[i], ps2[i], ps3[i], ps4[i], ps5[i], ps6[i], ps7[i], ps8[i], ps9[i], ps10[i], ps11[i], ps12[i], ps13[i], ps14[i], ps15[i],
			ps16[i], ps17[i], ps18[i], ps19[i], ps20[i], ps21[i], ps22[i], ps23[i], ps24[i], ps25[i], ps26[i], ps27[i], ps28[i], ps29[i], ps30[i], ps31[i],
			ps32[i], ps33[i], ps34[i], ps35[i], ps36[i], ps37[i], ps38[i], ps39[i], ps40[i], ps41[i], ps42[i], ps43[i], ps44[i], ps45[i], ps46[i], ps47[i],
			ps48[i], ps49[i], ps50[i], ps51[i], ps52[i], ps53[i], ps54[i], ps55[i], ps56[i], ps57[i], ps58[i], ps59[i], ps60[i], ps61[i], ps62[i], ps63[i],
			ps64[i], ps65[i], ps66[i], ps67[i], ps68[i], ps69[i], ps70[i], ps71[i], ps72[i], ps73[i], ps74[i], ps75[i], ps76[i], ps77[i], ps78[i], ps79[i],
			ps80[i], ps81[i], ps82[i], ps83[i], ps84[i], ps85[i], ps86[i], ps87[i], ps88[i], ps89[i], ps90[i], ps91[i], ps92[i], ps93[i], ps94[i], ps95[i],
			ps96[i], ps97[i], ps98[i], ps99[i], ps100[i], ps101[i], ps102[i], ps103[i], ps104[i], ps105[i], ps106[i], ps107[i], ps108[i], ps109[i], ps110[i], ps111[i],
			ps112[i], ps113[i], ps114[i], ps115[i], ps116[i], ps117[i], ps118[i], ps119[i], ps120[i], ps121[i], ps122[i], ps123[i], ps124[i], ps125[i], ps126[i], ps127[i],
			ps128[i], ps129[i], ps130[i], ps131[i], ps132[i], ps133[i], ps134[i], ps135[i], ps136[i], ps137[i], ps138[i], ps139[i], ps140[i], ps141[i], ps142[i], ps143[i],
			ps144[i], ps145[i], ps146[i], ps147[i], ps148[i], ps149[i], ps150[i], ps151[i], ps152[i], ps153[i], ps154[i], ps155[i], ps156[i], ps157[i], ps158[i], ps159[i],
			ps160[i], ps161[i], ps162[i], ps163[i], ps164[i], ps165[i], ps166[i], ps167[i], ps168[i], ps169[i], ps170[i], ps171[i], ps172[i], ps173[i], ps174[i], ps175[i],
			ps176[i], ps177[i], ps178[i], ps179[i], ps180[i], ps181[i], ps182[i], ps183[i], ps184[i], ps185[i], ps186[i], ps187[i], ps188[i], ps189[i], ps190[i], ps191[i],
			ps192[i], ps193[i], ps194[i], ps195[i], ps196[i], ps197[i], ps198[i], ps199[i], ps200[i], ps201[i], ps202[i], ps203[i], ps204[i], ps205[i], ps206[i], ps207[i],
			ps208[i], ps209[i], ps210[i], ps211[i], ps212[i], ps213[i], ps214[i], ps215[i], ps216[i], ps217[i], ps218[i], ps219[i], ps220[i], ps221[i], ps222[i], ps223[i],
			ps224[i], ps225[i], ps226[i], ps227[i], ps228[i], ps229[i], ps230[i], ps231[i], ps232[i], ps233[i], ps234[i], ps235[i], ps236[i], ps237[i], ps238[i], ps239[i],
			ps240[i], ps241[i], ps242[i], ps243[i], ps244[i], ps245[i], ps246[i], ps247[i], ps248[i], ps249[i], ps250[i], ps251[i], ps252[i], ps253[i], ps254[i], ps255[i],
			ps256[i], ps257[i]), 0), tmp, h)
	}
	api.AssertIsEqual(h, root)
}
