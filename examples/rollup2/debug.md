=== RUN   TestAccount
--- PASS: TestAccount (0.00s)
=== RUN   TestCircuitSignature
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
=== CONT  TestCircuitSignature/BN254/plonk
=== CONT  TestCircuitSignature/BN254/marshal/binary
=== CONT  TestCircuitSignature/BN254/marshal-public/json
--- PASS: TestCircuitSignature (1.66s)
    --- PASS: TestCircuitSignature/fuzz (1.45s)
        --- PASS: TestCircuitSignature/fuzz/BN254/groth16 (0.76s)
        --- PASS: TestCircuitSignature/fuzz/BN254/plonk (0.69s)
    --- PASS: TestCircuitSignature/BN254/marshal-public/binary (0.00s)
    --- PASS: TestCircuitSignature/BN254/marshal/binary (0.00s)
    --- PASS: TestCircuitSignature/BN254/marshal-public/json (0.00s)
    --- PASS: TestCircuitSignature/BN254/marshal/json (0.00s)
    --- PASS: TestCircuitSignature/BN254/groth16 (1.48s)
    --- PASS: TestCircuitSignature/BN254/plonk (2.02s)
=== RUN   TestCircuitInclusionProof
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
=== PAUSE TestCircuitInclusionProof/BN254/groth16
=== RUN   TestCircuitInclusionProof/BN254/plonk
=== PAUSE TestCircuitInclusionProof/BN254/plonk
=== RUN   TestCircuitInclusionProof/fuzz
=== RUN   TestCircuitInclusionProof/fuzz/BN254/groth16
=== RUN   TestCircuitInclusionProof/fuzz/BN254/plonk
=== CONT  TestCircuitInclusionProof/BN254/marshal/json
=== CONT  TestCircuitInclusionProof/BN254/marshal/binary
=== CONT  TestCircuitInclusionProof/BN254/groth16
=== CONT  TestCircuitInclusionProof/BN254/marshal-public/json
=== CONT  TestCircuitInclusionProof/BN254/plonk
=== CONT  TestCircuitInclusionProof/BN254/marshal-public/binary
--- PASS: TestCircuitInclusionProof (13.08s)
    --- PASS: TestCircuitInclusionProof/fuzz (4.71s)
        --- PASS: TestCircuitInclusionProof/fuzz/BN254/groth16 (2.47s)
        --- PASS: TestCircuitInclusionProof/fuzz/BN254/plonk (2.24s)
    --- PASS: TestCircuitInclusionProof/BN254/marshal/binary (0.00s)
    --- PASS: TestCircuitInclusionProof/BN254/marshal-public/binary (0.00s)
    --- PASS: TestCircuitInclusionProof/BN254/marshal-public/json (0.00s)
    --- PASS: TestCircuitInclusionProof/BN254/marshal/json (0.00s)
    --- PASS: TestCircuitInclusionProof/BN254/groth16 (43.32s)
    --- PASS: TestCircuitInclusionProof/BN254/plonk (51.65s)
