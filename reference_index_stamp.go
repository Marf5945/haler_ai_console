package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type referenceIndexStamp struct {
	Size        int64 `json:"size"`
	ModUnixNano int64 `json:"mod_unix_nano"`
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

func writeReferenceIndexStamp(indexPath string, info os.FileInfo) error {
	if info == nil {
		return nil
	}
	stamp := referenceIndexStamp{
		Size:        info.Size(),
		ModUnixNano: info.ModTime().UnixNano(),
	}
	data, err := json.Marshal(stamp)
	if err != nil {
		return err
	}
	return os.WriteFile(referenceIndexStampPath(indexPath), data, 0o600)
}

func referenceIndexStampMatches(indexPath string, info os.FileInfo) bool {
	if info == nil {
		return false
	}
	if _, err := os.Stat(indexPath); err != nil {
		return false
	}
	stamp, ok := readReferenceIndexStamp(indexPath)
	if !ok {
		return false
	}
	return stamp.Size == info.Size() && stamp.ModUnixNano == info.ModTime().UnixNano()
}

func referenceIndexStampPath(indexPath string) string {
	ext := filepath.Ext(indexPath)
	base := indexPath[:len(indexPath)-len(ext)]
	return base + ".stamp.json"
}
