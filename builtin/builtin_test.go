// builtin_test.go — builtin package 單元測試。
// 覆蓋：store CRUD、import txt/md/docx、export txt/md/docx、
//
//	encoding 轉換（Big5/UTF-16）、path guard、size guard。
package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

// --- Store 測試 ---

func TestStoreSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	blob := &DocumentBlob{
		Meta: DocMeta{
			DocID:       "doc-test-1",
			DisplayName: "test.txt",
			Format:      "txt",
		},
		Content: "Hello 測試",
	}

	// Save
	if err := store.Save(blob); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load 回來比對
	loaded, err := store.Load("doc-test-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Content != "Hello 測試" {
		t.Errorf("content mismatch: got %q", loaded.Content)
	}
	if loaded.Meta.ContentHash == "" {
		t.Error("content hash should not be empty after Save")
	}
	if loaded.Meta.WordCount == 0 {
		t.Error("word count should not be zero")
	}
	if loaded.SchemaVersion != "document_blob.v1" {
		t.Errorf("schema version: got %q", loaded.SchemaVersion)
	}
}

func TestStoreList(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewStore(dir)

	// 存兩個文件
	store.Save(&DocumentBlob{Meta: DocMeta{DocID: "doc-a", DisplayName: "a.txt", Format: "txt"}, Content: "aaa"})
	store.Save(&DocumentBlob{Meta: DocMeta{DocID: "doc-b", DisplayName: "b.md", Format: "md"}, Content: "bbb"})

	metas, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 2 {
		t.Errorf("expected 2 metas, got %d", len(metas))
	}
}

func TestStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewStore(dir)

	store.Save(&DocumentBlob{Meta: DocMeta{DocID: "doc-del", DisplayName: "del.txt", Format: "txt"}, Content: "bye"})

	if err := store.Delete("doc-del"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Load 應該失敗
	_, err := store.Load("doc-del")
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

// --- Encoding 測試 ---

func TestDetectUTF8(t *testing.T) {
	input := []byte("Hello 你好世界")
	content, enc, err := DetectAndConvert(input)
	if err != nil {
		t.Fatalf("DetectAndConvert: %v", err)
	}
	if enc != "utf-8" {
		t.Errorf("expected utf-8, got %s", enc)
	}
	if content != "Hello 你好世界" {
		t.Errorf("content mismatch: %q", content)
	}
}

func TestDetectUTF8BOM(t *testing.T) {
	// UTF-8 BOM: EF BB BF
	input := append([]byte{0xEF, 0xBB, 0xBF}, []byte("BOM test")...)
	content, enc, err := DetectAndConvert(input)
	if err != nil {
		t.Fatalf("DetectAndConvert: %v", err)
	}
	if enc != "utf-8-bom" {
		t.Errorf("expected utf-8-bom, got %s", enc)
	}
	if content != "BOM test" {
		t.Errorf("content mismatch: %q", content)
	}
}

func TestDetectBig5(t *testing.T) {
	// 用 golang.org/x/text 正向編碼產生 Big5 bytes，避免手寫 hex 出錯
	big5Enc, _ := htmlindex.Get("big5")
	big5Bytes, _, err := transform.Bytes(big5Enc.NewEncoder(), []byte("測試"))
	if err != nil {
		t.Fatalf("encode to big5: %v", err)
	}
	content, enc, detectErr := DetectAndConvert(big5Bytes)
	if detectErr != nil {
		t.Fatalf("DetectAndConvert: %v", detectErr)
	}
	if enc != "big5" {
		t.Errorf("expected big5, got %s", enc)
	}
	if content != "測試" {
		t.Errorf("content mismatch: got %q, want 測試", content)
	}
}

// --- Path Guard 測試 ---

func TestPathGuardProjectInternal(t *testing.T) {
	root := t.TempDir()
	guard := NewPathGuard(root)

	internal := filepath.Join(root, "documents", "test.txt")
	os.MkdirAll(filepath.Dir(internal), 0o700)
	os.WriteFile(internal, []byte("x"), 0o600)

	if !guard.IsProjectInternal(internal) {
		t.Error("expected internal path to be recognized")
	}
	if guard.IsProjectInternal("/tmp/outside.txt") {
		t.Error("expected external path to be rejected")
	}
}

func TestPathGuardRejectDotDot(t *testing.T) {
	root := t.TempDir()
	guard := NewPathGuard(root)

	err := guard.ValidateImportPath(filepath.Join(root, "..", "escape.txt"))
	if err == nil {
		t.Error("expected error for .. traversal")
	}
}

func TestPathGuardNeedsConfirmation(t *testing.T) {
	root := t.TempDir()
	guard := NewPathGuard(root)

	if guard.NeedsConfirmation(filepath.Join(root, "inside.txt")) {
		t.Error("project internal should not need confirmation")
	}
	if !guard.NeedsConfirmation("/tmp/outside.txt") {
		t.Error("external path should need confirmation")
	}
}

// --- Size Guard 測試 ---

func TestClassifySize(t *testing.T) {
	if ClassifySize(100) != SizeInline {
		t.Error("100 bytes should be inline")
	}
	if ClassifySize(600*1024) != SizeStream {
		t.Error("600KB should be stream")
	}
	if ClassifySize(10*1024*1024) != SizeAsync {
		t.Error("10MB should be async")
	}
}

func TestReadWithGuard(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.txt")
	os.WriteFile(path, []byte("hello"), 0o600)

	data, class, err := ReadWithGuard(path, 0)
	if err != nil {
		t.Fatalf("ReadWithGuard: %v", err)
	}
	if class != SizeInline {
		t.Errorf("expected inline, got %d", class)
	}
	if string(data) != "hello" {
		t.Errorf("data mismatch: %q", data)
	}
}

// --- Import 測試 ---

func TestImportTxt(t *testing.T) {
	storeDir := t.TempDir()
	store, _ := NewStore(filepath.Join(storeDir, "documents"))

	// 建立測試 txt 檔案
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "hello.txt")
	os.WriteFile(srcPath, []byte("Hello from txt"), 0o600)

	guard := NewPathGuard(storeDir)
	result, err := ImportFromDrop(store, guard, srcPath, TFIDFVectorizer{})
	if err != nil {
		t.Fatalf("ImportFromDrop: %v", err)
	}
	if result.Blob.Meta.Format != "txt" {
		t.Errorf("format: got %q", result.Blob.Meta.Format)
	}
	if result.Blob.Content != "Hello from txt" {
		t.Errorf("content: got %q", result.Blob.Content)
	}
	if result.Encoding != "utf-8" {
		t.Errorf("encoding: got %q", result.Encoding)
	}
}

