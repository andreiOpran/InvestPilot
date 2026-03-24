package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

var EncryptionKey = os.Getenv("AES_MASTER_KEY")

// encrypt plain string into hex-encoded string useing AES-GCM
func EncryptAES(plainText string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := aesGCM.Seal(nonce, nonce, []byte(plainText), nil)
	return hex.EncodeToString(cipherText), nil
}

// decrypt a hex-encoded cipher string back to plain text using AES-GCM
func DecryptAES(encryptedText string, key []byte) (string, error) {
	cipherText, err := hex.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(cipherText) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherBytes := cipherText[:nonceSize], cipherText[nonceSize:]
	plainText, err := aesGCM.Open(nil, nonce, cipherBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}
