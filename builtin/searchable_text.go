package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SupportedSearchableFormats are formats whose content can be extracted into
// plain text for local search and reference-vector indexing.
var SupportedSearchableFormats = map[string]bool{
	".txt":   true,
	".md":    true,
	".csv":   true,
	".tsv":   true,
	".json":  true,
	".jsonl": true,
	".log":   true,
	".yaml":  true,
	".yml":   true,
	".html":  true,
	".htm":   true,
	".docx":  true,
	".xlsx":  true,
	".pptx":  true,
	".odt":   true,
	".ods":   true,
	".odp":   true,
	".epub":  true,
}

// IsSearchableFormat reports whether ExtractSearchableText knows how to index
// the file extension. It intentionally includes localsearch-only text formats
// such as .jsonl/.log/.yaml in addition to document import formats.
func IsSearchableFormat(filePath string) bool {
	return SupportedSearchableFormats[strings.ToLower(filepath.Ext(filePath))]
}

// ExtractSearchableText converts supported local file formats into plain text.
// It is the shared path for local search and reference vector indexes, so
// supported file types do not drift between the two systems.
func ExtractSearchableText(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".txt", ".md", ".jsonl", ".log", ".yaml", ".yml":
		raw, _, err := ReadWithGuard(filePath, 0)
		if err != nil {
			return "", fmt.Errorf("searchable_text: read %s: %w", filePath, err)
		}
		converted, _, err := DetectAndConvert(raw)
		if err != nil {
			return "", fmt.Errorf("searchable_text: encoding %s: %w", filePath, err)
		}
		return converted, nil
	case ".csv":
		return ReadCSV(filePath, ',')
	case ".tsv":
		return ReadCSV(filePath, '\t')
	case ".json":
		text, err := ExtractJSONText(filePath)
		if err == nil && strings.TrimSpace(text) != "" {
			return text, nil
		}
		data, readErr := os.ReadFile(filePath)
		if readErr != nil {
			if err != nil {
				return "", err
			}
			return "", readErr
		}
		return string(data), nil
	case ".html", ".htm":
		return ExtractHTMLTextFromFile(filePath)
	case ".docx":
		return ExtractDocxText(filePath)
	case ".xlsx":
		return ExtractXlsxText(filePath)
	case ".pptx":
		return ExtractPptxText(filePath)
	case ".odt":
		return ExtractOdtText(filePath)
	case ".ods":
		return ExtractOdsText(filePath)
	case ".odp":
		return ExtractOdpText(filePath)
	case ".epub":
		return ExtractEpubText(filePath)
	default:
		return "", fmt.Errorf("searchable_text: unsupported format %q", ext)
	}
}
