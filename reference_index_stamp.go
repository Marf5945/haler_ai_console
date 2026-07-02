// reference_index_stamp.go — 引用文件索引的「省工快取」（效能加速層）。
//
// 動機：indexReferenceFileIfNeeded 每次都要先跑 ExtractSearchableText（PDF/docx
// 抽字很貴）才能算 ContentHash 判斷索引是否過期。stamp 記錄「上次建索引時來源檔
// 的 size+mtime、以及當時的 vectorizer/chunker 指紋」——全部相符就直接跳過抽字。
//
// 正確性防線（兩層）：
//  1. stamp 不符 → 走原本的 ExtractSearchableText + ContentHash + IndexNeedsRebuild
//     完整判斷，結果永遠正確，最多只是慢一點。
//  2. stamp 相符的條件包含 vectorizer Type/ModelID/Dimension 與 ChunkerVersion，
//     換嵌入模型或升級切塊演算法時 stamp 自動失效，不會誤跳過重建。
//
// dense Dimension 必須兩邊都已知且相等才可命中快速路徑；未知時退回完整
// ContentHash 判斷，避免冷啟動時因未量到維度而誤跳過。
package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"ui_console/builtin"
)

type referenceIndexStamp struct {
	Size           int64  `json:"size"`
	ModUnixNano    int64  `json:"mod_unix_nano"`
	ChunkerVersion string `json:"chunker_version,omitempty"`
	VectorType     string `json:"vector_type,omitempty"`
	ModelID        string `json:"model_id,omitempty"`
	Dimension      int    `json:"dimension,omitempty"`
}

func readReferenceIndexStamp(indexPath string) (referenceIndexStamp, bool) {
	data, err := os.ReadFile(referenceIndexStampPath(indexPath))
	if err != nil {
		return referenceIndexStamp{}, false
	}
	var stamp referenceIndexStamp
	if err := json.Unmarshal(data, &stamp); err != nil {
		return referenceIndexStamp{}, false
	}
	if stamp.Size == 0 && stamp.ModUnixNano == 0 {
		return referenceIndexStamp{}, false
	}
	return stamp, true
}

func writeReferenceIndexStamp(indexPath string, info os.FileInfo, vec builtin.Vectorizer) error {
	if info == nil || vec == nil {
		return nil
	}
	meta := vec.Meta()
	stamp := referenceIndexStamp{
		Size:           info.Size(),
		ModUnixNano:    info.ModTime().UnixNano(),
		ChunkerVersion: builtin.ChunkerVersion,
		VectorType:     meta.Type,
		ModelID:        meta.ModelID,
		Dimension:      meta.Dimension,
	}
	data, err := json.Marshal(stamp)
	if err != nil {
		return err
	}
	return os.WriteFile(referenceIndexStampPath(indexPath), data, 0o600)
}

// referenceIndexStampMatches 回傳 true 表示可以完全跳過抽字與重建判斷。
func referenceIndexStampMatches(indexPath string, info os.FileInfo, vec builtin.Vectorizer) bool {
	if info == nil || vec == nil {
		return false
	}
	if _, err := os.Stat(indexPath); err != nil {
		return false // 索引本體不在，stamp 無效
	}
	stamp, ok := readReferenceIndexStamp(indexPath)
	if !ok {
		return false
	}
	if stamp.Size != info.Size() || stamp.ModUnixNano != info.ModTime().UnixNano() {
		return false
	}
	if stamp.ChunkerVersion != builtin.ChunkerVersion {
		return false
	}
	meta := vec.Meta()
	if stamp.VectorType != meta.Type || stamp.ModelID != meta.ModelID {
		return false
	}
	if meta.Type == "dense" {
		if stamp.Dimension <= 0 || meta.Dimension <= 0 {
			return false
		}
		if stamp.Dimension != meta.Dimension {
			return false
		}
	}
	return true
}

func referenceIndexStampPath(indexPath string) string {
	ext := filepath.Ext(indexPath)
	base := indexPath[:len(indexPath)-len(ext)]
	return base + ".stamp.json"
}
