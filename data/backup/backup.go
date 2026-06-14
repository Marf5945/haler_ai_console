// Package backup — 單一專案備份／還原（跨裝置接續對話的地基）。
//
// 加密「可選」（使用者勾選才加密，避免簡單保存忘記密碼）：
//
//	加密格式：magic(8) "AICBAK01" | salt(16) | nonce(12) | ciphertext(AES-256-GCM)
//	明文格式：magic(8) "AICBAKP1" | gzip(tar)
//
// gzip(tar) 內含：
//   - manifest.json（格式版本、project_id、建立時間、是否遮蔽）
//   - project/<相對路徑>（專案檔案，排除 runtime/ 暫存）
//
// 安全設計：
//   - 加密模式金鑰：scrypt(password, salt, N=2^15, r=8, p=1) → 32 bytes。
//     GCM 認證標籤即「防竄改封條」；AAD 綁定檔頭，換頭也拒開。
//   - 明文模式（password 為空）：無機密保護，僅靠 gzip CRC 偵測損壞；
//     呼叫端 UI 應明示「未加密，請勿放公開位置」。
//   - 匯入時自動辨識格式：IsBundleEncrypted 讓 UI 決定要不要問密碼。
//   - 匯入時每個 tar entry 走 storage.ValidateZipEntry 防路徑穿越，
//     並有單檔/總量上限防解壓炸彈。
//   - redact=true 時，memory/ 下的文字檔先過 memory.RedactBeforeWrite 再打包。
package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/scrypt"

	"ui_console/data/memory"
	"ui_console/data/storage"
)

const (
	magicHeader   = "AICBAK01" // 加密格式
	magicPlain    = "AICBAKP1" // 明文格式（使用者未勾選加密）
	saltLen       = 16
	nonceLen      = 12
	keyLen        = 32
	scryptN       = 1 << 15
	scryptR       = 8
	scryptP       = 1
	FormatVersion = 1

	// MinPasswordLen 密碼下限。
	MinPasswordLen = 8

	// 解壓上限（防 zip bomb）。
	maxEntrySize  = 64 << 20  // 單檔 64MB
	maxTotalSize  = 512 << 20 // 總量 512MB
	maxEntryCount = 20000
)

