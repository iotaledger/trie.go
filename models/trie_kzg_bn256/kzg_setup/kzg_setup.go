// the program kzg_setup generates new trusted setup for the KZG calculations from the
// secret entered from the keyboard and saves generated setup into the file
// Usage: kzg_setup <file name>
package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"syscall"

	trie_kzg_bn2562 "github.com/iotaledger/trie.go/models/trie_kzg_bn256"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/term"
)

const (
	minSeed     = 20
	defaultFile = "examples.setup"
	D           = 258
)

var suite = bn256.NewSuite()

func main() {
	if len(os.Args) > 2 {
		fmt.Printf("Usage: kzg_setup <file name>\n")
		return
	}
	fname := defaultFile
	if len(os.Args) == 2 {
		fname = os.Args[1]
	}
	fmt.Printf("generating new trusted KZG setup to file '%s'. D = %d... \n", fname, D)
	var seed []byte
	var err error
	for {
		fmt.Printf("please enter seed > %d symbols and press ENTER (CTRL-C to exit) > ", minSeed)
		seed, err = term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Printf("\nerror: %v\n", err)
			continue
		}
		if len(seed) < minSeed {
			fmt.Printf("\nerror: seed too short\n")
			continue
		}
		fmt.Println()
		break
	}
	h := blake2b.Sum256(seed)
	// destroy seed
	for i := range seed {
		seed[i] = 0
	}
	// hash seed random number of times
	for i := 0; i < 10+rand.Intn(90); i++ {
		h = blake2b.Sum256(h[:])
	}
	s := suite.G1().Scalar()
	s.SetBytes(h[:])
	h = [32]byte{} // destroy secret
	omega, _ := trie_kzg_bn2562.GenRootOfUnityQuasiPrimitive(suite, D)
	tr, err := trie_kzg_bn2562.TrustedSetupFromSecretPowers(suite, D, omega, s)
	s.Zero() // // destroy secret
	if err != nil {
		panic(err)
	}
	writeToFile(tr, fname)
}

func writeToFile(tr *trie_kzg_bn2562.TrustedSetup, fname string) {
	generateGoFile := strings.HasSuffix(fname, ".go")

	if !generateGoFile {
		err := ioutil.WriteFile(fname, tr.Bytes(), 0600)
		checkErr(err)
		fmt.Printf("success. The trusted setup has been generated and saved into the binary file '%s'\n", fname)
		if _, err := trie_kzg_bn2562.TrustedSetupFromFile(suite, fname); err != nil {
			fmt.Printf("reading trusted setup back from file '%s': %v\nFAIL\n", fname, err)
		} else {
			fmt.Printf("reading trusted setup back from file '%s': OK\nSUCCESS\n", fname)
		}
	} else {
		f, err := os.Create(fname)
		checkErr(err)
		defer func() { _ = f.Close() }()

		data := tr.Bytes()
		dataStr := ""
		i := 0
		for {
			d := data[i:]
			step := 40
			if len(d) < 40 {
				step = len(d)
			}
			i += step

			dataStr += fmt.Sprintf("\"%s\"", hex.EncodeToString(d[:step]))
			if i >= len(data) {
				dataStr += "\n"
				break
			}
			dataStr += " +\n"
		}
		res := strings.Replace(ftemplate, "{{HEX DATA}}", dataStr, 1)
		fmt.Fprint(f, res)
	}
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

var ftemplate = `// package main contains Go source code which has been generated automatically. Package name 'main' is a placeholder
package main

import "encoding/hex"

var binHex = {{HEX DATA}}

func GetTrustedSetupBin() []byte {
	ret, err := hex.DecodeString(binHex)
    if err != nil{
        panic(err)
	} 
	return ret
}
`
