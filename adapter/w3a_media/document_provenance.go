// document_provenance.go — §9A 文檔來源證明（第一刀：本地層 + Anchor 介面 no-op）。
//
// 與圖檔/音訊共用 W3AMediaInfo 管線，差別：
//   - MediaScope = ScopeDocument
//   - 文字沒有感知指紋；身分基準改「byte-hash + 正規化內容雜湊」：
//       Fingerprint.OverallByteHash       = sha256(raw)             // 精確檔案
//       Fingerprint.OverallPerceptualHash = "ndoc:" + sha256(norm)  // 唯一碼基準（格式微差仍同一份）
//     沿用本套件既有 "phash:"/"ahash:" 前綴慣例，extractHex 可剝前綴。
//   - 創作見證：沿用 AppOperationFingerprint 記錄「人類輸入 vs AI 處理」，
//     用 KeyManager 的金鑰（=錢包，私鑰永不離機）簽整份來源軌跡。
//
// Anchor 介面留給未來「上鏈/伺服器權威」；第一刀只附 LocalNoopAnchor（永不鑄碼）。
// AssetIdentity 為離線恆 inert 的預留欄位：UID/AnchorRef 在線上認證前皆空，
// 驗證流程一律不得依賴它。
//
// 零外部依賴：只用 stdlib（crypto/ed25519 等）。
package w3a_media

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// 版本標記。
const (
	DocNormalizerVersion = "norm-v1"
	AssetSchemaVersion   = "asset.v1"
	DocContentCodePrefix = "ndoc:" // 正規化內容雜湊在 OverallPerceptualHash 欄位的前綴
)

// 操作類型（Operation 欄位用值）；人類 vs AI 由產生檔案的系統依 Provenance/SourceType 判定。
const (
	OpHumanTextInput = "human_text_input"
	OpHumanEdit      = "human_edit"
	OpPaste          = "paste"
	OpAIGenerate     = "ai_generate"
	OpAIRewrite      = "ai_rewrite"
)

// ──────────────────────────────────────────────
// 預留欄位：資產身分（離線恆 inert）
// ──────────────────────────────────────────────

// AssetIdentity 線上認證/唯一碼/所有權預留欄位。
// 離線時 UID/AnchorRef/RegisteredAt 恆空、Registry 恆 unknown；
// OwnerPublicKey / OwnerProof 可本地先填（自簽，僅表「此金鑰背書此內容碼」）。
type AssetIdentity struct {
	UID            string         `json:"uid,omitempty"`              // 線上鑄出的唯一碼；未認證為空
	OwnerPublicKey string         `json:"owner_public_key,omitempty"` // 創造系統 Ed25519 公鑰（私鑰=錢包）
	OwnerProof     string         `json:"owner_proof,omitempty"`      // 私鑰對內容碼的簽章（hex）
	Registry       RegistryStatus `json:"registry,omitempty"`         // 連網前恆 unknown
	RegisteredAt   string         `json:"registered_at,omitempty"`    // ISO8601 鑄碼時間
	AnchorRef      string         `json:"anchor_ref,omitempty"`       // 預留：錨定參照（登錄號/ledger ref）
	AssetSchema    string         `json:"asset_schema,omitempty"`     // 本結構版本，獨立演進
}

// ──────────────────────────────────────────────
// 內容碼：byte + 正規化內容雜湊
// ──────────────────────────────────────────────

var reTrailingWS = regexp.MustCompile(`[ \t]+\n`)
var reMultiBlank = regexp.MustCompile(`\n{3,}`)

// NormalizeDocumentText 正規化：CRLF/CR→LF、去每行尾空白、收斂 3+ 連續空行為 1、首尾 Trim。
// 純位元組層級（stdlib），不做 NFC（避免拉 x/text 依賴）。
func NormalizeDocumentText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = reTrailingWS.ReplaceAllString(s, "\n")
	s = reMultiBlank.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// DocumentContentCode 回傳精確檔案 byte-hash 與正規化內容碼（皆 sha256:... 格式）。
