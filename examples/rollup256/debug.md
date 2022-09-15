=== RUN   TestAccount
--- PASS: TestAccount (0.00s)
=== RUN   TestCircuitSignature
Root = [31 216 227 109 213 28 236 44 183 7 200 33 129 166 159 103 248 141 212 199 160 5 122 132 186 122 182 126 143 59 233 140]
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
=== CONT  TestCircuitSignature/BN254/marshal/binary
=== CONT  TestCircuitSignature/BN254/marshal-public/binary
=== CONT  TestCircuitSignature/BN254/plonk
=== CONT  TestCircuitSignature/BN254/groth16
=== CONT  TestCircuitSignature/BN254/marshal-public/json
--- PASS: TestCircuitSignature (2.04s)
    --- PASS: TestCircuitSignature/fuzz (1.76s)
        --- PASS: TestCircuitSignature/fuzz/BN254/groth16 (1.00s)
        --- PASS: TestCircuitSignature/fuzz/BN254/plonk (0.76s)
    --- PASS: TestCircuitSignature/BN254/marshal-public/binary (0.00s)
    --- PASS: TestCircuitSignature/BN254/marshal/binary (0.00s)
    --- PASS: TestCircuitSignature/BN254/marshal-public/json (0.02s)
    --- PASS: TestCircuitSignature/BN254/marshal/json (0.02s)
    --- PASS: TestCircuitSignature/BN254/groth16 (1.48s)
    --- PASS: TestCircuitSignature/BN254/plonk (1.96s)
=== RUN   TestCircuitInclusionProof
Root = [14 2 111 105 126 230 83 21 187 60 47 78 5 109 34 128 177 91 193 79 92 12 201 72 109 221 130 104 45 68 126 92]
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
FAIL	github.com/iotaledger/trie.go/examples/rollup256	44.759s
