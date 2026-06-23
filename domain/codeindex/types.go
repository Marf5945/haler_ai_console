package codeindex

import "time"

type SectionKind string

const (
	KindFunction  SectionKind = "function"
	KindMethod    SectionKind = "method"
	KindStruct    SectionKind = "struct"
	KindInterface SectionKind = "interface"
	KindType      SectionKind = "type"
	KindTypeAlias SectionKind = "type_alias"
	KindConst     SectionKind = "const_block"
	KindVar       SectionKind = "var_block"
)

type TagSource string

const (
	TagSourceSystem   TagSource = "system"
	TagSourceManual   TagSource = "manual"
	TagSourceInferred TagSource = "inferred"
)

type EdgeKind string

const (
	EdgeCalls    EdgeKind = "calls"
	EdgeUsesType EdgeKind = "uses_type"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type Section struct {
	ID           string      `json:"id"`
	FilePath     string      `json:"file_path"`
	Package      string      `json:"package"`
	SymbolName   string      `json:"symbol_name"`
	Receiver     string      `json:"receiver,omitempty"`
	Kind         SectionKind `json:"kind"`
	StartLine    int         `json:"start_line"`
	EndLine      int         `json:"end_line"`
	DocStartLine int         `json:"doc_start_line,omitempty"`
	DocEndLine   int         `json:"doc_end_line,omitempty"`
	Doc          string      `json:"doc,omitempty"`
	Summary      string      `json:"summary"`
	RiskLevel    RiskLevel   `json:"risk_level"`
	Hash         string      `json:"hash"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type Tag struct {
	SectionID string    `json:"section_id"`
	Tag       string    `json:"tag"`
	Weight    float64   `json:"weight"`
	Source    TagSource `json:"source"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Edge struct {
	FromSectionID string    `json:"from_section_id"`
	ToSymbol      string    `json:"to_symbol"`
	ToSectionID   string    `json:"to_section_id,omitempty"`
	Kind          EdgeKind  `json:"kind"`
	Weight        float64   `json:"weight"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type FileRecord struct {
	FilePath   string    `json:"file_path"`
	SHA256     string    `json:"sha256"`
	Package    string    `json:"package,omitempty"`
	ParseError string    `json:"parse_error,omitempty"`
	IndexedAt  time.Time `json:"indexed_at"`
}

type Snapshot struct {
	Version       int          `json:"version"`
	WorkspaceRoot string       `json:"workspace_root"`
	IndexedAt     time.Time    `json:"indexed_at"`
	Sections      []Section    `json:"sections"`
	Tags          []Tag        `json:"tags"`
	Edges         []Edge       `json:"edges"`
	Files         []FileRecord `json:"files"`
}

type RebuildResult struct {
	IndexedFiles int       `json:"indexed_files"`
	ReusedFiles  int       `json:"reused_files"`
	DeletedFiles int       `json:"deleted_files"`
	ParseErrors  int       `json:"parse_errors"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type QueryOptions struct {
	Limit           int  `json:"limit"`
	IncludeRelated  bool `json:"include_related"`
	ContextBefore   int  `json:"context_before"`
	ContextAfter    int  `json:"context_after"`
	MaxContextLines int  `json:"max_context_lines"`
	MaxContextBytes int  `json:"max_context_bytes"`
}

type Match struct {
	Section Section  `json:"section"`
	Score   int      `json:"score"`
	Tags    []string `json:"tags"`
	Reasons []string `json:"reasons"`
	Related bool     `json:"related"`
}

type ContextSection struct {
	Section   Section  `json:"section"`
	Score     int      `json:"score"`
	Tags      []string `json:"tags"`
	Reasons   []string `json:"reasons"`
	Related   bool     `json:"related"`
	Content   string   `json:"content"`
	Truncated bool     `json:"truncated"`
}
