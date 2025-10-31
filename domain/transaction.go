package domain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"

	"fmt"
	"log"
	"math/big"
)

type TxType int

const (
	TxTypeTransfer       TxType = iota
	TxTypeContractDeploy TxType = 1
	TxTypeContractCall   TxType = 2
)

type TxInput struct {
	TxID      []byte
	VoutIndex int
	Signature []byte
	PublicKey []byte
}

type TxOutput struct {
	Value      int64
	PubKeyHash []byte
}

type Transaction struct {
	ID      []byte
	Vin     []TxInput  `json:"vinList"`
	Vout    []TxOutput `json:"voutList"`
	Type    TxType     `json:"type"`
	Payload []byte     `json:"payload"`
}

type jsonTxInputHash struct {
	TxID      string `json:"txId"`
	VoutIndex int    `json:"voutIndex"`
	Signature string `json:"signature"`
	PublicKey string `json:"publicKey"`
}

type jsonTxOutputHash struct {
	Value      string `json:"value"`
	PubKeyHash string `json:"pubKeyHash"`
}

type jsonTransactionHash struct {
	ID      string             `json:"id"`
	Vin     []jsonTxInputHash  `json:"vinList"`
	Vout    []jsonTxOutputHash `json:"voutList"`
	Type    TxType             `json:"type"`
	Payload string             `json:"payload"`
}

func (tx *Transaction) Hash() []byte {

	hashCopy := jsonTransactionHash{
		ID:      base64.StdEncoding.EncodeToString(tx.ID),
		Payload: base64.StdEncoding.EncodeToString(tx.Payload),
		Type:    tx.Type,
		Vin:     make([]jsonTxInputHash, len(tx.Vin)),
		Vout:    make([]jsonTxOutputHash, len(tx.Vout)),
	}

	for i, vin := range tx.Vin {
		hashCopy.Vin[i] = jsonTxInputHash{
			TxID:      base64.StdEncoding.EncodeToString(vin.TxID),
			VoutIndex: vin.VoutIndex,
			Signature: "",
			PublicKey: base64.StdEncoding.EncodeToString(vin.PublicKey),
		}
	}

	for i, out := range tx.Vout {
		hashCopy.Vout[i] = jsonTxOutputHash{
			Value:      fmt.Sprintf("%d", out.Value),
			PubKeyHash: base64.StdEncoding.EncodeToString(out.PubKeyHash),
		}
	}

	jsonData, err := json.Marshal(hashCopy)
	if err != nil {
		Handle(fmt.Errorf("failed to marshal hash struct: %v", err))
	}

	fmt.Println("--- Backend JSON Hashed (struct) ---")
	logData, _ := json.MarshalIndent(hashCopy, "", "  ")
	fmt.Println(string(logData))
	fmt.Println("----------------------------------")

	hash := sha256.Sum256(jsonData)
	return hash[:]
}

func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}
	hash := sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].TxID) == 0
}

func (out *TxOutput) Lock(address string) {
	pubKeyHash := DecodeAddress(address)
	out.PubKeyHash = pubKeyHash
}

func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}

func (in *TxInput) CanBeUnlockedWith(pubKeyHash []byte) bool {
	lockingHash := HashPubKey(in.PublicKey)
	return bytes.Equal(lockingHash, pubKeyHash)
}

func NewCoinbaseTransaction(toAddress string, amount int64) *Transaction {
	txin := TxInput{
		TxID:      []byte{},
		VoutIndex: -1,
		Signature: nil,
		PublicKey: []byte("Reward"),
	}

	txout := TxOutput{Value: amount, PubKeyHash: nil}
	txout.Lock(toAddress)

	tx := Transaction{
		ID:      nil,
		Vin:     []TxInput{txin},
		Vout:    []TxOutput{txout},
		Type:    TxTypeTransfer,
		Payload: nil,
	}
	tx.SetID()
	return &tx
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	for _, vin := range tx.Vin {
		inputs = append(inputs, TxInput{
			TxID:      vin.TxID,
			VoutIndex: vin.VoutIndex,
			Signature: nil,
			PublicKey: nil,
		})
	}

	return Transaction{
		ID:      tx.ID,
		Vin:     inputs,
		Vout:    tx.Vout,
		Type:    tx.Type,
		Payload: tx.Payload,
	}
}

func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTxs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	txCopy := tx.TrimmedCopy()

	for inID, vin := range txCopy.Vin {
		prevTx := prevTxs[string(vin.TxID)]
		if len(prevTx.ID) == 0 {
			Handle(fmt.Errorf("ERROR: Referenced transaction not found: %x", vin.TxID))
			return
		}

		prevOut := prevTx.Vout[vin.VoutIndex]

		txCopy.Vin[inID].PublicKey = prevOut.PubKeyHash

		dataToSign := txCopy.Hash()

		txCopy.Vin[inID].PublicKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, dataToSign)
		Handle(err)

		rBytes := r.Bytes()
		sBytes := s.Bytes()
		sig := make([]byte, 64)
		copy(sig[32-len(rBytes):], rBytes)
		copy(sig[64-len(sBytes):], sBytes)

		tx.Vin[inID].Signature = sig

		tx.Vin[inID].PublicKey = append(privKey.PublicKey.X.Bytes(), privKey.PublicKey.Y.Bytes()...)
	}
}

func (tx *Transaction) Verify(prevTxs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		prevTx := prevTxs[string(vin.TxID)]
		if len(prevTx.ID) == 0 {
			log.Printf("Verify ERROR: Referenced transaction not found: %x", vin.TxID)
			return false
		}

		prevOut := prevTx.Vout[vin.VoutIndex]

		txCopy.Vin[inID].PublicKey = prevOut.PubKeyHash
		dataToVerify := txCopy.Hash()
		txCopy.Vin[inID].PublicKey = nil

		if len(vin.Signature) != 64 {
			log.Println("Verify ERROR: Invalid signature length")
			return false
		}
		r, s := big.Int{}, big.Int{}
		r.SetBytes(vin.Signature[:32])
		s.SetBytes(vin.Signature[32:])

		if len(vin.PublicKey) != 64 {
			log.Println("Verify ERROR: Invalid public key length")
			return false
		}
		x, y := big.Int{}, big.Int{}
		x.SetBytes(vin.PublicKey[:32])
		y.SetBytes(vin.PublicKey[32:])

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if !ecdsa.Verify(&rawPubKey, dataToVerify, &r, &s) {
			log.Printf("Verify ERROR: ECDSA verification failed for TX %x", tx.ID)
			return false
		}
	}

	return true
}

func Handle(err error) {
	if err != nil {
		log.Panic(err)
	}
}
