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
=== CONT  TestCircuitSignature/BN254/marshal-public/json
=== CONT  TestCircuitSignature/BN254/marshal/binary
=== CONT  TestCircuitSignature/BN254/marshal-public/binary
=== CONT  TestCircuitSignature/BN254/plonk
=== CONT  TestCircuitSignature/BN254/groth16
--- PASS: TestCircuitSignature (1.77s)
    --- PASS: TestCircuitSignature/fuzz (1.56s)
        --- PASS: TestCircuitSignature/fuzz/BN254/groth16 (0.77s)
        --- PASS: TestCircuitSignature/fuzz/BN254/plonk (0.80s)
    --- PASS: TestCircuitSignature/BN254/marshal-public/binary (0.00s)
    --- PASS: TestCircuitSignature/BN254/marshal/binary (0.00s)
    --- PASS: TestCircuitSignature/BN254/marshal-public/json (0.00s)
    --- PASS: TestCircuitSignature/BN254/marshal/json (0.00s)
    --- PASS: TestCircuitSignature/BN254/groth16 (1.53s)
    --- PASS: TestCircuitSignature/BN254/plonk (2.06s)
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
=== CONT  TestCircuitInclusionProof/BN254/groth16
=== CONT  TestCircuitInclusionProof/BN254/plonk
=== CONT  TestCircuitInclusionProof/BN254/marshal-public/json
=== CONT  TestCircuitInclusionProof/BN254/marshal-public/binary
=== CONT  TestCircuitInclusionProof/BN254/marshal/binary
--- PASS: TestCircuitInclusionProof (6.93s)
    --- PASS: TestCircuitInclusionProof/fuzz (2.84s)
        --- PASS: TestCircuitInclusionProof/fuzz/BN254/groth16 (1.11s)
        --- PASS: TestCircuitInclusionProof/fuzz/BN254/plonk (1.73s)
    --- PASS: TestCircuitInclusionProof/BN254/marshal-public/binary (0.00s)
    --- PASS: TestCircuitInclusionProof/BN254/marshal/binary (0.00s)
    --- PASS: TestCircuitInclusionProof/BN254/marshal-public/json (0.00s)
    --- PASS: TestCircuitInclusionProof/BN254/marshal/json (0.00s)
    --- PASS: TestCircuitInclusionProof/BN254/groth16 (20.23s)
    --- PASS: TestCircuitInclusionProof/BN254/plonk (24.42s)
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
=== CONT  TestCircuitUpdateAccount/BN254/plonk
=== CONT  TestCircuitUpdateAccount/BN254/marshal-public/binary
=== CONT  TestCircuitUpdateAccount/BN254/groth16
=== CONT  TestCircuitUpdateAccount/BN254/marshal-public/json
=== CONT  TestCircuitUpdateAccount/BN254/marshal/binary
--- PASS: TestCircuitUpdateAccount (0.13s)
    --- PASS: TestCircuitUpdateAccount/fuzz (0.04s)
        --- PASS: TestCircuitUpdateAccount/fuzz/BN254/groth16 (0.02s)
        --- PASS: TestCircuitUpdateAccount/fuzz/BN254/plonk (0.02s)
    --- PASS: TestCircuitUpdateAccount/BN254/marshal-public/binary (0.00s)
    --- PASS: TestCircuitUpdateAccount/BN254/marshal/binary (0.00s)
    --- PASS: TestCircuitUpdateAccount/BN254/marshal/json (0.00s)
    --- PASS: TestCircuitUpdateAccount/BN254/marshal-public/json (0.00s)
    --- PASS: TestCircuitUpdateAccount/BN254/groth16 (0.28s)
    --- PASS: TestCircuitUpdateAccount/BN254/plonk (0.41s)
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
=== CONT  TestCircuitFull/BN254/groth16
=== CONT  TestCircuitFull/BN254/marshal-public/json
=== CONT  TestCircuitFull/BN254/marshal-public/binary
=== CONT  TestCircuitFull/BN254/plonk
=== CONT  TestCircuitFull/BN254/marshal/binary
--- PASS: TestCircuitFull (4.87s)
    --- PASS: TestCircuitFull/fuzz (0.66s)
        --- PASS: TestCircuitFull/fuzz/BN254/groth16 (0.35s)
        --- PASS: TestCircuitFull/fuzz/BN254/plonk (0.31s)
    --- PASS: TestCircuitFull/BN254/marshal-public/binary (0.00s)
    --- PASS: TestCircuitFull/BN254/marshal/binary (0.00s)
    --- PASS: TestCircuitFull/BN254/marshal-public/json (0.00s)
    --- PASS: TestCircuitFull/BN254/marshal/json (0.00s)
    --- PASS: TestCircuitFull/BN254/groth16 (21.68s)
    --- PASS: TestCircuitFull/BN254/plonk (24.85s)
=== RUN   TestOperatorReadAccount
--- PASS: TestOperatorReadAccount (0.02s)
=== RUN   TestSignTransfer
--- PASS: TestSignTransfer (0.02s)
=== RUN   TestOperatorUpdateAccount
validating PoI for sender with index '0' PASS
--- PASS: TestOperatorUpdateAccount (0.02s)
PASS
ok      github.com/iotaledger/trie.go/examples/rollup2  63.179s