// 正規化碼是「唯一性/錨定」的基準：格式微差（空白/換行）仍視為同一份。
func DocumentContentCode(raw []byte) (byteHash, normalizedCode string) {
	bh := sha256.Sum256(raw)
	nh := sha256.Sum256([]byte(NormalizeDocumentText(string(raw))))
	return fmt.Sprintf("sha256:%x", bh[:]), fmt.Sprintf("sha256:%x", nh[:])
}

// ──────────────────────────────────────────────
// 操作指紋（沿用圖檔機制）
// ──────────────────────────────────────────────

// NewDocOp 建一筆文檔操作指紋。op 用上面的 Op* 常數；影響範圍以字元位移表示。
func NewDocOp(op string, charStart, charEnd, count int) AppOperationFingerprint {
	return AppOperationFingerprint{
		Operation:     op,
		TimeRange:     time.Now().UTC().Format(time.RFC3339),
		AffectedRange: fmt.Sprintf("char:%d~%d", charStart, charEnd),
		SummaryCount:  count,
	}
}

// ──────────────────────────────────────────────
// 簽署 / 驗證來源軌跡（用 KeyManager 的金鑰＝錢包）
// ──────────────────────────────────────────────

// provenancePayload 取要簽的穩定位元組：scope + 兩個雜湊 + 操作軌跡。
// 故意排除簽章與 Asset（簽章不簽自己；Asset 屬可後填的線上層）。
func provenancePayload(info *W3AMediaInfo) ([]byte, error) {
	type sigPayload struct {
		Scope       MediaScope                `json:"scope"`
		ByteHash    string                    `json:"byte_hash"`
		ContentHash string                    `json:"content_hash"`
		Operations  []AppOperationFingerprint `json:"operations"`
	}
	return json.Marshal(sigPayload{
		Scope:       info.MediaScope,
		ByteHash:    info.Fingerprint.OverallByteHash,
		ContentHash: info.Fingerprint.OverallPerceptualHash,
		Operations:  info.Operations,
	})
}

