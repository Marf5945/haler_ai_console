//go:build darwin

// scheduler_wake_darwin.go — Phase G macOS 實作：用 user LaunchAgent + StartCalendarInterval
// 在排程時間喚醒機器並重啟 app（免 sudo，可從睡眠喚醒；關機無法喚醒）。
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const schedulerWakeLabel = "com.aiconsole.scheduler.wake"

func schedulerWakePlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", schedulerWakeLabel+".plist"), nil
}

// setSchedulerWake 安裝/移除「背景喚醒」LaunchAgent。
func (a *App) setSchedulerWake(enable bool) error {
	plistPath, err := schedulerWakePlistPath()
	if err != nil {
		return err
	}
	// 先卸載既有的（忽略錯誤：可能尚未載入）。
	_ = exec.Command("launchctl", "unload", plistPath).Run()
	if !enable {
		if rmErr := os.Remove(plistPath); rmErr != nil && !os.IsNotExist(rmErr) {
			return rmErr
		}
		return nil
	}
	intervals := a.schedulerWakeIntervals()
	if len(intervals) == 0 {
		_ = os.Remove(plistPath)
		return fmt.Errorf("沒有啟用中的排程，無需背景喚醒")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	plist := buildSchedulerWakePlist(schedulerWakeLabel, exe, intervals)
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
		return err
	}
	if out, lerr := exec.Command("launchctl", "load", plistPath).CombinedOutput(); lerr != nil {
		return fmt.Errorf("launchctl load 失敗: %v: %s", lerr, strings.TrimSpace(string(out)))
	}
	return nil
}

func buildSchedulerWakePlist(label, exe string, intervals []calendarInterval) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n<dict>\n")
	b.WriteString("  <key>Label</key>\n  <string>" + plistEscape(label) + "</string>\n")
	b.WriteString("  <key>ProgramArguments</key>\n  <array>\n")
	b.WriteString("    <string>" + plistEscape(exe) + "</string>\n")
	b.WriteString("    <string>--scheduled-wake</string>\n")
	b.WriteString("  </array>\n")
	b.WriteString("  <key>RunAtLoad</key>\n  <false/>\n")
	b.WriteString("  <key>ProcessType</key>\n  <string>Background</string>\n")
	b.WriteString("  <key>StartCalendarInterval</key>\n  <array>\n")
	for _, iv := range intervals {
		b.WriteString("    <dict>\n")
		writePlistInt(&b, "Minute", iv.Minute)
		writePlistInt(&b, "Hour", iv.Hour)
		writePlistInt(&b, "Day", iv.Day)
		writePlistInt(&b, "Month", iv.Month)
		writePlistInt(&b, "Weekday", iv.Weekday)
		b.WriteString("    </dict>\n")
	}
	b.WriteString("  </array>\n")
	b.WriteString("</dict>\n</plist>\n")
	return b.String()
}

func writePlistInt(b *strings.Builder, key string, v *int) {
	if v == nil {
		return
	}
	b.WriteString("      <key>" + key + "</key>\n      <integer>" + strconv.Itoa(*v) + "</integer>\n")
}

func plistEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
