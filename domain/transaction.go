package domain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"math/big"
)

// TxType đại diện cho loại giao dịch
type TxType int

const (
	TxTypeTransfer       TxType = iota // 0: Giao dịch chuyển tiền (mặc định)
	TxTypeContractDeploy TxType = 1    // 1: Triển khai Smart Contract
	TxTypeContractCall   TxType = 2    // 2: Gọi hàm Smart Contract
)

// TxInput đại diện cho một đầu vào của giao dịch
type TxInput struct {
	TxID      []byte // ID của giao dịch chứa Output này
	VoutIndex int    // Vị trí (index) của Output trong giao dịch đó
	Signature []byte // Chữ ký (để mở khóa Output)
	PublicKey []byte // Public key (để xác thực chữ ký)
}

// TxOutput đại diện cho một đầu ra của giao dịch
type TxOutput struct {
	Value      int64  // Số tiền
	PubKeyHash []byte // Hash của Public Key (địa chỉ) khóa số tiền này
}

// Transaction (Cấu trúc mới)
type Transaction struct {
	ID   []byte
	Vin  []TxInput  // Danh sách các đầu vào
	Vout []TxOutput // Danh sách các đầu ra

	// TRƯỜNG MỚI
	Type    TxType `json:"type"`    // Loại giao dịch
	Payload []byte `json:"payload"` // Dữ liệu đính kèm (code hoặc args)
}

// SetID tính và gán ID cho transaction
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

// IsCoinbase kiểm tra đây có phải giao dịch coinbase (phần thưởng)
func (tx *Transaction) IsCoinbase() bool {
	// Coinbase tx chỉ có 1 input, và input đó không tham chiếu (TxID rỗng)
	return len(tx.Vin) == 1 && len(tx.Vin[0].TxID) == 0
}

// Lock "khóa" một Output bằng địa chỉ
func (out *TxOutput) Lock(address string) {
	pubKeyHash := DecodeAddress(address) // Dùng hàm mới từ wallet.go
	out.PubKeyHash = pubKeyHash
}

// IsLockedWithKey kiểm tra xem Output này có bị khóa bởi PubKeyHash này không
func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}

// CanBeUnlockedWith kiểm tra xem Input này có thể mở khóa Output hay không
func (in *TxInput) CanBeUnlockedWith(pubKeyHash []byte) bool {
	lockingHash := HashPubKey(in.PublicKey)
	return bytes.Equal(lockingHash, pubKeyHash)
}

// NewCoinbaseTransaction (Cập nhật)
func NewCoinbaseTransaction(toAddress string, amount int64) *Transaction {
	// Input của Coinbase tx
	txin := TxInput{
		TxID:      []byte{}, // Không tham chiếu tx cũ
		VoutIndex: -1,
		Signature: nil,
		PublicKey: []byte("Reward"), // Dữ liệu tùy ý
	}

	// Output (phần thưởng)
	txout := TxOutput{Value: amount, PubKeyHash: nil}
	txout.Lock(toAddress) // Khóa bằng địa chỉ của thợ đào

	tx := Transaction{
		ID:      nil,
		Vin:     []TxInput{txin},
		Vout:    []TxOutput{txout},
		Type:    TxTypeTransfer, // Coinbase vẫn là 1 dạng Transfer
		Payload: nil,
	}
	tx.SetID()
	return &tx
}

// === Logic Sign & Verify (Phức tạp hơn) ===

// TrimmedCopy tạo một bản sao "cắt gọt" của transaction để ký
// Khi ký, chúng ta cần đảm bảo Signature và PublicKey trong Input phải rỗng
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	for _, vin := range tx.Vin {
		inputs = append(inputs, TxInput{
			TxID:      vin.TxID,
			VoutIndex: vin.VoutIndex,
			Signature: nil, // Quan trọng
			PublicKey: nil, // Quan trọng
		})
	}

	outputs := tx.Vout // Outputs không đổi

	return Transaction{ID: tx.ID, Vin: inputs, Vout: outputs}
}

// Sign ký vào transaction
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTxs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	// Tạo bản sao để ký
	txCopy := tx.TrimmedCopy()

	// Duyệt qua từng input và ký
	for inID, vin := range txCopy.Vin {
		prevTx := prevTxs[string(vin.TxID)] // Lấy transaction chứa output cũ
		if prevTx.ID == nil {
			log.Panic("LỖI: Không tìm thấy transaction tham chiếu")
		}

		// Lấy output cũ mà input này tham chiếu
		prevOut := prevTx.Vout[vin.VoutIndex]

		// Gán PubKeyHash của output cũ vào PublicKey (để hash)
		// Đây là phần dữ liệu sẽ được ký
		txCopy.Vin[inID].PublicKey = prevOut.PubKeyHash

		// Hash bản sao transaction
		txCopy.SetID() // Dùng ID làm dữ liệu để hash
		dataToSign := txCopy.ID

		// Ký
		r, s, err := ecdsa.Sign(rand.Reader, &privKey, dataToSign)
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		// Gán chữ ký và PublicKey thật vào transaction gốc (không phải bản copy)
		tx.Vin[inID].Signature = signature
		// Lấy full public key (X và Y) từ private key
		fullPublicKey := append(privKey.PublicKey.X.Bytes(), privKey.PublicKey.Y.Bytes()...)
		tx.Vin[inID].PublicKey = fullPublicKey
		// Xóa PubKeyHash trong bản copy để chuẩn bị cho vòng lặp input tiếp theo
		txCopy.Vin[inID].PublicKey = nil
	}
}

// Verify xác thực giao dịch
func (tx *Transaction) Verify(prevTxs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		prevTx := prevTxs[string(vin.TxID)]
		if prevTx.ID == nil {
			log.Panic("LỖI: Không tìm thấy transaction tham chiếu")
		}

		prevOut := prevTx.Vout[vin.VoutIndex]

		// Chuẩn bị dữ liệu giống hệt như lúc ký
		txCopy.Vin[inID].PublicKey = prevOut.PubKeyHash
		txCopy.SetID()
		dataToVerify := txCopy.ID

		// Tách chữ ký
		sigLen := len(vin.Signature)
		r, s := big.Int{}, big.Int{}
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		// Tách public key
		keyLen := len(vin.PublicKey)
		x, y := big.Int{}, big.Int{}
		x.SetBytes(vin.PublicKey[:(keyLen / 2)])
		y.SetBytes(vin.PublicKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if !ecdsa.Verify(&rawPubKey, dataToVerify, &r, &s) {
			return false // Xác thực thất bại
		}

		txCopy.Vin[inID].PublicKey = nil
	}

	return true
}
