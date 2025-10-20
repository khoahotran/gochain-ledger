package domain

import (
	"bytes"
	"crypto/sha256"
	"math"
	"math/big"
)

// Difficulty: Độ khó của Proof-of-Work.
// Chúng ta sẽ yêu cầu 16 bit 0 đứng đầu hash (con số này có thể tùy chỉnh)
const Difficulty = 16

// ProofOfWork đại diện cho cơ chế PoW
type ProofOfWork struct {
	Block  *Block
	Target *big.Int // Hash mục tiêu mà chúng ta cần tìm (phải nhỏ hơn)
}

// NewProofOfWork tạo một PoW mới cho một block
func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	// Dịch trái target 1 lượng: 256 (số bit SHA256) - Difficulty
	target.Lsh(target, uint(256-Difficulty))

	pow := &ProofOfWork{Block: b, Target: target}
	return pow
}

// prepareData chuẩn bị dữ liệu để hash (giống CalculateHash cũ)
// Chúng ta cần một hàm xác định (deterministic) để chuẩn bị data
func (pow *ProofOfWork) prepareData(nonce int64) []byte {
	data := bytes.Join(
		[][]byte{
			pow.Block.PrevBlockHash,
			pow.Block.HashTransactions(), // Hash của tất cả transaction
			IntToHex(pow.Block.Timestamp),
			IntToHex(int64(Difficulty)), // Thêm cả difficulty vào
			IntToHex(nonce),             // Nonce thay đổi liên tục
		},
		[]byte{},
	)
	return data
}

// Run bắt đầu quá trình đào (mining)
// Tìm một nonce sao cho hash(data + nonce) < Target
func (pow *ProofOfWork) Run() (int64, []byte) {
	var hashInt big.Int
	var hash [32]byte
	var nonce int64 = 0

	// Vòng lặp vô tận cho đến khi tìm được hash hợp lệ
	for nonce < math.MaxInt64 {
		data := pow.prepareData(nonce)
		hash = sha256.Sum256(data)
		
		// Chuyển hash (array bytes) sang big.Int để so sánh
		hashInt.SetBytes(hash[:])

		// So sánh hashInt với Target
		// Nếu hashInt < Target (Cmp trả về -1) -> Tìm thấy!
		if hashInt.Cmp(pow.Target) == -1 {
			break // Thoát vòng lặp
		} else {
			nonce++ // Thử nonce tiếp theo
		}
	}
	return nonce, hash[:]
}

// Validate xác thực PoW của một block
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int
	data := pow.prepareData(pow.Block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	// Block hợp lệ nếu hash < Target
	return hashInt.Cmp(pow.Target) == -1
}