package domain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
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

// jsonTxOutput is used for consistent JSON hashing with frontend
type jsonTxOutput struct {
	Value      string `json:"value"` // Serialize int64 as string
	PubKeyHash []byte `json:"pubKeyHash"`
}

// jsonTransaction is used for consistent JSON hashing with frontend
type jsonTransaction struct {
	ID      []byte         `json:"id"` // Still exclude during hash
	Vin     []TxInput      `json:"vin"`
	Vout    []jsonTxOutput `json:"vout"` // Use the custom output type
	Type    TxType         `json:"type"`
	Payload []byte         `json:"payload"`
}

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
			Signature: nil, // Omits Signature
			PublicKey: nil, // Omits PublicKey, crucial for JSON consistency
		})
	}
	outputs := tx.Vout // Outputs don't change

	// Ensure Type and Payload are included
	return Transaction{
		ID:      tx.ID, // Keep ID for structure, though it's ignored in Hash()
		Vin:     inputs,
		Vout:    outputs,
		Type:    tx.Type,
		Payload: tx.Payload,
	}
}

// Modify Sign to prepare data correctly for JSON hashing
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTxs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	txCopy := tx.TrimmedCopy() // Creates copy with nil Sig/PubKey

	for inID, vin := range txCopy.Vin {
		prevTx := prevTxs[string(vin.TxID)]
		if len(prevTx.ID) == 0 { // Check if map lookup failed
			Handle(fmt.Errorf("ERROR: Referenced transaction not found: %x", vin.TxID))
		}
		prevOut := prevTx.Vout[vin.VoutIndex]

		// For JSON hashing consistency with frontend:
		// Temporarily put the PubKeyHash into the PublicKey field of the input *copy*
		// that will be marshaled to JSON.
		txCopy.Vin[inID].PublicKey = prevOut.PubKeyHash

		// Use the updated Hash() method which now uses JSON
		dataToSign := txCopy.Hash()

		// Reset the PublicKey field in the copy *after* hashing
		// so it doesn't interfere with the next input's hash
		txCopy.Vin[inID].PublicKey = nil

		// Sign the hash
		r, s, err := ecdsa.Sign(rand.Reader, &privKey, dataToSign)
		Handle(err)
		signature := append(r.Bytes(), s.Bytes()...)

		// Set Signature and the *actual* PublicKey on the *original* transaction's input
		tx.Vin[inID].Signature = signature
		fullPublicKey := append(privKey.PublicKey.X.Bytes(), privKey.PublicKey.Y.Bytes()...)
		tx.Vin[inID].PublicKey = fullPublicKey
	}
}

// Modify Verify to prepare data correctly for JSON hashing
func (tx *Transaction) Verify(prevTxs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	txCopy := tx.TrimmedCopy() // Creates copy with nil Sig/PubKey
	curve := elliptic.P256()

	for inID, vin := range tx.Vin { // Iterate original tx inputs to get Sig/PubKey
		prevTx := prevTxs[string(vin.TxID)]
		if len(prevTx.ID) == 0 { // Check if map lookup failed
			Handle(fmt.Errorf("ERROR: Referenced transaction not found: %x", vin.TxID))
			return false // Or handle error appropriately
		}
		prevOut := prevTx.Vout[vin.VoutIndex]

		// Prepare the copy exactly like in Sign for hashing
		txCopy.Vin[inID].PublicKey = prevOut.PubKeyHash

		// Use the updated Hash() method (uses JSON)
		dataToVerify := txCopy.Hash()

		// Reset PublicKey in copy *after* hashing
		txCopy.Vin[inID].PublicKey = nil

		// --- Signature and Public Key Reconstruction (Same as before) ---
		if len(vin.Signature) == 0 || len(vin.PublicKey) == 0 {
			return false
		} // Basic check

		sigLen := len(vin.Signature)
		if sigLen != 64 {
			return false
		} // Assuming P256 -> 32 byte R + 32 byte S
		r, s := big.Int{}, big.Int{}
		r.SetBytes(vin.Signature[:32])
		s.SetBytes(vin.Signature[32:])

		keyLen := len(vin.PublicKey)
		// Assuming uncompressed P256 key (1 byte prefix + 32 byte X + 32 byte Y = 65 bytes)
		// OR compressed (33 bytes). Let's assume uncompressed was stored.
		// If only X,Y was stored (64 bytes):
		if keyLen != 64 {
			return false
		} // Adjust if you store compressed keys
		x, y := big.Int{}, big.Int{}
		x.SetBytes(vin.PublicKey[:32])
		y.SetBytes(vin.PublicKey[32:])

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if !ecdsa.Verify(&rawPubKey, dataToVerify, &r, &s) {
			return false // Verification failed
		}
	}
	return true // All inputs verified
}

func (tx *Transaction) Hash() []byte {
	// Tạo bản sao trung gian với kiểu dữ liệu JSON phù hợp
	jsonCopy := jsonTransaction{
		ID:      []byte{}, // Exclude ID
		Vin:     make([]TxInput, len(tx.Vin)),
		Vout:    make([]jsonTxOutput, len(tx.Vout)),
		Type:    tx.Type,
		Payload: tx.Payload,
	}

	// Copy Vin (Signature và PublicKey đã được nil ở TrimmedCopy)
	copy(jsonCopy.Vin, tx.Vin)

	// Copy Vout, chuyển đổi Value sang string
	for i, out := range tx.Vout {
		jsonCopy.Vout[i] = jsonTxOutput{
			Value:      fmt.Sprintf("%d", out.Value), // Convert int64 to string
			PubKeyHash: out.PubKeyHash,
		}
	}

	// Marshal bản sao JSON này
	jsonData, err := json.Marshal(jsonCopy)
	if err != nil {
		Handle(fmt.Errorf("failed to marshal transaction to JSON for hashing: %v", err))
	}

	hash := sha256.Sum256(jsonData)
	return hash[:]
}
