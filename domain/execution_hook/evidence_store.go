package execution_hook

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// EvidenceStore handles AES-256-GCM encrypted storage for hook evidence.
// The encryption key is derived from a device-local secret and never leaves the device.
type EvidenceStore struct {
	keyHex string // 64-char hex string = 32-byte AES-256 key
}

// NewEvidenceStore creates a store using the given hex-encoded 256-bit key.
// In production, this key is loaded from a device-local secret file or OS keychain.
func NewEvidenceStore(keyHex string) (*EvidenceStore, error) {
	if len(keyHex) != 64 {
		return nil, errors.New("evidence store: key must be 64 hex characters (256 bits)")
	}
	if _, err := hex.DecodeString(keyHex); err != nil {
		return nil, errors.New("evidence store: invalid hex key")
	}
	return &EvidenceStore{keyHex: keyHex}, nil
}

// NewEvidenceStoreFromFile loads or creates a device-local secret key file.
// The key file is stored at <hookDir>/device_secret.hex with permissions 0o600.
func NewEvidenceStoreFromFile(hookDir string) (*EvidenceStore, error) {
	if err := os.MkdirAll(hookDir, 0o700); err != nil {
		return nil, err
	}
	keyPath := filepath.Join(hookDir, "device_secret.hex")

	data, err := os.ReadFile(keyPath)
	if err == nil && len(data) == 64 {
		return NewEvidenceStore(string(data))
	}

	// Generate a fresh 256-bit key.
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	keyHex := hex.EncodeToString(key)
	if err := os.WriteFile(keyPath, []byte(keyHex), 0o600); err != nil {
		return nil, err
	}
	return NewEvidenceStore(keyHex)
}

// Encrypt encrypts plaintext with AES-256-GCM. Returns nonce+ciphertext.
func (e *EvidenceStore) Encrypt(plaintext []byte) ([]byte, error) {
	key, _ := hex.DecodeString(e.keyHex)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts nonce+ciphertext produced by Encrypt.
func (e *EvidenceStore) Decrypt(data []byte) ([]byte, error) {
	key, _ := hex.DecodeString(e.keyHex)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("evidence store: ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// WriteEncrypted encrypts data and writes it to path (creates dirs as needed).
func (e *EvidenceStore) WriteEncrypted(path string, plaintext []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	encrypted, err := e.Encrypt(plaintext)
	if err != nil {
		return err
	}
	return os.WriteFile(path, encrypted, 0o600)
}

// ReadDecrypted reads and decrypts data from path.
func (e *EvidenceStore) ReadDecrypted(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return e.Decrypt(data)
}
