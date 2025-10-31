package domain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"log"

	"github.com/mr-tron/base58"
)

const (
	version            = byte(0x00)
	addressChecksumLen = 4
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

	pubKeyHash := HashPubKey(w.PublicKey)

	versionedPayload := append([]byte{version}, pubKeyHash...)

	checksum := checksum(versionedPayload)

	fullPayload := append(versionedPayload, checksum...)

	address := base58.Encode(fullPayload)
	return address
}

func HashPubKey(pubKey []byte) []byte {

	publicSHA256 := sha256.Sum256(pubKey)

	publicSHA256_2 := sha256.Sum256(publicSHA256[:])

	return publicSHA256_2[:]
}

func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])
	return secondSHA[:addressChecksumLen]
}

func ValidateAddress(address string) bool {
	fullPayload, err := base58.Decode(address)
	if err != nil {
		return false
	}
	if len(fullPayload) < addressChecksumLen+1 {
		return false
	}
	actualChecksum := fullPayload[len(fullPayload)-addressChecksumLen:]
	versionedPayload := fullPayload[:len(fullPayload)-addressChecksumLen]
	targetChecksum := checksum(versionedPayload)
	return bytes.Equal(actualChecksum, targetChecksum)
}

func DecodeAddress(address string) []byte {
	fullPayload, err := base58.Decode(address)
	if err != nil {
		log.Panic(err)
	}

	pubKeyHash := fullPayload[1 : len(fullPayload)-addressChecksumLen]
	return pubKeyHash
}
