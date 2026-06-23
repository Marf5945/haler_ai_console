// skill_output_surface.go — 通用「skill 產出物落位 + 上架」模組。
//
// 目標（呼應使用者需求）：
//  1. 自動收集：任何 skill 執行後，掃描 outputs 暫存目錄，把新產生的檔案收進來，
//     不需各 skill 自己接線（dianliao BOM 以外的 skill 也適用）。
//  2. 依類型分資料夾：圖片 → data/references/images、影片 → data/videos、
//     其餘文件 → data/references/files，與 localSearchRoots 的來源分類對齊。
//  3. 沿用產出檔名：產出端已取了有意義的名字（如 電料BOM_SLM003_時間.xlsx）就沿用；
//     只有像 output.xlsx / result.png 這種代號才補上「skill 名稱 + 時間」。
//  4. 呈現在右側：落地後發 reference:imported 事件，前端立即刷新引用文件面板
//     （右側面板原本就有 5 秒輪詢，這裡再加即時路徑）。
//
// 落地採「搬移」：來源在 outputs 的暫存檔會被移走，保持 outputs 乾淨。
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"ui_console/adapter/debugtrace"
	"ui_console/data/storage"
)

// outputKind 產出物分類。
type outputKind string

const (
	outputKindImage    outputKind = "image"
	outputKindVideo    outputKind = "video"
	outputKindDocument outputKind = "document"
)

var (
	skillOutputImageExts = map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".webp": true, ".bmp": true, ".svg": true, ".tiff": true, ".tif": true,
	}
	skillOutputVideoExts = map[string]bool{
		".mp4": true, ".mov": true, ".mkv": true, ".webm": true,
		".avi": true, ".m4v": true,
	}
)

// skillOutputsDir 是 skill / 小程式產出落地的暫存目錄（搬移來源）。
// 與 dianliao_bom_flow.go / task_progress_binding.go 的 outputs 路徑一致。
func skillOutputsDir() string {
	return filepath.Join(storage.ProjectRoot(appDataRoot(), "default"), "outputs")
}

// referenceImagesDir 圖片引用庫（與 localSearchRoots 的 image 來源對齊）。
func referenceImagesDir() string {
	return filepath.Join(appDataRoot(), "data", "references", "images")
}

// referenceFilesDir 文件引用庫。
func referenceFilesDir() string {
	return filepath.Join(appDataRoot(), "data", "references", "files")
}

// classifyOutputKind 依副檔名決定產出物類型與落地資料夾。
func classifyOutputKind(name string) (outputKind, string) {
	ext := strings.ToLower(filepath.Ext(name))
	switch {
	case skillOutputImageExts[ext]:
		return outputKindImage, referenceImagesDir()
	case skillOutputVideoExts[ext]:
		return outputKindVideo, videoLibraryDir()
	default:
		return outputKindDocument, referenceFilesDir()
	}
}

// meaninglessOutputStems 視為「代號」的檔名主幹（不含副檔名、轉小寫後比對）。
var meaninglessOutputStems = map[string]bool{
	"output": true, "outputs": true, "out": true, "result": true,
	"results": true, "tmp": true, "temp": true, "untitled": true,
	"new": true, "file": true, "data": true, "report": true,
	"export": true, "final": true, "image": true, "img": true,
}

// isMeaninglessOutputName 判斷檔名是否只是代號（如 output.xlsx / result.png / tmp_3.csv / 12345.png）。
func isMeaninglessOutputName(name string) bool {
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	stem = strings.TrimSpace(strings.ToLower(stem))
	if stem == "" {
		return true
	}
	if meaninglessOutputStems[stem] {
		return true
	}
	// 去掉尾端數字與分隔字元後再比一次（output1 / result-2 / tmp_3 → output/result/tmp）。
	trimmed := strings.TrimRight(stem, "0123456789-_ ")
	if trimmed == "" {
		return true // 全是數字與分隔字元
	}
	return meaninglessOutputStems[trimmed]
}

// safeOutputFileComponent 把字串清成可當檔名的片段（保留中文，擋路徑/控制字元）。
func safeOutputFileComponent(value string) string {
	value = strings.TrimSpace(value)
	const illegal = `<>:"/\|?*`
	var b strings.Builder
	for _, r := range value {
		switch {
		case r < 0x20:
			b.WriteRune('-')
		case r == ' ':
			b.WriteRune('-')
		case strings.ContainsRune(illegal, r):
			b.WriteRune('-')
		default:
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), ".-")
	if runes := []rune(out); len(runes) > 60 {
		out = string(runes[:60])
	}
	return out
}

// deriveSkillOutputBaseName 決定落地檔名：
// 沿用產出檔名；無意義（代號）時才補「skill 名稱 + 時間」。
func deriveSkillOutputBaseName(originalName, skillDisplayName string, when time.Time) string {
	if !isMeaninglessOutputName(originalName) {
		return originalName
	}
	ext := filepath.Ext(originalName)
	base := safeOutputFileComponent(skillDisplayName)
	if base == "" {
		base = "skill產出"
	}
	return fmt.Sprintf("%s_%s%s", base, when.Format("20060102-150405"), ext)
}

