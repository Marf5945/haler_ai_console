//go:build !darwin

// scheduler_wake_other.go — 非 macOS 平台的背景喚醒 stub（目前僅 macOS 支援）。
package main

import "fmt"

func (a *App) setSchedulerWake(enable bool) error {
	return fmt.Errorf("背景喚醒目前僅支援 macOS")
}
