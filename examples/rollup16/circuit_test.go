/*
Copyright © 2020 ConsenSys

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rollup

import (
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
	"github.com/consensys/gnark/test"
	"github.com/iotaledger/trie.go/models/trie_mimc1"
)

type circuitSignature Circuit

// Circuit implements part of the rollup circuit only by delcaring a subset of the constraints
func (t *circuitSignature) Define(api frontend.API) error {
	if err := (*Circuit)(t).postInit(api); err != nil {
		return err
	}
	hFunc, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}
	return verifyTransferSignature(api, t.Transfers[0], hFunc)
}

func TestCircuitSignature(t *testing.T) {

	const nbAccounts = 10

	operator, users := createOperator(nbAccounts)

	// read accounts involved in the transfer
	sender, err := operator.readAccount(0)
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := operator.readAccount(1)
	if err != nil {
		t.Fatal(err)
	}

	// create the transfer and sign it
	amount := uint64(10)
	transfer := NewTransfer(amount, sender.pubKey, receiver.pubKey, sender.nonce)

	// sign the transfer
	_, err = transfer.Sign(users[0], operator.h)
	if err != nil {
		t.Fatal(err)
	}

	// update the state from the received transfer
	err = operator.updateState(transfer, 0)
	if err != nil {
		t.Fatal(err)
	}

	// verifies the signature of the transfer
	assert := test.NewAssert(t)

	var signatureCircuit circuitSignature
	assert.ProverSucceeded(&signatureCircuit, &operator.witnesses, test.WithCurves(ecc.BN254), test.WithCompileOpts(frontend.IgnoreUnconstrainedInputs()))

}

type circuitInclusionProof Circuit

// Circuit implements part of the rollup circuit only by delcaring a subset of the constraints
func (t *circuitInclusionProof) Define(api frontend.API) error {
	if err := (*Circuit)(t).postInit(api); err != nil {
		return err
	}
	hashFunc, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}
	trie_mimc1.Validate16(api, hashFunc, t.TrieRootHashesBefore[0],
		t.TrieProofsSenderBefore[0][0][:],
		t.TrieProofsSenderBefore[0][1][:],
		t.TrieProofsSenderBefore[0][2][:],
		t.TrieProofsSenderBefore[0][3][:],
		t.TrieProofsSenderBefore[0][4][:],
		t.TrieProofsSenderBefore[0][5][:],
		t.TrieProofsSenderBefore[0][6][:],
		t.TrieProofsSenderBefore[0][7][:],
		t.TrieProofsSenderBefore[0][8][:],
		t.TrieProofsSenderBefore[0][9][:],
		t.TrieProofsSenderBefore[0][10][:],
		t.TrieProofsSenderBefore[0][11][:],
		t.TrieProofsSenderBefore[0][12][:],
		t.TrieProofsSenderBefore[0][13][:],
		t.TrieProofsSenderBefore[0][14][:],
		t.TrieProofsSenderBefore[0][15][:],
		t.TrieProofsSenderBefore[0][16][:],
		t.TrieProofsSenderBefore[0][17][:],
		t.TriePathSenderBefore[0][:])
	trie_mimc1.Validate16(api, hashFunc, t.TrieRootHashesBefore[0],
		t.TrieProofsReceiverBefore[0][0][:],
		t.TrieProofsReceiverBefore[0][1][:],
		t.TrieProofsReceiverBefore[0][2][:],
		t.TrieProofsReceiverBefore[0][3][:],
		t.TrieProofsReceiverBefore[0][4][:],
		t.TrieProofsReceiverBefore[0][5][:],
		t.TrieProofsReceiverBefore[0][6][:],
		t.TrieProofsReceiverBefore[0][7][:],
		t.TrieProofsReceiverBefore[0][8][:],
		t.TrieProofsReceiverBefore[0][9][:],
		t.TrieProofsReceiverBefore[0][10][:],
		t.TrieProofsReceiverBefore[0][11][:],
		t.TrieProofsReceiverBefore[0][12][:],
		t.TrieProofsReceiverBefore[0][13][:],
		t.TrieProofsReceiverBefore[0][14][:],
		t.TrieProofsReceiverBefore[0][15][:],
		t.TrieProofsReceiverBefore[0][16][:],
		t.TrieProofsReceiverBefore[0][17][:],
		t.TriePathReceiverBefore[0][:])

	trie_mimc1.Validate16(api, hashFunc, t.TrieRootHashesAfter[0],
		t.TrieProofsSenderAfter[0][0][:],
		t.TrieProofsSenderAfter[0][1][:],
		t.TrieProofsSenderAfter[0][2][:],
		t.TrieProofsSenderAfter[0][3][:],
		t.TrieProofsSenderAfter[0][4][:],
		t.TrieProofsSenderAfter[0][5][:],
		t.TrieProofsSenderAfter[0][6][:],
		t.TrieProofsSenderAfter[0][7][:],
		t.TrieProofsSenderAfter[0][8][:],
		t.TrieProofsSenderAfter[0][9][:],
		t.TrieProofsSenderAfter[0][10][:],
		t.TrieProofsSenderAfter[0][11][:],
		t.TrieProofsSenderAfter[0][12][:],
		t.TrieProofsSenderAfter[0][13][:],
		t.TrieProofsSenderAfter[0][14][:],
		t.TrieProofsSenderAfter[0][15][:],
		t.TrieProofsSenderAfter[0][16][:],
		t.TrieProofsSenderAfter[0][17][:],
		t.TriePathSenderAfter[0][:])
	trie_mimc1.Validate16(api, hashFunc, t.TrieRootHashesAfter[0],
		t.TrieProofsReceiverAfter[0][0][:],
		t.TrieProofsReceiverAfter[0][1][:],
		t.TrieProofsReceiverAfter[0][2][:],
		t.TrieProofsReceiverAfter[0][3][:],
		t.TrieProofsReceiverAfter[0][4][:],
		t.TrieProofsReceiverAfter[0][5][:],
		t.TrieProofsReceiverAfter[0][6][:],
		t.TrieProofsReceiverAfter[0][7][:],
		t.TrieProofsReceiverAfter[0][8][:],
		t.TrieProofsReceiverAfter[0][9][:],
		t.TrieProofsReceiverAfter[0][10][:],
		t.TrieProofsReceiverAfter[0][11][:],
		t.TrieProofsReceiverAfter[0][12][:],
		t.TrieProofsReceiverAfter[0][13][:],
		t.TrieProofsReceiverAfter[0][14][:],
		t.TrieProofsReceiverAfter[0][15][:],
		t.TrieProofsReceiverAfter[0][16][:],
		t.TrieProofsReceiverAfter[0][17][:],
		t.TriePathReceiverAfter[0][:])

	return nil
}

func TestCircuitInclusionProof(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping rollup tests for circleCI")
	}

	operator, users := createOperator(nbAccounts)

	// read accounts involved in the transfer
	sender, err := operator.readAccount(0)
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := operator.readAccount(1)
	if err != nil {
		t.Fatal(err)
	}

	// create the transfer and sign it
	amount := uint64(16)
	transfer := NewTransfer(amount, sender.pubKey, receiver.pubKey, sender.nonce)

	// sign the transfer
	_, err = transfer.Sign(users[0], operator.h)
	if err != nil {
		t.Fatal(err)
	}

	// update the state from the received transfer
	err = operator.updateState(transfer, 0)
	if err != nil {
		t.Fatal(err)
	}

	// verifies the proofs of inclusion of the transfer
	assert := test.NewAssert(t)

	var inclusionProofCircuit circuitInclusionProof

	assert.ProverSucceeded(&inclusionProofCircuit, &operator.witnesses, test.WithCurves(ecc.BN254), test.WithCompileOpts(frontend.IgnoreUnconstrainedInputs()))

}

type circuitUpdateAccount Circuit

// Circuit implements part of the rollup circuit only by delcaring a subset of the constraints
func (t *circuitUpdateAccount) Define(api frontend.API) error {
	if err := (*Circuit)(t).postInit(api); err != nil {
		return err
	}
	verifyAccountUpdated(api, t.SenderAccountsBefore[0], t.ReceiverAccountsBefore[0],
		t.SenderAccountsAfter[0], t.ReceiverAccountsAfter[0], t.Transfers[0].Amount)
	return nil
}

func TestCircuitUpdateAccount(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping rollup tests for circleCI")
	}

	operator, users := createOperator(nbAccounts)

	// read accounts involved in the transfer
	sender, err := operator.readAccount(0)
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := operator.readAccount(1)
	if err != nil {
		t.Fatal(err)
	}

	// create the transfer and sign it
	amount := uint64(10)
	transfer := NewTransfer(amount, sender.pubKey, receiver.pubKey, sender.nonce)

	// sign the transfer
	_, err = transfer.Sign(users[0], operator.h)
	if err != nil {
		t.Fatal(err)
	}

	// update the state from the received transfer
	err = operator.updateState(transfer, 0)
	if err != nil {
		t.Fatal(err)
	}

	assert := test.NewAssert(t)

	var updateAccountCircuit circuitUpdateAccount

	assert.ProverSucceeded(&updateAccountCircuit, &operator.witnesses, test.WithCurves(ecc.BN254), test.WithCompileOpts(frontend.IgnoreUnconstrainedInputs()))

}

func TestCircuitFull(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping rollup tests for circleCI")
	}

	operator, users := createOperator(nbAccounts)

	// read accounts involved in the transfer
	sender, err := operator.readAccount(0)
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := operator.readAccount(1)
	if err != nil {
		t.Fatal(err)
	}

	// create the transfer and sign it
	amount := uint64(10)
	transfer := NewTransfer(amount, sender.pubKey, receiver.pubKey, sender.nonce)

	// sign the transfer
	_, err = transfer.Sign(users[0], operator.h)
	if err != nil {
		t.Fatal(err)
	}

	// update the state from the received transfer
	err = operator.updateState(transfer, 0)
	if err != nil {
		t.Fatal(err)
	}

	assert := test.NewAssert(t)
	// verifies the proofs of inclusion of the transfer

	var rollupCircuit Circuit

	// TODO full circuit has some unconstrained inputs, that's odd.
	assert.ProverSucceeded(&rollupCircuit, &operator.witnesses, test.WithCurves(ecc.BN254), test.WithCompileOpts(frontend.IgnoreUnconstrainedInputs()))

}