var (
	// ErrPasswordTooShort 密碼太短。
	ErrPasswordTooShort = fmt.Errorf("密碼至少需要 %d 個字元", MinPasswordLen)
	// ErrBadFormat 不是合法的備份檔。
	ErrBadFormat = errors.New("不是 AI Console 備份檔，或檔案已損壞")
	// ErrDecryptFailed 密碼錯誤或檔案被竄改（GCM 無法區分兩者）。
	ErrDecryptFailed = errors.New("密碼錯誤，或備份檔已被竄改")
	// ErrProjectExists 還原目標已存在。
	ErrProjectExists = errors.New("同名專案已存在")

	validProjectID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// Manifest 備份內容描述（tar 內 manifest.json）。
type Manifest struct {
	FormatVersion int    `json:"format_version"`
	ProjectID     string `json:"project_id"`
	CreatedAt     string `json:"created_at"`
	Encrypted     bool   `json:"encrypted"`
	Redacted      bool   `json:"redacted"`
	FileCount     int    `json:"file_count"`
	RedactionHits int    `json:"redaction_hits"`
}

// ExportResult 匯出結果。
type ExportResult struct {
	BundlePath    string `json:"bundle_path"`
	FileCount     int    `json:"file_count"`
	Encrypted     bool   `json:"encrypted"`
	Redacted      bool   `json:"redacted"`
	RedactionHits int    `json:"redaction_hits"`
	SizeBytes     int64  `json:"size_bytes"`
}

// ImportResult 匯入結果。
type ImportResult struct {
	ProjectID  string `json:"project_id"`
	RestoredAs string `json:"restored_as"`
	FileCount  int    `json:"file_count"`
	Redacted   bool   `json:"redacted"`
	CreatedAt  string `json:"created_at"`
}

// ImportMode 還原模式。
type ImportMode string

const (
	// ModeFailIfExists 同名專案存在時回報衝突（預設）。
	ModeFailIfExists ImportMode = "fail_if_exists"
	// ModeOverwrite 覆蓋既有專案（先還原到暫存區，成功才替換）。
	ModeOverwrite ImportMode = "overwrite"
	// ModeCopy 以「<id>-restored-<時間戳>」另存新專案。
	ModeCopy ImportMode = "copy"
)

// 匯出時整個跳過的頂層目錄（執行時暫存，不屬於可攜資料）。
var skipDirs = map[string]bool{
	"runtime": true,
}

// ──────────────────────────────────────────────
// 匯出
// ──────────────────────────────────────────────

// ExportProject 將 baseDir 下的指定專案打包成備份檔，寫入 destPath。
// password 非空 → 加密（至少 8 字元）；password 為空 → 明文備份，
// 對應 UI「加密備份檔」勾選框未勾的情況。
// redact=true 時對 memory/ 下的 .md/.txt 內容先做機密遮蔽。
func ExportProject(baseDir, projectID, destPath, password string, redact bool) (ExportResult, error) {
	encrypt := password != ""
	if encrypt {
		if err := validatePassword(password); err != nil {
			return ExportResult{}, err
		}
	}
	if !validProjectID.MatchString(projectID) {
		return ExportResult{}, fmt.Errorf("專案 ID 不合法: %q", projectID)
	}
	projectRoot := storage.ProjectRoot(baseDir, projectID)
	info, err := os.Stat(projectRoot)
	if err != nil || !info.IsDir() {
		return ExportResult{}, fmt.Errorf("找不到專案 %q", projectID)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	fileCount := 0
	redactionHits := 0

	err = filepath.Walk(projectRoot, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		top := strings.SplitN(rel, "/", 2)[0]
		if skipDirs[top] {
			if fi.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		// 只收一般檔案與目錄；symlink、device 一律不進備份。
		if fi.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if fi.IsDir() {
			return nil // tar 內以檔案路徑隱含目錄，還原時自動 MkdirAll
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("讀取 %s 失敗: %w", rel, err)
		}
		if redact && shouldRedact(rel) {
			cleaned, records := memory.RedactBeforeWrite(string(data))
			data = []byte(cleaned)
			redactionHits += len(records)
		}
		hdr := &tar.Header{
			Typeflag: tar.TypeReg,
			Name:     "project/" + rel,
			Mode:     0o600,
			Size:     int64(len(data)),
			ModTime:  fi.ModTime(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(data); err != nil {
			return err
		}
		fileCount++
		return nil
	})
	if err != nil {
		return ExportResult{}, err
	}

	manifest := Manifest{
		FormatVersion: FormatVersion,
		ProjectID:     projectID,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		Encrypted:     encrypt,
		Redacted:      redact,
		FileCount:     fileCount,
		RedactionHits: redactionHits,
	}
	mData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return ExportResult{}, err
	}
	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "manifest.json", Mode: 0o600, Size: int64(len(mData)), ModTime: time.Now(),
	}); err != nil {
		return ExportResult{}, err
	}
	if _, err := tw.Write(mData); err != nil {
		return ExportResult{}, err
	}
	if err := tw.Close(); err != nil {
		return ExportResult{}, err
	}
	if err := gz.Close(); err != nil {
		return ExportResult{}, err
	}

	var out []byte
	if encrypt {
		sealed, err := seal(buf.Bytes(), password)
		if err != nil {
			return ExportResult{}, err
		}
		out = sealed
	} else {
		out = append([]byte(magicPlain), buf.Bytes()...)
	}
	// 0600：備份檔僅限本人讀寫。
	if err := storage.AtomicWriteFile(destPath, out, 0o600); err != nil {
		return ExportResult{}, fmt.Errorf("寫入備份檔失敗: %w", err)
	}
	return ExportResult{
		BundlePath:    destPath,
		FileCount:     fileCount,
		Encrypted:     encrypt,
		Redacted:      redact,
		RedactionHits: redactionHits,
		SizeBytes:     int64(len(out)),
	}, nil
}

// shouldRedact 判斷哪些檔案在 redact 模式下要先遮蔽：
// memory/ 下的對話與摘要文字檔。結構化設定檔不動，避免破壞格式。
func shouldRedact(rel string) bool {
	if !strings.HasPrefix(rel, "memory/") {
		return false
	}
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".md", ".txt":
		return true
	}
	return false
}

// ──────────────────────────────────────────────
// 匯入
// ──────────────────────────────────────────────