// BuildDocumentProvenance 第一刀主入口：從原始位元組 + 操作軌跡建出文檔 W3AMediaInfo。
//   - 算 byte + 正規化內容碼
//   - 用 km 的金鑰簽整份來源軌跡（DeveloperSignature）
//   - 填 AssetIdentity 預留欄位（UID/AnchorRef 留空，OwnerPublicKey + 對內容碼的自簽先填）
func BuildDocumentProvenance(km *KeyManager, raw []byte, ops []AppOperationFingerprint) (*W3AMediaInfo, error) {
	if km == nil {
		return nil, fmt.Errorf("document provenance: nil key manager")
	}
	byteHash, contentCode := DocumentContentCode(raw)

	info := &W3AMediaInfo{
		Version:    "1.0",
		MediaScope: ScopeDocument,
		Status:     StatusW3AAppProcessed, // 由本 app 產生 + 簽署
		Fingerprint: DualLayerFingerprint{
			OverallByteHash:       byteHash,
			OverallPerceptualHash: DocContentCodePrefix + contentCode,
		},
		Operations: ops,
		CreatedAt:  time.Now().UTC(),
	}

	kp, err := km.GetOrCreateKeypair()
	if err != nil {
		return nil, err
	}
	privBytes, err := hex.DecodeString(kp.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	priv := ed25519.PrivateKey(privBytes)

	payload, err := provenancePayload(info)
	if err != nil {
		return nil, err
	}
	sig := ed25519.Sign(priv, payload)
	info.DeveloperSignature = &DeveloperSignature{
		AppID:     kp.AppID,
		PublicKey: kp.PublicKey,
		Signature: hex.EncodeToString(sig),
		SignedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	// 預留資產欄位：UID/AnchorRef 留空（尚未上鏈）；OwnerProof = 私鑰對內容碼自簽。
	ownerProof := ed25519.Sign(priv, []byte(contentCode))
	info.Asset = &AssetIdentity{
		OwnerPublicKey: kp.PublicKey,
		OwnerProof:     hex.EncodeToString(ownerProof),
		Registry:       RegistryUnknown,
		AssetSchema:    AssetSchemaVersion,
	}

	return info, nil
}

// VerifyDocumentProvenance 驗證文檔來源軌跡簽章是否有效（重算 payload 後驗章）。
// 注意：這只證明「此金鑰簽過這份軌跡」，不證明內容真為人工（見討論）。
func VerifyDocumentProvenance(info *W3AMediaInfo) (bool, error) {
	if info == nil || info.DeveloperSignature == nil {
		return false, nil
	}
	payload, err := provenancePayload(info)
	if err != nil {
		return false, err
	}
	pub, err := hex.DecodeString(info.DeveloperSignature.PublicKey)
	if err != nil {
		return false, fmt.Errorf("decode public key: %w", err)
	}
	sig, err := hex.DecodeString(info.DeveloperSignature.Signature)
	if err != nil {
		return false, fmt.Errorf("decode signature: %w", err)
	}
	return ed25519.Verify(ed25519.PublicKey(pub), payload, sig), nil
}

// ──────────────────────────────────────────────
// Anchor 介面（未來上鏈/伺服器權威的插槽）
// ──────────────────────────────────────────────

// AnchorReceipt 錨定登錄回執。
type AnchorReceipt struct {
	Code         string         `json:"code"`          // 被登錄的內容碼（正規化碼）
	UID          string         `json:"uid,omitempty"` // 鑄出的唯一碼
	Ref          string         `json:"ref,omitempty"` // 登錄號/ledger ref
	Status       RegistryStatus `json:"status"`
	RegisteredAt string         `json:"registered_at,omitempty"`
}

// Anchor 錨點抽象：未來換成伺服器權威或鏈，客戶端與檔案格式不變。
type Anchor interface {
	Register(code, ownerPublicKey string) (AnchorReceipt, error)
	Lookup(code string) (receipt AnchorReceipt, found bool, err error)
}

// LocalNoopAnchor 第一刀的離線實作：永不鑄碼，永遠回 unavailable/unknown。
type LocalNoopAnchor struct{}

func (LocalNoopAnchor) Register(code, ownerPublicKey string) (AnchorReceipt, error) {
	return AnchorReceipt{Code: code, Status: RegistryUnavailable}, nil
}

func (LocalNoopAnchor) Lookup(code string) (AnchorReceipt, bool, error) {
	return AnchorReceipt{Code: code, Status: RegistryUnknown}, false, nil
}

// ApplyAnchorReceipt 未來連網登錄成功後，把回執填回 Asset 預留欄位。
// no-op anchor 不會帶回 UID，故此呼叫對第一刀無副作用。
func ApplyAnchorReceipt(info *W3AMediaInfo, r AnchorReceipt) {
	if info == nil {
		return
	}
	if info.Asset == nil {
		info.Asset = &AssetIdentity{AssetSchema: AssetSchemaVersion}
	}
	if r.UID != "" {
		info.Asset.UID = r.UID
	}
	if r.Ref != "" {
		info.Asset.AnchorRef = r.Ref
	}
	if r.Status != "" {
		info.Asset.Registry = r.Status
	}
	if r.RegisteredAt != "" {
		info.Asset.RegisteredAt = r.RegisteredAt
	}
}

// DocumentUniqueCode 取要拿去錨定/比對的唯一碼（剝掉 ndoc: 前綴）。
func DocumentUniqueCode(info *W3AMediaInfo) string {
	if info == nil {
		return ""
	}
	return strings.TrimPrefix(info.Fingerprint.OverallPerceptualHash, DocContentCodePrefix)
}

// ──────────────────────────────────────────────
// Service 接線：為文檔建立 .w3a.json sidecar（平行於 CreateSidecar 的媒體版）
// ──────────────────────────────────────────────

// CreateDocumentSidecar 讀檔 → byte+正規化內容碼 → 簽署來源軌跡（沿用 recorder 的操作）
// → 寫 sidecar。加法接線，不影響既有媒體流程。
func (s *Service) CreateDocumentSidecar(filePath string) (*W3AMediaInfo, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read document: %w", err)
	}
	info, err := BuildDocumentProvenance(s.keyManager, raw, s.recorder.GetAll())
	if err != nil {
		return nil, err
	}
	info.FilePath = filePath
	MarkTrainingEligibility(info)
	if err := WriteSidecar(info, filePath); err != nil {
		return nil, err
	}
	return info, nil
}
