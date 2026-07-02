//go:build darwin || linux

package credential

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// decodeMasterKey decodes the base64-encoded 32-byte key shared by Darwin and Linux.
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