// InspectBundle 只解密讀 manifest，不落地任何檔案（給匯入前預覽用）。
func InspectBundle(bundlePath, password string) (Manifest, error) {
	plain, err := openSealed(bundlePath, password)
	if err != nil {
		return Manifest{}, err
	}
	manifest, _, err := readTar(plain, "")
	return manifest, err
}

// ImportProject 解密並還原備份檔到 baseDir 下的 data/projects/。
// 流程：解密 → 解包到暫存目錄（staging）→ 驗證 manifest → rename 落位。
// 任一步失敗都不會留下半套專案。
func ImportProject(baseDir, bundlePath, password string, mode ImportMode) (ImportResult, error) {
	plain, err := openSealed(bundlePath, password)
	if err != nil {
		return ImportResult{}, err
	}

	projectsDir := filepath.Join(storage.DataRoot(baseDir), "projects")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		return ImportResult{}, err
	}
	staging, err := os.MkdirTemp(projectsDir, ".restore-")
	if err != nil {
		return ImportResult{}, err
	}
	defer os.RemoveAll(staging) // 成功 rename 後此目錄已不存在，失敗則清掉

	manifest, fileCount, err := readTar(plain, staging)
	if err != nil {
		return ImportResult{}, err
	}
	if manifest.FormatVersion != FormatVersion {
		return ImportResult{}, fmt.Errorf("備份檔版本 %d 不支援（目前支援 %d）", manifest.FormatVersion, FormatVersion)
	}
	if !validProjectID.MatchString(manifest.ProjectID) {
		return ImportResult{}, fmt.Errorf("備份檔內的專案 ID 不合法: %q", manifest.ProjectID)
	}

	targetID := manifest.ProjectID
	if mode == ModeCopy {
		targetID = fmt.Sprintf("%s-restored-%d", manifest.ProjectID, time.Now().Unix())
	}
	targetRoot := storage.ProjectRoot(baseDir, targetID)

	if _, err := os.Stat(targetRoot); err == nil {
		switch mode {
		case ModeOverwrite:
			backupOld := targetRoot + ".pre-restore"
			_ = os.RemoveAll(backupOld)
			if err := os.Rename(targetRoot, backupOld); err != nil {
				return ImportResult{}, fmt.Errorf("搬移既有專案失敗: %w", err)
			}
			if err := os.Rename(filepath.Join(staging, "project"), targetRoot); err != nil {
				// 還原失敗 → 把舊的搬回來
				_ = os.Rename(backupOld, targetRoot)
				return ImportResult{}, fmt.Errorf("還原落位失敗: %w", err)
			}
			_ = os.RemoveAll(backupOld)
		default:
			return ImportResult{}, ErrProjectExists
		}
	} else {
		if err := os.Rename(filepath.Join(staging, "project"), targetRoot); err != nil {
			return ImportResult{}, fmt.Errorf("還原落位失敗: %w", err)
		}
	}

	// 補齊必要目錄與初始檔（runtime/ 等匯出時被排除的部分）。
	if err := storage.EnsureProjectLayout(baseDir, targetID); err != nil {
		return ImportResult{}, fmt.Errorf("補齊專案結構失敗: %w", err)
	}

	return ImportResult{
		ProjectID:  manifest.ProjectID,
		RestoredAs: targetID,
		FileCount:  fileCount,
		Redacted:   manifest.Redacted,
		CreatedAt:  manifest.CreatedAt,
	}, nil
}

