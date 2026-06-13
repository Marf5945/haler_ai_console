// video_library_binding.go — 影片檔暫存於獨立資料夾 data/videos。
// UI 仍與其他檔案同一份清單顯示（共用 ReferenceFile DTO、Source "library"），
// 僅實體存放分開；資料夾另在 localSearchRoots 註冊為 agent 可取用來源。
package main

import (
	"os"
	"path/filepath"
	"strings"
)

// videoLibraryDir 影片獨立資料夾。
func videoLibraryDir() string {
	return filepath.Join(appDataRoot(), "data", "videos")
}

// ImportVideoFile 把影片複製進 data/videos（前端依副檔名分流呼叫）。
// 複用 importReferenceFileToDir：collision-safe、回傳 ReferenceFile（Source "library"）。
func (a *App) ImportVideoFile(sourcePath string) (ReferenceFile, error) {
	ref, err := importReferenceFileToDir(sourcePath, videoLibraryDir())
	if err != nil {
		return ReferenceFile{}, err
	}
	a.maybeEmitConfigMissing(ref.Name)
	return ref, nil
}

// ListVideoFiles 列出 data/videos 內的影片（形狀同 ReferenceFile，與一般清單合併顯示）。
func (a *App) ListVideoFiles() ([]ReferenceFile, error) {
	dir := videoLibraryDir()
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
		})
	}
	return files, nil
}
