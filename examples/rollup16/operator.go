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
	"bytes"
	"fmt"
	"hash"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/iotaledger/trie.go/models/trie_mimc1"
	"github.com/iotaledger/trie.go/models/trie_mimc1/trie_mimc1_verify"
	"github.com/iotaledger/trie.go/trie"
)

var hFunc = mimc.NewMiMC()

// BatchSize size of a batch of transactions to put in a snark
var BatchSize = 10

// Queue queue for storing the transfers (fixed size queue)
type Queue struct {
	listTransfers chan Transfer
}

// NewQueue creates a new queue, batchSize is the capaciy
func NewQueue(batchSize int) Queue {
	resChan := make(chan Transfer, batchSize)
	var res Queue
	res.listTransfers = resChan
	return res
}

// Operator represents a rollup operator
type Operator struct {
	State      []byte            // list of accounts: index ∥ nonce ∥ balance ∥ pubkeyX ∥ pubkeyY, each chunk is 256 bits
	HashState  []byte            // Hashed version of the state, each chunk is 256bits: ... ∥ H(index ∥ nonce ∥ balance ∥ pubkeyX ∥ pubkeyY)) ∥ ...
	AccountMap map[string]uint64 // hashmap of all available accounts (the key is the account.pubkey.X), the value is the index of the account in the state
	nbAccounts int               // number of accounts managed by this operator
	h          hash.Hash         // hash function used to build the Merkle Tree
	q          Queue             // queue of transfers
	batch      int               // current number of transactions in a batch
	witnesses  Circuit           // witnesses for the snark cicruit
}

// NewOperator creates a new operator.
// nbAccounts is the number of accounts managed by this operator, h is the hash function for the merkle proofs
func NewOperator(nbAccounts int) Operator {
	res := Operator{}

	// create a list of empty accounts
	res.State = make([]byte, SizeAccount*nbAccounts)

	// initialize hash of the state
	res.HashState = make([]byte, hFunc.Size()*nbAccounts)
	for i := 0; i < nbAccounts; i++ {
		hFunc.Reset()
		_, _ = hFunc.Write(res.State[i*SizeAccount : i*SizeAccount+SizeAccount])
		s := hFunc.Sum([]byte{})
		copy(res.HashState[i*hFunc.Size():(i+1)*hFunc.Size()], s)
	}

	res.AccountMap = make(map[string]uint64)
	res.nbAccounts = nbAccounts
	res.h = hFunc
	res.q = NewQueue(BatchSize)
	res.batch = 0
	return res
}

// readAccount reads the account located at index i
func (o *Operator) readAccount(i uint64) (Account, error) {

	var res Account
	err := Deserialize(&res, o.State[int(i)*SizeAccount:int(i)*SizeAccount+SizeAccount])
	if err != nil {
		return res, err
	}
	return res, nil
}

