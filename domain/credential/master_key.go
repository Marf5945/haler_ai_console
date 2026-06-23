package credential

import (
	"crypto/rand"
	"errors"
	"io"
)

var (
	ErrCredentialProviderUnavailable = errors.New("credential provider unavailable")
	ErrCredentialMigrationRequired   = errors.New("credential migration requires user confirmation")
	ErrCredentialMigrationDisabled   = errors.New("credential migration disabled; credentials must be reset")
)

// MasterKeyProvider owns the OS-specific storage for the AES master key.
// Store never falls back to the legacy derived key outside explicit migration.
type MasterKeyProvider interface {
	LoadOrCreateKey() ([32]byte, error)
	ProviderID() string
}

func randomMasterKey() ([32]byte, error) {
	var key [32]byte
	_, err := io.ReadFull(rand.Reader, key[:])
	return key, err
}

type staticMasterKeyProvider struct {
	id  string
	key [32]byte
	err error
}

// NewStaticMasterKeyProvider is used by tests and tightly controlled tooling.
func NewStaticMasterKeyProvider(id string, key [32]byte) MasterKeyProvider {
	return staticMasterKeyProvider{id: id, key: key}
}

func (p staticMasterKeyProvider) LoadOrCreateKey() ([32]byte, error) {
	if p.err != nil {
		return [32]byte{}, p.err
	}
	return p.key, nil
}

func (p staticMasterKeyProvider) ProviderID() string {
	if p.id == "" {
		return "static_test"
	}
	return p.id
}

type unavailableMasterKeyProvider struct {
	id  string
	err error
}

func (p unavailableMasterKeyProvider) LoadOrCreateKey() ([32]byte, error) {
	if p.err != nil {
		return [32]byte{}, p.err
	}
	return [32]byte{}, ErrCredentialProviderUnavailable
}

func (p unavailableMasterKeyProvider) ProviderID() string {
	if p.id == "" {
		return "unavailable"
	}
	return p.id
}
