package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"time"
)

type Block struct {
	Timestamp     int64
	PrevBlockHash []byte
	Hash          []byte
	Transactions  []*Transaction // Dữ liệu chính là danh sách giao dịch
	Nonce         int64          // Số Nonce dùng cho Proof-of-Work
}

// CalculateHash tính hash của block
func (b *Block) CalculateHash() []byte {
	// Chuẩn bị dữ liệu để hash
	// Chúng ta không hash chính trường Hash
	data := bytes.Join(
		[][]byte{
			b.PrevBlockHash,
			b.HashTransactions(), // Hash của tất cả transaction
			IntToHex(b.Timestamp),
			IntToHex(b.Nonce),
		},
		[]byte{},
	)

	hash := sha256.Sum256(data)
	return hash[:]
}

// HashTransactions tạo một hash duy nhất đại diện cho tất cả transaction trong block
// Đây là một phiên bản đơn giản của Merkle Root
func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte
	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID)
	}
	// Nối tất cả các ID giao dịch lại và hash chúng
	txHash := sha256.Sum256(bytes.Join(txHashes, []byte{}))
	return txHash[:]
}

// NewBlock giờ sẽ chạy PoW để tìm Hash và Nonce
func NewBlock(prevBlockHash []byte, transactions []*Transaction) *Block {
	block := &Block{
		Timestamp:     time.Now().Unix(),
		PrevBlockHash: prevBlockHash,
		Hash:          []byte{},
		Transactions:  transactions,
		Nonce:         0, // Nonce ban đầu
	}

	// Tạo PoW object
	pow := NewProofOfWork(block)

	// Chạy PoW để tìm nonce và hash
	nonce, hash := pow.Run()

	// Gán kết quả cho block
	block.Nonce = nonce
	block.Hash = hash

	log.Printf("Đã đào được block mới! Hash: %x\n", hash)
	return block
}

// NewGenesisBlock tạo block đầu tiên của chuỗi
func NewGenesisBlock(coinbaseTx *Transaction) *Block {
	// Block Genesis cũng phải được đào
	return NewBlock([]byte{}, []*Transaction{coinbaseTx})
}

// Serialize chuyển Block thành []byte
func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}
	return result.Bytes()
}

// DeserializeBlock chuyển []byte thành Block
func DeserializeBlock(data []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}
	return &block
}

// Helper function (có thể chuyển ra file utils)
func IntToHex(n int64) []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(n)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}