// updateAccount updates the state according to transfer
// numTransfer is the number of the transfer currently handled (between 0 and batchSize)
func (o *Operator) updateState(t Transfer, numTransfer int) error {

	var posSender, posReceiver uint64
	var ok bool

	// ext := strconv.Itoa(numTransfer)
	segmentSize := o.h.Size()
	// fmt.Println("SegmentSize =", segmentSize)

	// read sender's account
	b := t.senderPubKey.A.X.Bytes()
	if posSender, ok = o.AccountMap[string(b[:])]; !ok {
		return ErrNonExistingAccount
	}
	senderAccount, err := o.readAccount(posSender)
	if err != nil {
		return err
	}

	// read receiver's account
	b = t.receiverPubKey.A.X.Bytes()
	if posReceiver, ok = o.AccountMap[string(b[:])]; !ok {
		return ErrNonExistingAccount
	}
	receiverAccount, err := o.readAccount(posReceiver)
	if err != nil {
		return err
	}

	// set witnesses for the public keys
	o.witnesses.PublicKeysSender[numTransfer].A.X = senderAccount.pubKey.A.X
	o.witnesses.PublicKeysSender[numTransfer].A.Y = senderAccount.pubKey.A.Y
	o.witnesses.PublicKeysReceiver[numTransfer].A.X = receiverAccount.pubKey.A.X
	o.witnesses.PublicKeysReceiver[numTransfer].A.Y = receiverAccount.pubKey.A.Y

	// set witnesses for the accounts before update
	o.witnesses.SenderAccountsBefore[numTransfer].Index = senderAccount.index
	o.witnesses.SenderAccountsBefore[numTransfer].Nonce = senderAccount.nonce
	o.witnesses.SenderAccountsBefore[numTransfer].Balance = senderAccount.balance

	o.witnesses.ReceiverAccountsBefore[numTransfer].Index = receiverAccount.index
	o.witnesses.ReceiverAccountsBefore[numTransfer].Nonce = receiverAccount.nonce
	o.witnesses.ReceiverAccountsBefore[numTransfer].Balance = receiverAccount.balance

	//  Set witnesses for the proof of inclusion of sender and receivers account before update
	var buf bytes.Buffer
	_, err = buf.Write(o.HashState)
	if err != nil {
		return err
	}

	index_to_segment := make(map[uint64][]byte)
	store := trie.NewInMemoryKVStore()
	model := trie_mimc1.New(trie.PathArity16)
	tr := trie.New(model, store, nil)
	tr.ReadAll(&buf, segmentSize, index_to_segment)
	tr.Commit()
	rootCommitment := trie.RootCommitment(tr)
	// fmt.Printf("root commitment: %v\n", rootCommitment)

	// This proof is analogous to the `proofSet` in gnark
	proof := model.Proof(index_to_segment[posSender], tr)
	// validate proof
	err = trie_mimc1_verify.Validate(proof, rootCommitment.Bytes())

	// fmt.Print("Root = ")
	// fmt.Println(rootCommitment.Bytes())
	if err != nil {
		fmt.Printf("validating PoI for '%d': %s\n", posSender, err.Error())
	} else {
		fmt.Printf("validating PoI for sender with index '%d' PASS\n", posSender)
	}

	trieSenderProofs, trieSenderPath := generateHashes(proof.Path)

	buf.Reset() // the buffer needs to be reset
	_, err = buf.Write(o.HashState)
	if err != nil {
		return err
	}
	tr.ReadAll(&buf, segmentSize, index_to_segment)
	proof = model.Proof(index_to_segment[posReceiver], tr)
	// validate proof
	// err = trie_mimc_verify.Validate(proof, rootCommitment.Bytes())
	// if err != nil {
	// 	fmt.Printf("validating PoI for '%d': %s\n", posReceiver, err.Error())
	// } else {
	// 	fmt.Printf("validating PoI for receiver with index '%d' PASS\n", posReceiver)
	// }

	trieReceiverProofs, trieReceiverPath := generateHashes(proof.Path)
	// fmt.Printf("trieSenderProofs %d \n", len(trieSenderProofs))
	o.witnesses.TrieRootHashesBefore[numTransfer] = rootCommitment.Bytes()
	for i := 0; i < depth; i++ {
		for j := 0; j < proofSetSize; j++ {
			// fmt.Printf("j = %d, i = %d, hash = ", i, j)
			// fmt.Println(trieSenderProofs[i][j])
			o.witnesses.TrieProofsSenderBefore[numTransfer][j][i] = trieSenderProofs[i][j]
			o.witnesses.TrieProofsReceiverBefore[numTransfer][j][i] = trieReceiverProofs[i][j]
		}
		if i < depth-1 {
			// fmt.Printf("Path = %d\n", trieSenderPath[i])
			o.witnesses.TriePathSenderBefore[numTransfer][i] = trieSenderPath[i]
			o.witnesses.TriePathReceiverBefore[numTransfer][i] = trieReceiverPath[i]
		}
	}

	// set witnesses for the transfer
	o.witnesses.Transfers[numTransfer].Amount = t.amount
	o.witnesses.Transfers[numTransfer].Signature.R.X = t.signature.R.X
	o.witnesses.Transfers[numTransfer].Signature.R.Y = t.signature.R.Y
	o.witnesses.Transfers[numTransfer].Signature.S = t.signature.S[:]

	// verifying the signature. The msg is the hash (o.h) of the transfer
	// nonce ∥ amount ∥ senderpubKey(x&y) ∥ receiverPubkey(x&y)
	resSig, err := t.Verify(o.h)
	if err != nil {
		return err
	}
	if !resSig {
		return ErrWrongSignature
	}

	// checks if the amount is correct
	var bAmount, bBalance big.Int
	receiverAccount.balance.ToBigIntRegular(&bBalance)
	t.amount.ToBigIntRegular(&bAmount)
	if bAmount.Cmp(&bBalance) == 1 {
		return ErrAmountTooHigh
	}

	// check if the nonce is correct
	if t.nonce != senderAccount.nonce {
		return ErrNonce
	}

	// update the balance of the sender
	senderAccount.balance.Sub(&senderAccount.balance, &t.amount)

	// update the balance of the receiver
	receiverAccount.balance.Add(&receiverAccount.balance, &t.amount)

	// update the nonce of the sender
	senderAccount.nonce++

	// set the witnesses for the account after update
	o.witnesses.SenderAccountsAfter[numTransfer].Index = senderAccount.index
	o.witnesses.SenderAccountsAfter[numTransfer].Nonce = senderAccount.nonce
	o.witnesses.SenderAccountsAfter[numTransfer].Balance = senderAccount.balance

	o.witnesses.ReceiverAccountsAfter[numTransfer].Index = receiverAccount.index
	o.witnesses.ReceiverAccountsAfter[numTransfer].Nonce = receiverAccount.nonce
	o.witnesses.ReceiverAccountsAfter[numTransfer].Balance = receiverAccount.balance

	// update the state of the operator
	copy(o.State[int(posSender)*SizeAccount:], senderAccount.Serialize())
	o.h.Reset()
	_, _ = o.h.Write(senderAccount.Serialize())
	bufSender := o.h.Sum([]byte{})
	copy(o.HashState[int(posSender)*o.h.Size():(int(posSender)+1)*o.h.Size()], bufSender)

	copy(o.State[int(posReceiver)*SizeAccount:], receiverAccount.Serialize())
	o.h.Reset()
	_, _ = o.h.Write(receiverAccount.Serialize())
	bufReceiver := o.h.Sum([]byte{})
	copy(o.HashState[int(posReceiver)*o.h.Size():(int(posReceiver)+1)*o.h.Size()], bufReceiver)

	//  Set witnesses for the proof of inclusion of sender and receivers account after update
	buf.Reset()
	_, err = buf.Write(o.HashState)
	if err != nil {
		return err
	}

	tr.ReadAll(&buf, segmentSize, index_to_segment)
	tr.Commit()
	rootCommitment = trie.RootCommitment(tr)
	proof = model.Proof(index_to_segment[posSender], tr)
	// validate proof
	// err = trie_mimc_verify.Validate(proof, rootCommitment.Bytes())
	// if err != nil {
	// 	fmt.Printf("validating PoI for '%d': %s\n", posReceiver, err.Error())
	// } else {
	// 	fmt.Printf("validating PoI for sender with index '%d' PASS\n", posReceiver)
	// }
	trieSenderProofs, trieSenderPath = generateHashes(proof.Path)

	buf.Reset() // the buffer needs to be reset
	_, err = buf.Write(o.HashState)
	if err != nil {
		return err
	}
	tr.ReadAll(&buf, segmentSize, index_to_segment)
	proof = model.Proof(index_to_segment[posReceiver], tr)
	// validate proof
	// err = trie_mimc_verify.Validate(proof, rootCommitment.Bytes())
	// if err != nil {
	// 	fmt.Printf("validating PoI for '%d': %s\n", posReceiver, err.Error())
	// } else {
	// 	fmt.Printf("validating PoI for receiver with index '%d' PASS\n", posReceiver)
	// }
	trieReceiverProofs, trieReceiverPath = generateHashes(proof.Path)

	o.witnesses.TrieRootHashesAfter[numTransfer] = rootCommitment.Bytes()
	for i := 0; i < depth; i++ {
		for j := 0; j < proofSetSize; j++ {
			o.witnesses.TrieProofsSenderAfter[numTransfer][j][i] = trieSenderProofs[i][j]
			o.witnesses.TrieProofsReceiverAfter[numTransfer][j][i] = trieReceiverProofs[i][j]
		}
		if i < depth-1 {
			o.witnesses.TriePathSenderAfter[numTransfer][i] = trieSenderPath[i]
			o.witnesses.TriePathReceiverAfter[numTransfer][i] = trieReceiverPath[i]
		}

	}

	return nil
}