func TestImportMd(t *testing.T) {
	storeDir := t.TempDir()
	store, _ := NewStore(filepath.Join(storeDir, "documents"))

	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "readme.md")
	os.WriteFile(srcPath, []byte("# Title\n\nContent here"), 0o600)

	guard := NewPathGuard(storeDir)
	result, err := ImportFromDrop(store, guard, srcPath, TFIDFVectorizer{})
	if err != nil {
		t.Fatalf("ImportFromDrop: %v", err)
	}
	if result.Blob.Meta.Format != "md" {
		t.Errorf("format: got %q", result.Blob.Meta.Format)
	}
}

func TestImportUnsupportedFormat(t *testing.T) {
	storeDir := t.TempDir()
	store, _ := NewStore(filepath.Join(storeDir, "documents"))

	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "image.png")
	os.WriteFile(srcPath, []byte("fake png"), 0o600)

	guard := NewPathGuard(storeDir)
	_, err := ImportFromDrop(store, guard, srcPath, TFIDFVectorizer{})
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

// --- Docx Read 測試 ---

func TestExtractDocxText(t *testing.T) {
	// 先用 GenerateDocx 產生一個測試 docx，再用 ExtractDocxText 讀回來
	tmp := t.TempDir()
	docxPath := filepath.Join(tmp, "test.docx")

	original := "第一行\n第二行\n第三行"
	if err := GenerateDocx(original, docxPath); err != nil {
		t.Fatalf("GenerateDocx: %v", err)
	}

	extracted, err := ExtractDocxText(docxPath)
	if err != nil {
		t.Fatalf("ExtractDocxText: %v", err)
	}

	// 比對每一行
	origLines := strings.Split(original, "\n")
	extLines := strings.Split(extracted, "\n")
	if len(extLines) != len(origLines) {
		t.Fatalf("line count mismatch: orig=%d ext=%d", len(origLines), len(extLines))
	}
	for i, want := range origLines {
		if extLines[i] != want {
			t.Errorf("line %d: want %q got %q", i, want, extLines[i])
		}
	}
}

