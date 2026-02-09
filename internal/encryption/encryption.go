package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// KeySize is the size of the AES key in bytes
	KeySize = 32 // AES-256
	// SaltSize is the size of the salt in bytes
	SaltSize = 16
	// Iterations is the number of iterations for PBKDF2
	// 100,000 iterations provides strong brute-force resistance per OWASP guidelines
	Iterations = 100000
	// Prefix is used to identify encrypted tokens
	Prefix = "enc:"
)

var (
	// ErrInvalidEncryptedData is returned when attempting to decrypt invalid data
	ErrInvalidEncryptedData = errors.New("invalid encrypted data")
	// ErrInvalidPassword is returned when the password is invalid
	ErrInvalidPassword = errors.New("invalid password")
	// ErrNotEncrypted is returned when a token is not encrypted
	ErrNotEncrypted = errors.New("token is not encrypted")
)

// IsEncrypted checks if a token is encrypted
func IsEncrypted(token string) bool {
	return strings.HasPrefix(token, Prefix)
}

// DeriveKey derives an encryption key from a password and salt
func DeriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, Iterations, KeySize, sha256.New)
}

// Encrypt encrypts a token with the given password
func Encrypt(token, password string) (string, error) {
	if IsEncrypted(token) {
		return "", errors.New("token is already encrypted")
	}

	// Generate a random salt
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	// Derive key from password and salt
	key := DeriveKey(password, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generate a nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the token
	ciphertext := aesGCM.Seal(nil, nonce, []byte(token), nil)

	// Combine salt, nonce, and ciphertext
	encrypted := make([]byte, 0, len(salt)+len(nonce)+len(ciphertext))
	encrypted = append(encrypted, salt...)
	encrypted = append(encrypted, nonce...)
	encrypted = append(encrypted, ciphertext...)

	// Encode as base64 and add prefix
	return Prefix + base64.StdEncoding.EncodeToString(encrypted), nil
}

// Decrypt decrypts a token with the given password
func Decrypt(encryptedToken, password string) (string, error) {
	if !IsEncrypted(encryptedToken) {
		return "", ErrNotEncrypted
	}

	// Remove prefix
	data := strings.TrimPrefix(encryptedToken, Prefix)

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidEncryptedData, err)
	}

	// Check minimal length requirements
	if len(decoded) < SaltSize+12 {
		return "", ErrInvalidEncryptedData
	}

	// Extract salt, nonce, and ciphertext
	salt := decoded[:SaltSize]
	decoded = decoded[SaltSize:]

	// Derive key from password and salt
	key := DeriveKey(password, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(decoded) < nonceSize {
		return "", ErrInvalidEncryptedData
	}

	nonce := decoded[:nonceSize]
	ciphertext := decoded[nonceSize:]

	// Decrypt
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrInvalidPassword
	}

	return string(plaintext), nil
}