func generateHashes(proofs []*trie_mimc1.ProofElement) ([][proofSetSize][]byte, []int) {
	proofsLength := len(proofs)
	// print(proofsLength)
	hashesAll := make([][proofSetSize][]byte, depth)
	path := make([]int, depth-1)
	for i, p := range proofs {

		// Put in reverse order
		hashes := [proofSetSize][]byte{}
		for j := 0; j < proofSetSize; j++ {
			hashes[j] = make([]byte, 32)
		}
		if i == proofsLength-1 {
			for j := 0; j < proofSetSize-2; j++ {
				copy(hashes[j][:], p.Children[byte(j)])
			}
			copy(hashes[proofSetSize-2][:], p.Terminal)
			copy(hashes[proofSetSize-1][:], trie_mimc1.HashData(p.PathFragment))
		} else {
			for j := 0; j < proofSetSize-2; j++ {
				if j != p.ChildIndex {
					copy(hashes[j][:], p.Children[byte(j)])
				}
				copy(hashes[j][:], p.Children[byte(j)])
			}
			copy(hashes[proofSetSize-2][:], p.Terminal)
			copy(hashes[proofSetSize-1][:], trie_mimc1.HashData(p.PathFragment))
			path[proofsLength-i-2] = p.ChildIndex
		}
		hashesAll[proofsLength-i-1] = hashes
	}
	return hashesAll, path
}