=== RUN   TestCircuitUpdateAccount
validating PoI for sender with index '0' PASS
=== RUN   TestCircuitUpdateAccount/BN254/marshal/json
=== PAUSE TestCircuitUpdateAccount/BN254/marshal/json
=== RUN   TestCircuitUpdateAccount/BN254/marshal/binary
=== PAUSE TestCircuitUpdateAccount/BN254/marshal/binary
=== RUN   TestCircuitUpdateAccount/BN254/marshal-public/json
=== PAUSE TestCircuitUpdateAccount/BN254/marshal-public/json
=== RUN   TestCircuitUpdateAccount/BN254/marshal-public/binary
=== PAUSE TestCircuitUpdateAccount/BN254/marshal-public/binary
=== RUN   TestCircuitUpdateAccount/BN254/groth16
=== PAUSE TestCircuitUpdateAccount/BN254/groth16
=== RUN   TestCircuitUpdateAccount/BN254/plonk
=== PAUSE TestCircuitUpdateAccount/BN254/plonk
=== RUN   TestCircuitUpdateAccount/fuzz
=== RUN   TestCircuitUpdateAccount/fuzz/BN254/groth16
=== RUN   TestCircuitUpdateAccount/fuzz/BN254/plonk
=== CONT  TestCircuitUpdateAccount/BN254/marshal/json
=== CONT  TestCircuitUpdateAccount/BN254/groth16
=== CONT  TestCircuitUpdateAccount/BN254/marshal-public/json
=== CONT  TestCircuitUpdateAccount/BN254/marshal-public/binary
=== CONT  TestCircuitUpdateAccount/BN254/marshal/binary
=== CONT  TestCircuitUpdateAccount/BN254/plonk
--- PASS: TestCircuitUpdateAccount (0.12s)
    --- PASS: TestCircuitUpdateAccount/fuzz (0.04s)
        --- PASS: TestCircuitUpdateAccount/fuzz/BN254/groth16 (0.01s)
        --- PASS: TestCircuitUpdateAccount/fuzz/BN254/plonk (0.02s)
    --- PASS: TestCircuitUpdateAccount/BN254/marshal-public/binary (0.00s)
    --- PASS: TestCircuitUpdateAccount/BN254/marshal/binary (0.00s)
    --- PASS: TestCircuitUpdateAccount/BN254/marshal-public/json (0.00s)
    --- PASS: TestCircuitUpdateAccount/BN254/marshal/json (0.00s)
    --- PASS: TestCircuitUpdateAccount/BN254/groth16 (0.30s)
    --- PASS: TestCircuitUpdateAccount/BN254/plonk (0.40s)
=== RUN   TestCircuitFull
validating PoI for sender with index '0' PASS
=== RUN   TestCircuitFull/BN254/marshal/json
=== PAUSE TestCircuitFull/BN254/marshal/json
=== RUN   TestCircuitFull/BN254/marshal/binary
=== PAUSE TestCircuitFull/BN254/marshal/binary
=== RUN   TestCircuitFull/BN254/marshal-public/json
=== PAUSE TestCircuitFull/BN254/marshal-public/json
=== RUN   TestCircuitFull/BN254/marshal-public/binary
=== PAUSE TestCircuitFull/BN254/marshal-public/binary
=== RUN   TestCircuitFull/BN254/groth16
=== PAUSE TestCircuitFull/BN254/groth16
=== RUN   TestCircuitFull/BN254/plonk
=== PAUSE TestCircuitFull/BN254/plonk
=== RUN   TestCircuitFull/fuzz
=== RUN   TestCircuitFull/fuzz/BN254/groth16
=== RUN   TestCircuitFull/fuzz/BN254/plonk
=== CONT  TestCircuitFull/BN254/marshal/json
=== CONT  TestCircuitFull/BN254/marshal/binary
=== CONT  TestCircuitFull/BN254/groth16
=== CONT  TestCircuitFull/BN254/marshal-public/json
=== CONT  TestCircuitFull/BN254/plonk
=== CONT  TestCircuitFull/BN254/marshal-public/binary
--- PASS: TestCircuitFull (9.83s)
    --- PASS: TestCircuitFull/fuzz (1.27s)
        --- PASS: TestCircuitFull/fuzz/BN254/groth16 (0.60s)
        --- PASS: TestCircuitFull/fuzz/BN254/plonk (0.67s)
    --- PASS: TestCircuitFull/BN254/marshal/binary (0.00s)
    --- PASS: TestCircuitFull/BN254/marshal-public/binary (0.00s)
    --- PASS: TestCircuitFull/BN254/marshal-public/json (0.00s)
    --- PASS: TestCircuitFull/BN254/marshal/json (0.00s)
    --- PASS: TestCircuitFull/BN254/groth16 (43.53s)
    --- PASS: TestCircuitFull/BN254/plonk (49.60s)
=== RUN   TestOperatorReadAccount
--- PASS: TestOperatorReadAccount (0.02s)
=== RUN   TestSignTransfer
--- PASS: TestSignTransfer (0.02s)
=== RUN   TestOperatorUpdateAccount
validating PoI for sender with index '0' PASS
--- PASS: TestOperatorUpdateAccount (0.02s)
PASS
ok  	github.com/iotaledger/trie.go/examples/rollup2	122.945s
