// document_blob.go — DocumentBlob + DocMeta 結構定義。
// 合併式儲存：meta 和 content 在同一個 JSON 檔案中，確保原子性。
// 這是 builtin 文件能力的核心資料結構，所有 store/import/export 都依賴它。
package builtin

import "time"

// DocMeta 文件的 metadata，不含內容本體。
// List 操作只回傳 []DocMeta，避免載入大量內容。
type DocMeta struct {
	DocID           string    `json:"doc_id"`            // 唯一識別碼，格式: "doc-" + UnixNano
	DisplayName     string    `json:"display_name"`      // 使用者看到的檔名（含副檔名）
	Format          string    `json:"format"`            // "txt" | "md" | "docx"
	CreatedAt       time.Time `json:"created_at"`        // 建立時間
	UpdatedAt       time.Time `json:"updated_at"`        // 最後更新時間
	ContentHash     string    `json:"content_hash"`      // 內容的 SHA-256，用於去重和校驗
	OriginalHash    string    `json:"original_hash"`     // 原始檔 byte SHA-256，用於 W3A / 原檔校驗
	OriginalPath    string    `json:"original_path"`     // project-managed 原始檔副本位置
	W3AID           string    `json:"w3a_id"`            // 文件級 W3A 快取編號
	WordCount       int       `json:"word_count"`        // 字數（中文算字元數，英文算空白分隔）
	ChunkCount      int       `json:"chunk_count"`       // 可搜尋 chunks 數量
	VectorIndexedAt time.Time `json:"vector_indexed_at"` // 最近一次向量索引時間
	SourceHint      string    `json:"source_hint"`       // 使用者標記的來源提示（選填）
	Tags            []string  `json:"tags"`              // 使用者或 agent 指派的標籤
}

// DocumentBlob 合併式儲存單元：meta + content 在同一個 JSON。
// 磁碟上存為 documents/{doc_id}.json。
type DocumentBlob struct {
	SchemaVersion string `json:"schema_version"` // 固定 "document_blob.v1"
	Meta          DocMeta `json:"meta"`
	Content       string  `json:"content"` // 文件的純文字內容（已轉 UTF-8）
}
