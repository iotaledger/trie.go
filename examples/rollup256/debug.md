=== RUN   TestAccount
--- PASS: TestAccount (0.00s)
=== RUN   TestCircuitSignature
Root = [45 174 230 205 44 253 5 154 85 193 213 131 69 65 214 241 99 24 89 9 130 80 228 52 238 74 176 66 136 111 114 84]
validating PoI for sender with index '0' PASS
=== RUN   TestCircuitSignature/BN254/marshal/json
=== PAUSE TestCircuitSignature/BN254/marshal/json
=== RUN   TestCircuitSignature/BN254/marshal/binary
=== PAUSE TestCircuitSignature/BN254/marshal/binary
=== RUN   TestCircuitSignature/BN254/marshal-public/json
=== PAUSE TestCircuitSignature/BN254/marshal-public/json
=== RUN   TestCircuitSignature/BN254/marshal-public/binary
=== PAUSE TestCircuitSignature/BN254/marshal-public/binary
=== RUN   TestCircuitSignature/BN254/groth16
=== PAUSE TestCircuitSignature/BN254/groth16
=== RUN   TestCircuitSignature/BN254/plonk
=== PAUSE TestCircuitSignature/BN254/plonk
=== RUN   TestCircuitSignature/fuzz
=== RUN   TestCircuitSignature/fuzz/BN254/groth16
=== RUN   TestCircuitSignature/fuzz/BN254/plonk
=== CONT  TestCircuitSignature/BN254/marshal/json
=== CONT  TestCircuitSignature/BN254/marshal-public/binary
=== CONT  TestCircuitSignature/BN254/groth16
=== CONT  TestCircuitSignature/BN254/marshal/binary
=== CONT  TestCircuitSignature/BN254/marshal-public/json
=== CONT  TestCircuitSignature/BN254/plonk
--- PASS: TestCircuitSignature (2.30s)
    --- PASS: TestCircuitSignature/fuzz (2.03s)
        --- PASS: TestCircuitSignature/fuzz/BN254/groth16 (0.97s)
        --- PASS: TestCircuitSignature/fuzz/BN254/plonk (1.05s)
    --- PASS: TestCircuitSignature/BN254/marshal-public/binary (0.00s)
    --- PASS: TestCircuitSignature/BN254/marshal/binary (0.00s)
    --- PASS: TestCircuitSignature/BN254/marshal-public/json (0.01s)
    --- PASS: TestCircuitSignature/BN254/marshal/json (0.02s)
    --- PASS: TestCircuitSignature/BN254/groth16 (1.56s)
    --- PASS: TestCircuitSignature/BN254/plonk (2.04s)
=== RUN   TestCircuitInclusionProof
Root = [30 61 254 9 216 220 20 183 215 246 228 91 176 121 178 29 255 69 217 86 146 198 19 40 69 77 246 206 239 101 78 40]
validating PoI for sender with index '0' PASS
=== RUN   TestCircuitInclusionProof/BN254/marshal/json
=== PAUSE TestCircuitInclusionProof/BN254/marshal/json
=== RUN   TestCircuitInclusionProof/BN254/marshal/binary
=== PAUSE TestCircuitInclusionProof/BN254/marshal/binary
=== RUN   TestCircuitInclusionProof/BN254/marshal-public/json
=== PAUSE TestCircuitInclusionProof/BN254/marshal-public/json
=== RUN   TestCircuitInclusionProof/BN254/marshal-public/binary
=== PAUSE TestCircuitInclusionProof/BN254/marshal-public/binary
=== RUN   TestCircuitInclusionProof/BN254/groth16
signal: killed
FAIL	github.com/iotaledger/trie.go/examples/rollup256	63.322s
