// w3a_media/app_fingerprint.go — §9A.4–§9A.5 App 操作指紋 + 開發者簽章。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ Console 作為完整 W3A-aware app，需要：                      │
// │  1. 產生 Ed25519 金鑰對（本機儲存，不進 export）            │
// │  2. 對每次媒體操作產生 AppOperationFingerprint              │
// │  3. 用開發者金鑰簽署操作指紋                                │
// │  4. 驗證其他 W3A-aware app 的簽章                           │
// │                                                             │
// │ 硬規則：                                                    │
// │  - 未簽名的操作指紋 → 僅作為 low-trust hint（§9A.5）       │
// │  - self-declared app tags 不足以取得高信任權重              │
// │  - 開發者簽章僅證明操作日誌來自宣稱的 app，不證明內容品質  │
// │                                                             │
// │ 使用 Go 標準庫 crypto/ed25519，零外部依賴                   │
// └─────────────────────────────────────────────────────────────┘
package w3a_media

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ──────────────────────────────────────────────
// 金鑰對管理
// ──────────────────────────────────────────────

// AppKeyPair Console 本機 Ed25519 金鑰對。
type AppKeyPair struct {
	AppID      string `json:"app_id"`
	PublicKey  string `json:"public_key"`  // hex 編碼
	PrivateKey string `json:"private_key"` // hex 編碼（僅本機，不進 export）
	CreatedAt  string `json:"created_at"`
}

// KeyManager 管理 Console 的 Ed25519 金鑰對。
type KeyManager struct {
	mu       sync.Mutex
	keypair  *AppKeyPair
	filePath string
}

// NewKeyManager 建立金鑰管理器。
func NewKeyManager(hookRoot string) *KeyManager {
	return &KeyManager{
		filePath: filepath.Join(hookRoot, "w3a_app_keypair.json"),
	}
}

// GetOrCreateKeypair 取得現有金鑰對，若不存在則產生新的。
func (km *KeyManager) GetOrCreateKeypair() (*AppKeyPair, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	// 嘗試從檔案載入
	if km.keypair == nil {
		if data, err := os.ReadFile(km.filePath); err == nil {
			var kp AppKeyPair
			if json.Unmarshal(data, &kp) == nil {
				km.keypair = &kp
			}
		}
	}

	// 已存在則回傳
	if km.keypair != nil {
		return km.keypair, nil
	}

	// 產生新金鑰對
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}

	kp := &AppKeyPair{
		AppID:      "ai_console",
		PublicKey:  hex.EncodeToString(pub),
		PrivateKey: hex.EncodeToString(priv),
		CreatedAt:  time.Now().Format(time.RFC3339),
	}

	// 儲存（permission 0600，僅擁有者可讀寫）
	data, _ := json.MarshalIndent(kp, "", "  ")
	if err := os.WriteFile(km.filePath, data, 0600); err != nil {
		return nil, fmt.Errorf("save keypair: %w", err)
	}

	km.keypair = kp
	return kp, nil
}

// GetPublicKey 取得公鑰（hex 編碼）。
func (km *KeyManager) GetPublicKey() (string, error) {
	kp, err := km.GetOrCreateKeypair()
	if err != nil {
		return "", err
	}
	return kp.PublicKey, nil
}

// ──────────────────────────────────────────────
// 操作指紋簽署
// ──────────────────────────────────────────────

// SignOperation 用 Console 的金鑰簽署操作指紋。
func (km *KeyManager) SignOperation(op AppOperationFingerprint) (*DeveloperSignature, error) {
	kp, err := km.GetOrCreateKeypair()
	if err != nil {
		return nil, err
	}

	// 序列化操作指紋為簽署內容
	payload, err := json.Marshal(op)
	if err != nil {
		return nil, fmt.Errorf("marshal operation: %w", err)
	}

	// 解碼私鑰
	privBytes, err := hex.DecodeString(kp.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	privKey := ed25519.PrivateKey(privBytes)

	// Ed25519 簽署
	sig := ed25519.Sign(privKey, payload)

	return &DeveloperSignature{
		AppID:     kp.AppID,
		PublicKey: kp.PublicKey,
		Signature: hex.EncodeToString(sig),
		SignedAt:  time.Now().Format(time.RFC3339),
	}, nil
}

// ──────────────────────────────────────────────
// 簽章驗證
// ──────────────────────────────────────────────

// VerifySignature 驗證操作指紋的開發者簽章是否有效。
func VerifySignature(op AppOperationFingerprint, sig DeveloperSignature) (bool, error) {
	// 序列化操作指紋
	payload, err := json.Marshal(op)
	if err != nil {
		return false, fmt.Errorf("marshal operation: %w", err)
	}

	// 解碼公鑰與簽章
	pubBytes, err := hex.DecodeString(sig.PublicKey)
	if err != nil {
		return false, fmt.Errorf("decode public key: %w", err)
	}
	sigBytes, err := hex.DecodeString(sig.Signature)
	if err != nil {
		return false, fmt.Errorf("decode signature: %w", err)
	}

	pubKey := ed25519.PublicKey(pubBytes)
	return ed25519.Verify(pubKey, payload, sigBytes), nil
}

// ──────────────────────────────────────────────
// 操作記錄器
// ──────────────────────────────────────────────

// OperationRecorder 記錄 Console 內的媒體操作。
type OperationRecorder struct {
	mu         sync.Mutex
	operations []AppOperationFingerprint
}

// NewOperationRecorder 建立操作記錄器。
func NewOperationRecorder() *OperationRecorder {
	return &OperationRecorder{}
}

// Record 記錄一次操作。
func (r *OperationRecorder) Record(op AppOperationFingerprint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.operations = append(r.operations, op)
}

// GetAll 取得所有已記錄的操作。
func (r *OperationRecorder) GetAll() []AppOperationFingerprint {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]AppOperationFingerprint, len(r.operations))
	copy(result, r.operations)
	return result
}

// Clear 清除記錄。
func (r *OperationRecorder) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.operations = nil
}
