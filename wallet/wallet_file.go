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
	walletDir    = "wallets"
	scryptN      = 16384
	scryptR      = 8
	scryptP      = 1
	scryptKeyLen = 32
)

type WalletFile struct {
	Address      string `json:"address"`
	PublicKey    []byte `json:"public_key"`
	EncryptedKey []byte `json:"encrypted_key"`
	Salt         []byte `json:"salt"`
}

func EncryptKey(privKey *ecdsa.PrivateKey, password string) ([]byte, []byte, error) {

	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, nil, err
	}

	aesKey, err := scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		return nil, nil, err
	}

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

	plaintext := privKey.D.Bytes()
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, salt, nil
}

func (wf *WalletFile) decryptKey(password string) (*ecdsa.PrivateKey, error) {

	aesKey, err := scrypt.Key([]byte(password), wf.Salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		return nil, err
	}

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

		return nil, fmt.Errorf("giải mã thất bại (sai mật khẩu?)")
	}

	privKey := new(ecdsa.PrivateKey)
	privKey.D = new(big.Int).SetBytes(plaintext)
	privKey.PublicKey.Curve = elliptic.P256()
	privKey.PublicKey.X, privKey.PublicKey.Y = privKey.PublicKey.Curve.ScalarBaseMult(plaintext)

	return privKey, nil
}

func (wf *WalletFile) Save() error {

	if err := os.MkdirAll(walletDir, 0755); err != nil {
		return err
	}

	filename := filepath.Join(walletDir, fmt.Sprintf("%s.json", wf.Address))

	data, err := json.MarshalIndent(wf, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func Load(address string) (*WalletFile, error) {
	filename := filepath.Join(walletDir, fmt.Sprintf("%s.json", address))

	data, err := os.ReadFile(filename)
	if err != nil {

		return nil, fmt.Errorf("ví %s không tìm thấy", address)
	}

	var wf WalletFile
	err = json.Unmarshal(data, &wf)
	if err != nil {
		return nil, err
	}
	return &wf, nil
}

func LoadAndDecrypt(address, password string) (*domain.Wallet, error) {
	wf, err := Load(address)
	if err != nil {
		return nil, err
	}

	privKey, err := wf.decryptKey(password)
	if err != nil {
		return nil, err
	}

	return &domain.Wallet{
		PrivateKey: *privKey,
		PublicKey:  wf.PublicKey,
	}, nil
}
