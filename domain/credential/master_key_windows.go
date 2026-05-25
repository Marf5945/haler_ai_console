//go:build windows

package credential

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

type osMasterKeyProvider struct {
	path string
}

func NewOSMasterKeyProvider(baseDir string, _ string, _ string) MasterKeyProvider {
	return osMasterKeyProvider{path: filepath.Join(baseDir, "data", "secrets", "master_key.dpapi")}
}

func (p osMasterKeyProvider) ProviderID() string {
	return "windows_dpapi"
}

func (p osMasterKeyProvider) LoadOrCreateKey() ([32]byte, error) {
	if protected, err := os.ReadFile(p.path); err == nil && len(protected) > 0 {
		return dpapiUnprotect(protected)
	}
	key, err := randomMasterKey()
	if err != nil {
		return [32]byte{}, err
	}
	protected, err := dpapiProtect(key[:])
	if err != nil {
		return [32]byte{}, err
	}
	if err := os.MkdirAll(filepath.Dir(p.path), 0700); err != nil {
		return [32]byte{}, err
	}
	if err := os.WriteFile(p.path, []byte(base64.StdEncoding.EncodeToString(protected)), 0600); err != nil {
		return [32]byte{}, err
	}
	return key, nil
}

type dataBlob struct {
	cbData uint32
	pbData *byte
}

var (
	crypt32              = syscall.NewLazyDLL("Crypt32.dll")
	kernel32             = syscall.NewLazyDLL("Kernel32.dll")
	procCryptProtectData = crypt32.NewProc("CryptProtectData")
	procUnprotectData    = crypt32.NewProc("CryptUnprotectData")
	procLocalFree        = kernel32.NewProc("LocalFree")
)

func dpapiProtect(data []byte) ([]byte, error) {
	in := blobFromBytes(data)
	var out dataBlob
	r, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&out)),
	)
	if r == 0 {
		return nil, err
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return blobToBytes(out), nil
}

func dpapiUnprotect(encoded []byte) ([32]byte, error) {
	protected, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		return [32]byte{}, err
	}
	in := blobFromBytes(protected)
	var out dataBlob
	r, _, err := procUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&out)),
	)
	if r == 0 {
		return [32]byte{}, err
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	plain := blobToBytes(out)
	if len(plain) != 32 {
		return [32]byte{}, fmt.Errorf("master key length=%d want 32", len(plain))
	}
	var key [32]byte
	copy(key[:], plain)
	return key, nil
}

func blobFromBytes(data []byte) dataBlob {
	if len(data) == 0 {
		return dataBlob{}
	}
	return dataBlob{cbData: uint32(len(data)), pbData: &data[0]}
}

func blobToBytes(blob dataBlob) []byte {
	if blob.pbData == nil || blob.cbData == 0 {
		return nil
	}
	view := unsafe.Slice(blob.pbData, blob.cbData)
	out := make([]byte, len(view))
	copy(out, view)
	return out
}