// moveIntoDirCollisionSafe 把 sourcePath 搬進 destDir，套用 preferredName；撞名補奈秒時間戳。
// 同分割區走 rename（最快）；跨裝置失敗則 copy+remove。
func moveIntoDirCollisionSafe(sourcePath, destDir, preferredName string) (string, error) {
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return "", err
	}
	base := strings.TrimSpace(preferredName)
	if base == "" {
		base = filepath.Base(sourcePath)
	}
	target := filepath.Join(destDir, base)
	if _, err := os.Stat(target); err == nil {
		ext := filepath.Ext(base)
		stem := base[:len(base)-len(ext)]
		target = filepath.Join(destDir, fmt.Sprintf("%s-%d%s", stem, time.Now().UnixNano(), ext))
	}
	if err := os.Rename(sourcePath, target); err != nil {
		if cerr := copyFileExclusive(sourcePath, target); cerr != nil {
			return "", cerr
		}
		_ = os.Remove(sourcePath)
	}
	return target, nil
}

// copyFileExclusive O_EXCL 複製，避免覆蓋既有檔。
func copyFileExclusive(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}

func skillOutputDetail(kind outputKind, skillDisplayName string) string {
	label := "檔案產出"
	switch kind {
	case outputKindImage:
		label = "圖片產出"
	case outputKindVideo:
		label = "影片產出"
	}
	if strings.TrimSpace(skillDisplayName) != "" {
		return label + "：" + strings.TrimSpace(skillDisplayName)
	}
	return label
}

// surfaceSkillOutput 把單一 skill 產出物分類落地到正確引用庫，並通知右側面板。
// 搬移（非複製）：來源在 outputs 的暫存檔會被移走。回傳落地後的 ReferenceFile。
func (a *App) surfaceSkillOutput(sourcePath, skillDisplayName string) (ReferenceFile, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return ReferenceFile{}, fmt.Errorf("skill output: source path is empty")
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return ReferenceFile{}, err
	}
	if info.IsDir() {
		return ReferenceFile{}, fmt.Errorf("skill output: folders are not supported")
	}
	originalName := filepath.Base(sourcePath)
	kind, destDir := classifyOutputKind(originalName)
	preferred := deriveSkillOutputBaseName(originalName, skillDisplayName, info.ModTime())

	finalPath, err := moveIntoDirCollisionSafe(sourcePath, destDir, preferred)
	if err != nil {
		return ReferenceFile{}, err
	}
	finalName := filepath.Base(finalPath)

	// 文件可索引就建向量索引（圖片/影片非可搜尋格式，indexReferenceFileIfNeeded 會自動略過）。
	if a != nil {
		if ierr := a.indexReferenceFileIfNeeded(finalPath, referenceVectorsDir(), a.currentVectorizer()); ierr != nil {
			debugtrace.Record("skill_output.index_error", "", map[string]interface{}{
				"name": finalName, "error": ierr.Error(),
			})
		}
		a.maybeEmitConfigMissing(finalName)
	}

	ref := ReferenceFile{
		Name:   finalName,
		Path:   finalPath,
		Source: "library",
		Status: "ready",
		Detail: skillOutputDetail(kind, skillDisplayName),
	}
	// 即時通知前端刷新右側引用文件面板（5 秒輪詢之外的即時路徑）。
	if a != nil && a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "reference:imported", map[string]interface{}{
			"name":   finalName,
			"path":   finalPath,
			"kind":   string(kind),
			"source": "skill_output",
			"skill":  skillDisplayName,
		})
	}
	return ref, nil
}

// harvestSkillOutputs 掃描 outputs 目錄，把「since 之後新產生 / 更新」的檔案逐一落地上架。
// 回傳成功上架的清單；個別失敗會被略過（記 trace），不中斷整批。
func (a *App) harvestSkillOutputs(skillDisplayName string, since time.Time) []ReferenceFile {
	dir := skillOutputsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	type cand struct {
		path string
		mod  time.Time
	}
	var cands []cand
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		info, ierr := entry.Info()
		if ierr != nil {
			continue
		}
		if info.ModTime().Before(since) {
			continue
		}
		cands = append(cands, cand{path: filepath.Join(dir, entry.Name()), mod: info.ModTime()})
	}
	// 由舊到新落地，落地後 ListReferenceFiles 會再依新→舊排序顯示。
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].mod.Before(cands[j].mod) })

	var landed []ReferenceFile
	for _, c := range cands {
		ref, serr := a.surfaceSkillOutput(c.path, skillDisplayName)
		if serr != nil {
			debugtrace.Record("skill_output.harvest_error", "", map[string]interface{}{
				"path": c.path, "error": serr.Error(),
			})
			continue
		}
		landed = append(landed, ref)
	}
	if len(landed) > 0 {
		debugtrace.Record("skill_output.harvested", "", map[string]interface{}{
			"skill": skillDisplayName, "count": len(landed),
		})
	}
	return landed
}

// ListReferenceImages 列出圖片引用庫 data/references/images（與一般清單合併顯示）。
func (a *App) ListReferenceImages() ([]ReferenceFile, error) {
	dir := referenceImagesDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make([]ReferenceFile, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasPrefix(name, ".") {
			continue
		}
		files = append(files, ReferenceFile{
			Name:   name,
			Path:   filepath.Join(dir, name),
			Source: "library",
			Status: "ready",
			Detail: "圖片",
		})
	}
	return files, nil
}
