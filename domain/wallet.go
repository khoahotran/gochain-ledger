package domain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"log"

	"github.com/mr-tron/base58"
	"golang.org/x/crypto/ripemd160"
	// Import mới
)

// Giữ nguyên const và struct Wallet
const (
	version            = byte(0x00) // Dùng để thêm vào đầu Public Key Hash
	addressChecksumLen = 4          // 4 bytes cho checksum
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func NewWallet() *Wallet {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	publicKey := append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...)
	return &Wallet{PrivateKey: *privateKey, PublicKey: publicKey}
}

func (w *Wallet) GetAddress() string {
	// Bước 1 & 2: Hash Public Key (SHA256 -> RIPEMD160)
	pubKeyHash := HashPubKey(w.PublicKey)

	// Bước 3: Thêm version
	versionedPayload := append([]byte{version}, pubKeyHash...)

	// Bước 4: Tạo Checksum (Hash 2 lần)
	checksum := checksum(versionedPayload)

	// Bước 5: Nối payload và checksum
	fullPayload := append(versionedPayload, checksum...)

	// Bước 6: Mã hóa Base58
	address := base58.Encode(fullPayload)
	return address
}

// HashPubKey là hàm helper để hash public key
func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)

	hasher := ripemd160.New()
	_, err := hasher.Write(publicSHA256[:])
	if err != nil {
		log.Panic(err)
	}
	publicRIPEMD160 := hasher.Sum(nil)
	return publicRIPEMD160
}

// checksum tạo checksum 4 bytes
func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])
	return secondSHA[:addressChecksumLen]
}

// ValidateAddress (Hàm helper để kiểm tra địa chỉ)
func ValidateAddress(address string) bool {
	fullPayload, err := base58.Decode(address)
	if err != nil {
		return false
	}
	actualChecksum := fullPayload[len(fullPayload)-addressChecksumLen:]
	versionedPayload := fullPayload[:len(fullPayload)-addressChecksumLen]
	targetChecksum := checksum(versionedPayload)
	return bytes.Equal(actualChecksum, targetChecksum)
}

// DecodeAddress (Lấy lại PubKeyHash từ địa chỉ Base58)
func DecodeAddress(address string) []byte {
	fullPayload, err := base58.Decode(address)
	if err != nil {
		log.Panic(err)
	}
	// Lấy PubKeyHash (bỏ byte version và 4 bytes checksum)
	pubKeyHash := fullPayload[1 : len(fullPayload)-addressChecksumLen]
	return pubKeyHash
}