// --- Docx Write 測試 ---

func TestGenerateDocx(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "output.docx")

	if err := GenerateDocx("Hello World\n你好世界", path); err != nil {
		t.Fatalf("GenerateDocx: %v", err)
	}

	// 確認檔案存在且是有效 zip
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("output not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
	// 確認 zip 開頭 (PK magic bytes)
	data, _ := os.ReadFile(path)
	if len(data) < 4 || data[0] != 'P' || data[1] != 'K' {
		t.Error("output is not a valid zip file")
	}
}

func TestGenerateDocxSpecialChars(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "special.docx")

	// 包含 XML 特殊字元
	content := `<script>alert("xss")</script> & "quotes"`
	if err := GenerateDocx(content, path); err != nil {
		t.Fatalf("GenerateDocx: %v", err)
	}

	// 讀回來確認特殊字元被正確處理
	extracted, err := ExtractDocxText(path)
	if err != nil {
		t.Fatalf("ExtractDocxText: %v", err)
	}
	if extracted != content {
		t.Errorf("roundtrip mismatch:\nwant: %q\ngot:  %q", content, extracted)
	}
}

// --- Export 測試 ---

func TestExportTxt(t *testing.T) {
	storeDir := t.TempDir()
	store, _ := NewStore(filepath.Join(storeDir, "documents"))

	store.Save(&DocumentBlob{
		Meta:    DocMeta{DocID: "doc-exp-1", DisplayName: "export.txt", Format: "txt"},
		Content: "Export content",
	})

	tmpPath, err := ExportToTemp(store, "doc-exp-1")
	if err != nil {
		t.Fatalf("ExportToTemp: %v", err)
	}
	data, _ := os.ReadFile(tmpPath)
	if string(data) != "Export content" {
		t.Errorf("exported content mismatch: %q", data)
	}
	// 清理
	os.Remove(tmpPath)
}

func TestExportDocx(t *testing.T) {
	storeDir := t.TempDir()
	store, _ := NewStore(filepath.Join(storeDir, "documents"))

	store.Save(&DocumentBlob{
		Meta:    DocMeta{DocID: "doc-exp-docx", DisplayName: "export.docx", Format: "docx"},
		Content: "Docx export test\n第二段",
	})

	tmpPath, err := ExportToTemp(store, "doc-exp-docx")
	if err != nil {
		t.Fatalf("ExportToTemp: %v", err)
	}

	// 確認是 valid zip
	data, _ := os.ReadFile(tmpPath)
	if len(data) < 4 || data[0] != 'P' || data[1] != 'K' {
		t.Error("exported docx is not a valid zip")
	}

	// 讀回文字確認 roundtrip
	extracted, err := ExtractDocxText(tmpPath)
	if err != nil {
		t.Fatalf("ExtractDocxText: %v", err)
	}
	if !strings.Contains(extracted, "Docx export test") {
		t.Errorf("exported docx missing content: %q", extracted)
	}
	os.Remove(tmpPath)
}

// --- Import Docx 測試 ---

func TestImportDocx(t *testing.T) {
	// 先產生一個 .docx
	srcDir := t.TempDir()
	docxPath := filepath.Join(srcDir, "import_test.docx")
	GenerateDocx("Docx import 測試\n第二段落", docxPath)

	storeDir := t.TempDir()
	store, _ := NewStore(filepath.Join(storeDir, "documents"))
	guard := NewPathGuard(storeDir)

	result, err := ImportFromDrop(store, guard, docxPath, TFIDFVectorizer{})
	if err != nil {
		t.Fatalf("ImportFromDrop docx: %v", err)
	}
	if result.Blob.Meta.Format != "docx" {
		t.Errorf("format: got %q", result.Blob.Meta.Format)
	}
	if !strings.Contains(result.Blob.Content, "Docx import 測試") {
		t.Errorf("content missing: %q", result.Blob.Content)
	}
}

// --- CSV roundtrip 測試 ---

func TestCSVRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	csvPath := filepath.Join(tmp, "test.csv")

	// 寫 CSV
	content := "名稱\t數量\n蘋果\t3\n香蕉\t5"
	if err := WriteCSV(content, csvPath, ','); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}

	// 讀回來
	text, err := ReadCSV(csvPath, ',')
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}
	if !strings.Contains(text, "蘋果") || !strings.Contains(text, "香蕉") {
		t.Errorf("CSV roundtrip content missing: %q", text)
	}
}

// --- JSON roundtrip 測試 ---

func TestJSONRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	jsonPath := filepath.Join(tmp, "test.json")

	// 寫 JSON
	if err := WriteJSONPretty("Hello JSON 測試", jsonPath); err != nil {
		t.Fatalf("WriteJSONPretty: %v", err)
	}

	// 讀回來
	text, err := ExtractJSONText(jsonPath)
	if err != nil {
		t.Fatalf("ExtractJSONText: %v", err)
	}
	if !strings.Contains(text, "Hello JSON 測試") {
		t.Errorf("JSON roundtrip content missing: %q", text)
	}
}

// --- HTML roundtrip 測試 ---

func TestHTMLRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	htmlPath := filepath.Join(tmp, "test.html")

	// 寫 HTML
	if err := GenerateHTML("Hello HTML\n第二行", "Test", htmlPath); err != nil {
		t.Fatalf("GenerateHTML: %v", err)
	}

	// 讀回文字
	text, err := ExtractHTMLTextFromFile(htmlPath)
	if err != nil {
		t.Fatalf("ExtractHTMLTextFromFile: %v", err)
	}
	if !strings.Contains(text, "Hello HTML") || !strings.Contains(text, "第二行") {
		t.Errorf("HTML roundtrip content missing: %q", text)
	}
}

// --- RTF export 測試（僅 export）---

func TestRTFExport(t *testing.T) {
	tmp := t.TempDir()
	rtfPath := filepath.Join(tmp, "test.rtf")

	if err := GenerateRTF("RTF 測試\n第二段", rtfPath); err != nil {
		t.Fatalf("GenerateRTF: %v", err)
	}

	// 讀回檔案確認包含 RTF 頭
	data, err := os.ReadFile(rtfPath)
	if err != nil {
		t.Fatalf("read rtf: %v", err)
	}
	if !strings.HasPrefix(string(data), `{\rtf1`) {
		t.Error("RTF file missing header")
	}
	// 確認 Unicode escape 存在（中文字元）
	if !strings.Contains(string(data), `\u`) {
		t.Error("RTF should contain Unicode escape for CJK chars")
	}
}

// --- XLSX roundtrip 測試 ---

func TestXlsxRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	xlsxPath := filepath.Join(tmp, "test.xlsx")

	content := "Name\tAge\nAlice\t30\nBob\t25"
	if err := GenerateXlsx(content, xlsxPath); err != nil {
		t.Fatalf("GenerateXlsx: %v", err)
	}

	// 讀回來
	text, err := ExtractXlsxText(xlsxPath)
	if err != nil {
		t.Fatalf("ExtractXlsxText: %v", err)
	}
	if !strings.Contains(text, "Alice") || !strings.Contains(text, "Bob") {
		t.Errorf("XLSX roundtrip content missing: %q", text)
	}
}

func TestGenerateStyledXlsxWritesStylesAndColumns(t *testing.T) {
	tmp := t.TempDir()
	xlsxPath := filepath.Join(tmp, "styled.xlsx")

	err := GenerateStyledXlsx(XlsxSpec{
		SheetName: "報表",
		Rows: [][]XlsxCell{
			{
				{Value: "Name", Style: "header"},
				{Value: "Age", Style: "header"},
			},
			{
				{Value: "Alice"},
				{Value: 30},
			},
		},
		Cells: map[string]XlsxCell{
			"AA1": {Value: "遠欄", Style: "header"},
		},
		Styles: map[string]XlsxStyle{
			"header": {Bold: true, FontColor: "C00000", FillColor: "FFF2CC", Align: "center"},
		},
		ColWidths: map[string]float64{"A": 16, "AA": 20},
	}, xlsxPath)
	if err != nil {
		t.Fatalf("GenerateStyledXlsx: %v", err)
	}

	styles, err := zipReadFile(xlsxPath, "xl/styles.xml")
	if err != nil {
		t.Fatalf("read styles: %v", err)
	}
	stylesText := string(styles)
	for _, want := range []string{`<b/>`, `rgb="FFC00000"`, `rgb="FFFFF2CC"`, `horizontal="center"`, `<cellXfs count="2">`} {
		if !strings.Contains(stylesText, want) {
			t.Fatalf("styles.xml missing %s:\n%s", want, stylesText)
		}
	}

	sheet, err := zipReadFile(xlsxPath, "xl/worksheets/sheet1.xml")
	if err != nil {
		t.Fatalf("read sheet: %v", err)
	}
	sheetText := string(sheet)
	for _, want := range []string{`<col min="1" max="1" width="16.00"`, `<col min="27" max="27" width="20.00"`, `r="AA1"`, `s="1"`} {
		if !strings.Contains(sheetText, want) {
			t.Fatalf("sheet1.xml missing %s:\n%s", want, sheetText)
		}
	}
}

