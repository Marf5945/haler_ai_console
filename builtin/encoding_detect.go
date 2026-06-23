// encoding_detect.go — 字元編碼偵測與轉換。
// 台灣使用者常見 Big5、UTF-16 BOM 的 .txt 檔案，不處理會亂碼。
// 依賴 golang.org/x/text（已在 go.mod indirect，Wails 帶入）。
package builtin

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// DetectAndConvert 偵測 raw bytes 的編碼，轉成 UTF-8 字串。
// 偵測順序：UTF-8 BOM → UTF-16 BOM → 有效 UTF-8 → Big5 嘗試 → 原樣回傳。
func DetectAndConvert(raw []byte) (string, string, error) {
	// 1. UTF-8 BOM (EF BB BF)
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		return string(raw[3:]), "utf-8-bom", nil
	}

	// 2. UTF-16 LE BOM (FF FE)
	if len(raw) >= 2 && raw[0] == 0xFF && raw[1] == 0xFE {
		decoded, err := decodeWith(raw[2:], unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM))
		if err != nil {
			return "", "", fmt.Errorf("encoding: utf-16le decode: %w", err)
		}
		return decoded, "utf-16le", nil
	}

	// 3. UTF-16 BE BOM (FE FF)
	if len(raw) >= 2 && raw[0] == 0xFE && raw[1] == 0xFF {
		decoded, err := decodeWith(raw[2:], unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM))
		if err != nil {
			return "", "", fmt.Errorf("encoding: utf-16be decode: %w", err)
		}
		return decoded, "utf-16be", nil
	}

	// 4. 有效 UTF-8（無 BOM）
	if utf8.Valid(raw) {
		return string(raw), "utf-8", nil
	}

	// 5. 嘗試 Big5（台灣常見）
	decoded, err := decodeWith(raw, mustLookup("big5"))
	if err == nil && utf8.ValidString(decoded) {
		return decoded, "big5", nil
	}

	// 6. 嘗試 GBK（簡體中文，次要）
	decoded, err = decodeWith(raw, mustLookup("gbk"))
	if err == nil && utf8.ValidString(decoded) {
		return decoded, "gbk", nil
	}

	// 7. 無法辨識，原樣回傳（可能會亂碼，但不阻斷流程）
	return string(raw), "unknown", nil
}

// decodeWith 用指定編碼解碼 bytes。
func decodeWith(raw []byte, enc encoding.Encoding) (string, error) {
	reader := transform.NewReader(bytes.NewReader(raw), enc.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// mustLookup 從 htmlindex 查詢編碼。找不到會 panic（只在初始化時呼叫已知名稱）。
func mustLookup(name string) encoding.Encoding {
	enc, err := htmlindex.Get(name)
	if err != nil {
		panic(fmt.Sprintf("encoding_detect: unknown encoding %q: %v", name, err))
	}
	return enc
}
