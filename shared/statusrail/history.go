package statusrail

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type HistoryStore struct {
	path string
	key  [32]byte
}

func NewHistoryStore(root string) *HistoryStore {
	host, _ := os.Hostname()
	configDir, _ := os.UserConfigDir()
	key := sha256.Sum256([]byte("ai-console-statusrail-v1.8.1:" + host + ":" + configDir))
	return &HistoryStore{
		path: filepath.Join(root, "data", "status_rail", "status_rail_history.md.enc"),
		key:  key,
	}
}

func (s *HistoryStore) LoadPhrases(fallback []string) []string {
	payload, err := s.readPlain()
	if err != nil || strings.TrimSpace(payload) == "" {
		_ = s.SavePhrases(fallback)
		return append([]string(nil), fallback...)
	}
	var phrases []string
	cutoff := time.Now().Add(-8 * time.Hour)
	for _, line := range strings.Split(payload, "\n") {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		createdAt, err := time.Parse(time.RFC3339, parts[0])
		if err != nil || createdAt.Before(cutoff) {
			continue
		}
		if phrase := strings.TrimSpace(parts[1]); phrase != "" {
			phrases = append(phrases, phrase)
		}
	}
	if len(phrases) == 0 {
		return append([]string(nil), fallback...)
	}
	return phrases
}

func (s *HistoryStore) SavePhrases(phrases []string) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	lines := make([]string, 0, len(phrases))
	for _, phrase := range phrases {
		phrase = strings.TrimSpace(strings.ReplaceAll(phrase, "\n", " "))
		if phrase != "" {
			lines = append(lines, now+"|"+phrase)
		}
	}
	return s.writePlain(strings.Join(lines, "\n"))
}

func (s *HistoryStore) readPlain() (string, error) {
	encoded, err := os.ReadFile(s.path)
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		return "", err
	}
	if len(ciphertext) < 12 {
		return "", errors.New("status rail history nonce missing")
	}
	block, err := aes.NewCipher(s.key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce, data := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s *HistoryStore) writePlain(payload string) error {
	block, err := aes.NewCipher(s.key[:])
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	ciphertext := gcm.Seal(append([]byte(nil), nonce...), nonce, []byte(payload), nil)
	return os.WriteFile(s.path, []byte(base64.StdEncoding.EncodeToString(ciphertext)), 0o600)
}
