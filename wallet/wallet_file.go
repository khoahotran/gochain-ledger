package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"

	"github.com/khoahotran/gochain-ledger/domain"
	"golang.org/x/crypto/scrypt"
)

const (
	walletDir    = "wallets" // Thư mục lưu ví
	scryptN      = 16384
	scryptR      = 8
	scryptP      = 1
	scryptKeyLen = 32 // 32 bytes = 256-bit key
)

// WalletFile là struct được lưu vào file JSON
// Nó chứa private key ĐÃ ĐƯỢC MÃ HÓA
type WalletFile struct {
	Address      string `json:"address"`
	PublicKey    []byte `json:"public_key"`
	EncryptedKey []byte `json:"encrypted_key"` // Private key đã mã hóa
	Salt         []byte `json:"salt"`          // Salt dùng cho scrypt
}

// --- Logic Mã hóa / Giải mã ---

// encryptKey mã hóa private key bằng mật khẩu
func EncryptKey(privKey *ecdsa.PrivateKey, password string) ([]byte, []byte, error) {
	// 1. Tạo Salt ngẫu nhiên
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, nil, err
	}

	// 2. Tạo khóa AES từ mật khẩu và salt (dùng scrypt)
	aesKey, err := scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		return nil, nil, err
	}

	// 3. Mã hóa private key (bytes)
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	// Dùng private key D (dạng bytes)
	plaintext := privKey.D.Bytes()
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, salt, nil
}

// decryptKey giải mã private key
func (wf *WalletFile) decryptKey(password string) (*ecdsa.PrivateKey, error) {
	// 1. Tạo lại khóa AES từ mật khẩu và salt đã lưu
	aesKey, err := scrypt.Key([]byte(password), wf.Salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		return nil, err
	}

	// 2. Giải mã
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(wf.EncryptedKey) < nonceSize {
		return nil, fmt.Errorf("ciphertext quá ngắn")
	}

	nonce, ciphertext := wf.EncryptedKey[:nonceSize], wf.EncryptedKey[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Lỗi này thường là do sai mật khẩu!
		return nil, fmt.Errorf("giải mã thất bại (sai mật khẩu?)")
	}

	// 3. Tạo lại đối tượng ecdsa.PrivateKey
	privKey := new(ecdsa.PrivateKey)
	privKey.D = new(big.Int).SetBytes(plaintext)
	privKey.PublicKey.Curve = elliptic.P256()
	privKey.PublicKey.X, privKey.PublicKey.Y = privKey.PublicKey.Curve.ScalarBaseMult(plaintext)

	return privKey, nil
}

// --- Logic Lưu / Tải ---

// SaveLưu file ví vào thư mục 'wallets/'
func (wf *WalletFile) Save() error {
	// Đảm bảo thư mục "wallets" tồn tại
	if err := os.MkdirAll(walletDir, 0755); err != nil {
		return err
	}

	// Tên file là ADDR.json
	filename := filepath.Join(walletDir, fmt.Sprintf("%s.json", wf.Address))

	data, err := json.MarshalIndent(wf, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// LoadTải file ví
func Load(address string) (*WalletFile, error) {
	filename := filepath.Join(walletDir, fmt.Sprintf("%s.json", address))

	data, err := os.ReadFile(filename)
	if err != nil {
		// Lỗi này thường là do ví không tồn tại
		return nil, fmt.Errorf("ví %s không tìm thấy", address)
	}

	var wf WalletFile
	err = json.Unmarshal(data, &wf)
	if err != nil {
		return nil, err
	}
	return &wf, nil
}

// LoadAndDecrypt tải ví và giải mã
func LoadAndDecrypt(address, password string) (*domain.Wallet, error) {
	wf, err := Load(address)
	if err != nil {
		return nil, err
	}

	privKey, err := wf.decryptKey(password)
	if err != nil {
		return nil, err
	}

	// Trả về domain.Wallet (dùng cho logic nghiệp vụ)
	return &domain.Wallet{
		PrivateKey: *privKey,
		PublicKey:  wf.PublicKey,
	}, nil
}