// --- PPTX roundtrip 測試 ---

func TestPptxRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	pptxPath := filepath.Join(tmp, "test.pptx")

	if err := GeneratePptx(pptxPath, "投影片標題\n第二行內容"); err != nil {
		t.Fatalf("GeneratePptx: %v", err)
	}

	text, err := ExtractPptxText(pptxPath)
	if err != nil {
		t.Fatalf("ExtractPptxText: %v", err)
	}
	if !strings.Contains(text, "投影片標題") || !strings.Contains(text, "第二行內容") {
		t.Errorf("PPTX roundtrip content missing: %q", text)
	}
}

// --- ODT roundtrip 測試 ---

func TestOdtRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	odtPath := filepath.Join(tmp, "test.odt")

	if err := GenerateOdt("ODT 文件\n第二段", odtPath); err != nil {
		t.Fatalf("GenerateOdt: %v", err)
	}

	text, err := ExtractOdtText(odtPath)
	if err != nil {
		t.Fatalf("ExtractOdtText: %v", err)
	}
	if !strings.Contains(text, "ODT 文件") || !strings.Contains(text, "第二段") {
		t.Errorf("ODT roundtrip content missing: %q", text)
	}
}

// --- ODS roundtrip 測試 ---

func TestOdsRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	odsPath := filepath.Join(tmp, "test.ods")

	content := "品名\t價格\n蘋果\t50\n香蕉\t30"
	if err := GenerateOds(content, odsPath); err != nil {
		t.Fatalf("GenerateOds: %v", err)
	}

	text, err := ExtractOdsText(odsPath)
	if err != nil {
		t.Fatalf("ExtractOdsText: %v", err)
	}
	if !strings.Contains(text, "蘋果") || !strings.Contains(text, "香蕉") {
		t.Errorf("ODS roundtrip content missing: %q", text)
	}
}

// --- ODP roundtrip 測試 ---

func TestOdpRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	odpPath := filepath.Join(tmp, "test.odp")

	if err := GenerateOdp("簡報標題\n第二頁內容", odpPath); err != nil {
		t.Fatalf("GenerateOdp: %v", err)
	}

	text, err := ExtractOdpText(odpPath)
	if err != nil {
		t.Fatalf("ExtractOdpText: %v", err)
	}
	if !strings.Contains(text, "簡報標題") || !strings.Contains(text, "第二頁內容") {
		t.Errorf("ODP roundtrip content missing: %q", text)
	}
}

// --- Export 新格式測試 ---

func TestExportCSV(t *testing.T) {
	storeDir := t.TempDir()
	store, _ := NewStore(filepath.Join(storeDir, "documents"))
	store.Save(&DocumentBlob{
		Meta:    DocMeta{DocID: "doc-csv", DisplayName: "data.csv", Format: "csv"},
		Content: "a\tb\n1\t2",
	})

	tmpPath, err := ExportToTemp(store, "doc-csv")
	if err != nil {
		t.Fatalf("ExportToTemp csv: %v", err)
	}
	if !strings.HasSuffix(tmpPath, ".csv") {
		t.Errorf("expected .csv extension, got %s", tmpPath)
	}
	os.Remove(tmpPath)
}

