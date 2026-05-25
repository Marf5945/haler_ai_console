//go:build darwin

package credential

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
)

type osMasterKeyProvider struct {
	service string
	account string
}

func NewOSMasterKeyProvider(_ string, service string, account string) MasterKeyProvider {
	return osMasterKeyProvider{service: service, account: account}
}

func (p osMasterKeyProvider) ProviderID() string {
	return "macos_keychain"
}

func (p osMasterKeyProvider) LoadOrCreateKey() ([32]byte, error) {
	if key, err := p.find(); err == nil {
		return key, nil
	}
	key, err := randomMasterKey()
	if err != nil {
		return [32]byte{}, err
	}
	if err := p.add(key); err != nil {
		return [32]byte{}, err
	}
	return key, nil
}

func (p osMasterKeyProvider) find() ([32]byte, error) {
	cmd := exec.Command("security", "find-generic-password", "-s", p.service, "-a", p.account, "-w")
	out, err := cmd.Output()
	if err != nil {
		return [32]byte{}, err
	}
	return decodeMasterKey(out)
}

func (p osMasterKeyProvider) add(key [32]byte) error {
	encoded := base64.StdEncoding.EncodeToString(key[:])
	cmd := exec.Command("security", "add-generic-password", "-s", p.service, "-a", p.account, "-w", encoded, "-U")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func decodeMasterKey(raw []byte) ([32]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return [32]byte{}, err
	}
	if len(decoded) != 32 {
		return [32]byte{}, fmt.Errorf("master key length=%d want 32", len(decoded))
	}
	var key [32]byte
	copy(key[:], decoded)
	return key, nil
}