// readTar 解析 gzip+tar。staging 非空時將 project/ 內容落地到 staging，
// 為空時只讀 manifest。所有 entry 都做路徑穿越與大小檢查。
func readTar(plain []byte, staging string) (Manifest, int, error) {
	gz, err := gzip.NewReader(bytes.NewReader(plain))
	if err != nil {
		return Manifest{}, 0, ErrBadFormat
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	var manifest Manifest
	manifestSeen := false
	fileCount := 0
	var total int64

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Manifest{}, 0, ErrBadFormat
		}
		if hdr.Typeflag != tar.TypeReg {
			continue // 不接受 symlink/hardlink/device entry
		}
		if hdr.Size > maxEntrySize {
			return Manifest{}, 0, fmt.Errorf("備份內檔案過大: %s", hdr.Name)
		}
		total += hdr.Size
		fileCount++
		if total > maxTotalSize || fileCount > maxEntryCount {
			return Manifest{}, 0, errors.New("備份內容超出安全上限")
		}

		if hdr.Name == "manifest.json" {
			data, err := io.ReadAll(io.LimitReader(tr, maxEntrySize))
			if err != nil {
				return Manifest{}, 0, ErrBadFormat
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return Manifest{}, 0, ErrBadFormat
			}
			manifestSeen = true
			continue
		}

		rel, ok := strings.CutPrefix(hdr.Name, "project/")
		if !ok || rel == "" {
			return Manifest{}, 0, fmt.Errorf("備份內出現非預期路徑: %s", hdr.Name)
		}
		if staging == "" {
			// 只讀 manifest 模式：跳過內容。
			if _, err := io.Copy(io.Discard, io.LimitReader(tr, maxEntrySize)); err != nil {
				return Manifest{}, 0, ErrBadFormat
			}
			continue
		}
		projectDir := filepath.Join(staging, "project")
		// 路徑硬化：拒絕 ../、絕對路徑、跳出 staging 的 entry。
		dest, err := storage.ValidateZipEntry(projectDir, rel)
		if err != nil {
			return Manifest{}, 0, fmt.Errorf("備份內路徑不安全 %q: %w", hdr.Name, err)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return Manifest{}, 0, err
		}
		data, err := io.ReadAll(io.LimitReader(tr, maxEntrySize))
		if err != nil {
			return Manifest{}, 0, ErrBadFormat
		}
		if err := os.WriteFile(dest, data, 0o600); err != nil {
			return Manifest{}, 0, err
		}
	}
	if !manifestSeen {
		return Manifest{}, 0, ErrBadFormat
	}
	return manifest, fileCount, nil
}

// ──────────────────────────────────────────────
// 加解密
// ──────────────────────────────────────────────

func validatePassword(password string) error {
	if len([]rune(password)) < MinPasswordLen {
		return ErrPasswordTooShort
	}
	return nil
}

func deriveKey(password string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, keyLen)
}

// seal = header + AES-256-GCM(plaintext)，AAD 綁定 header。
func seal(plaintext []byte, password string) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	key, err := deriveKey(password, salt)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, nonceLen)
	if err != nil {
		return nil, err
	}
	header := make([]byte, 0, len(magicHeader)+saltLen+nonceLen)
	header = append(header, magicHeader...)
	header = append(header, salt...)
	header = append(header, nonce...)
	ciphertext := gcm.Seal(nil, nonce, plaintext, header)
	return append(header, ciphertext...), nil
}

// IsBundleEncrypted 只讀檔頭判斷是否為加密備份，讓 UI 決定要不要問密碼。
func IsBundleEncrypted(bundlePath string) (bool, error) {
	f, err := os.Open(bundlePath)
	if err != nil {
		return false, fmt.Errorf("讀取備份檔失敗: %w", err)
	}
	defer f.Close()
	head := make([]byte, len(magicHeader))
	if _, err := io.ReadFull(f, head); err != nil {
		return false, ErrBadFormat
	}
	switch string(head) {
	case magicHeader:
		return true, nil
	case magicPlain:
		return false, nil
	default:
		return false, ErrBadFormat
	}
}

// openSealed 開啟備份檔，自動辨識加密／明文格式。
// 明文格式忽略 password；加密格式要求密碼。
func openSealed(bundlePath, password string) ([]byte, error) {
	raw, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("讀取備份檔失敗: %w", err)
	}
	// 明文格式：去掉 magic 直接回 gzip(tar)。
	if len(raw) > len(magicPlain) && string(raw[:len(magicPlain)]) == magicPlain {
		return raw[len(magicPlain):], nil
	}
	// 加密格式。
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	headerLen := len(magicHeader) + saltLen + nonceLen
	if len(raw) < headerLen+1 || string(raw[:len(magicHeader)]) != magicHeader {
		return nil, ErrBadFormat
	}
	salt := raw[len(magicHeader) : len(magicHeader)+saltLen]
	nonce := raw[len(magicHeader)+saltLen : headerLen]
	key, err := deriveKey(password, salt)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, nonceLen)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, raw[headerLen:], raw[:headerLen])
	if err != nil {
		return nil, ErrDecryptFailed
	}
	return plain, nil
}