func TestExportXlsx(t *testing.T) {
	storeDir := t.TempDir()
	store, _ := NewStore(filepath.Join(storeDir, "documents"))
	store.Save(&DocumentBlob{
		Meta:    DocMeta{DocID: "doc-xlsx", DisplayName: "data.xlsx", Format: "xlsx"},
		Content: "A\tB\n1\t2",
	})

	tmpPath, err := ExportToTemp(store, "doc-xlsx")
	if err != nil {
		t.Fatalf("ExportToTemp xlsx: %v", err)
	}

	// 確認是 valid zip
	data, _ := os.ReadFile(tmpPath)
	if len(data) < 4 || data[0] != 'P' || data[1] != 'K' {
		t.Error("exported xlsx is not a valid zip")
	}
	os.Remove(tmpPath)
}

// --- Magic Bytes 驗證測試 ---

func TestMagicBytesValidZip(t *testing.T) {
	// 用 GenerateDocx 產生真實 zip，驗證通過
	tmp := t.TempDir()
	docxPath := filepath.Join(tmp, "real.docx")
	GenerateDocx("test", docxPath)

	if err := ValidateMagicBytes(docxPath, "docx"); err != nil {
		t.Errorf("valid docx should pass: %v", err)
	}
}

func TestMagicBytesFakeDocx(t *testing.T) {
	// 純文字偽裝成 .docx → 應該失敗
	tmp := t.TempDir()
	fakePath := filepath.Join(tmp, "fake.docx")
	os.WriteFile(fakePath, []byte("I am not a zip file"), 0o600)

	if err := ValidateMagicBytes(fakePath, "docx"); err == nil {
		t.Error("fake docx should fail magic bytes check")
	}
}

func TestMagicBytesTxtSkipped(t *testing.T) {
	// 純文字格式不做驗證，任何內容都通過
	tmp := t.TempDir()
	txtPath := filepath.Join(tmp, "test.txt")
	os.WriteFile(txtPath, []byte("anything"), 0o600)

	if err := ValidateMagicBytes(txtPath, "txt"); err != nil {
		t.Errorf("txt should skip magic bytes: %v", err)
	}
}

func TestMagicBytesJSON(t *testing.T) {
	tmp := t.TempDir()

	// 合法 JSON
	validPath := filepath.Join(tmp, "valid.json")
	os.WriteFile(validPath, []byte(`{"key":"val"}`), 0o600)
	if err := ValidateMagicBytes(validPath, "json"); err != nil {
		t.Errorf("valid json should pass: %v", err)
	}

	// 非法 JSON（非 { 或 [ 開頭）
	invalidPath := filepath.Join(tmp, "invalid.json")
	os.WriteFile(invalidPath, []byte("hello world"), 0o600)
	if err := ValidateMagicBytes(invalidPath, "json"); err == nil {
		t.Error("invalid json should fail")
	}
}

func TestMagicBytesRTF(t *testing.T) {
	tmp := t.TempDir()

	// 合法 RTF
	rtfPath := filepath.Join(tmp, "valid.rtf")
	os.WriteFile(rtfPath, []byte(`{\rtf1\ansi test}`), 0o600)
	if err := ValidateMagicBytes(rtfPath, "rtf"); err != nil {
		t.Errorf("valid rtf should pass: %v", err)
	}

	// 非法 RTF
	fakePath := filepath.Join(tmp, "fake.rtf")
	os.WriteFile(fakePath, []byte("not rtf"), 0o600)
	if err := ValidateMagicBytes(fakePath, "rtf"); err == nil {
		t.Error("fake rtf should fail")
	}
}

func TestExportPptx(t *testing.T) {
	storeDir := t.TempDir()
	store, _ := NewStore(filepath.Join(storeDir, "documents"))
	store.Save(&DocumentBlob{
		Meta:    DocMeta{DocID: "doc-pptx", DisplayName: "slides.pptx", Format: "pptx"},
		Content: "Slide content",
	})

	tmpPath, err := ExportToTemp(store, "doc-pptx")
	if err != nil {
		t.Fatalf("ExportToTemp pptx: %v", err)
	}

	data, _ := os.ReadFile(tmpPath)
	if len(data) < 4 || data[0] != 'P' || data[1] != 'K' {
		t.Error("exported pptx is not a valid zip")
	}
	os.Remove(tmpPath)
}
