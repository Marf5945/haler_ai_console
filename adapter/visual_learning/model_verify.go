// visual_learning/model_verify.go — 模型完整性驗證（§14.6.6）。
//
// 規範依據：AI_Console_Spec_v4_2.md §14.6.6
//
// 核心規則：
//   - LoadModel() 前必須驗證模型檔 SHA256。
//   - hash 必須來自受信任的 manifest（build-time embedded）。
//   - hash 不符時回傳 ErrModelIntegrityMismatch，拒絕載入。
//   - 不得 fallback 到未驗證模型，不得靜默降級。
//   - 驗證只在 LoadModel() 執行一次，Infer() 不重複驗證。
//
// manifest 格式（model_hashes.json）：
//
//	{
//	  "models": {
//	    "yolo_nano.mlmodelc": "sha256:abcdef1234...",
//	    "yolo_nano.onnx": "sha256:fedcba4321..."
//	  }
//	}
//
// 安全性考量：
//   manifest 使用 //go:embed 嵌入二進位，攻擊者無法同時替換模型和 manifest。
//   如果需要更新模型，必須重新編譯 app（manifest 跟著更新）。
package visual_learning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ──────────────────────────────────────────────
// Manifest 結構
// ──────────────────────────────────────────────

// ModelHashManifest 是受信任的模型 hash 清單。
// 在正式 build 中由 //go:embed 從 app bundle 嵌入。
// 開發/測試環境可透過 NewModelVerifierFromJSON() 手動載入。
type ModelHashManifest struct {
	// Models 是模型檔名 → SHA256 hash 的對照表。
	// hash 格式："sha256:<hex string>"
	Models map[string]string `json:"models"`
}

// ──────────────────────────────────────────────
// ModelVerifier
// ──────────────────────────────────────────────

// ModelVerifier 負責驗證模型檔完整性。
// 每個 InferenceEngine 實作在 LoadModel() 內部呼叫此驗證器。
type ModelVerifier struct {
	manifest ModelHashManifest
}

// NewModelVerifier 從嵌入的 JSON manifest 建立驗證器。
// manifestJSON 應來自 //go:embed model_hashes.json。
func NewModelVerifier(manifestJSON []byte) (*ModelVerifier, error) {
	var m ModelHashManifest
	if err := json.Unmarshal(manifestJSON, &m); err != nil {
		return nil, fmt.Errorf("model verifier: failed to parse manifest — %w", err)
	}
	if m.Models == nil {
		return nil, fmt.Errorf("model verifier: manifest contains no models")
	}
	return &ModelVerifier{manifest: m}, nil
}

// Verify 驗證指定路徑的模型檔 SHA256 是否與 manifest 一致。
//
// modelPath: 模型檔的完整路徑（例如 "assets/models/yolo_nano.onnx"）
//
// 驗證流程：
//  1. 從路徑取得檔名
//  2. 在 manifest 中查找對應的預期 hash
//  3. 計算檔案的 SHA256
//  4. 比對 — 不符時回傳 ErrModelIntegrityMismatch
//
// 對於 .mlmodelc 目錄（CoreML 編譯模型），驗證目錄內所有檔案的聯合 hash。
func (v *ModelVerifier) Verify(modelPath string) error {
	fileName := filepath.Base(modelPath)

	expectedHash, ok := v.manifest.Models[fileName]
	if !ok {
		return fmt.Errorf("model verifier: model %q not found in trusted manifest", fileName)
	}

	// 解析 "sha256:<hex>" 格式
	if !strings.HasPrefix(expectedHash, "sha256:") {
		return fmt.Errorf("model verifier: invalid hash format for %q — expected 'sha256:<hex>'", fileName)
	}
	expectedHex := strings.TrimPrefix(expectedHash, "sha256:")

	// 計算實際 hash
	actualHex, err := computeModelHash(modelPath)
	if err != nil {
		return fmt.Errorf("model verifier: failed to hash model %q — %w", fileName, err)
	}

	// 比對（constant-time 比較可用於防止 timing attack，但此處非加密場景）
	if actualHex != expectedHex {
		return fmt.Errorf("%w: model %q — expected sha256:%s, got sha256:%s",
			ErrModelIntegrityMismatch, fileName, expectedHex, actualHex)
	}

	return nil
}

// ──────────────────────────────────────────────
// Hash 計算
// ──────────────────────────────────────────────

// computeModelHash 計算模型檔的 SHA256 hash。
// 如果 path 是目錄（.mlmodelc），則遍歷所有檔案並聯合 hash。
// 如果 path 是單一檔案（.onnx），則直接計算該檔案的 hash。
func computeModelHash(modelPath string) (string, error) {
	info, err := os.Stat(modelPath)
	if err != nil {
		return "", err
	}

	h := sha256.New()

	if info.IsDir() {
		// .mlmodelc 是目錄結構，遍歷所有檔案
		err = filepath.Walk(modelPath, func(path string, fi os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if fi.IsDir() {
				return nil
			}
			// 將相對路徑加入 hash（確保檔名順序一致影響 hash）
			relPath, _ := filepath.Rel(modelPath, path)
			h.Write([]byte(relPath))

			f, ferr := os.Open(path)
			if ferr != nil {
				return ferr
			}
			defer f.Close()
			if _, cerr := io.Copy(h, f); cerr != nil {
				return cerr
			}
			return nil
		})
		if err != nil {
			return "", err
		}
	} else {
		// 單一檔案（.onnx）
		f, ferr := os.Open(modelPath)
		if ferr != nil {
			return "", ferr
		}
		defer f.Close()
		if _, cerr := io.Copy(h, f); cerr != nil {
			return "", cerr
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
