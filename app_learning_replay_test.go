package main

import (
	"testing"

	"ui_console/adapter/visual_learning"
)

func TestCanAutoConfirmBrowserReplayAllowsNormalBrowserPage(t *testing.T) {
	step := visual_learning.LearningReplayStep{
		WindowProcess: "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
		WindowTitle:   "YouTube - Google Chrome",
	}
	relocated := visual_learning.AnchorRelocationResult{Confidence: 0.61, Reason: "OpenCV fallback matched a candidate"}
	if !canAutoConfirmBrowserReplay(step, relocated) {
		t.Fatal("expected normal Chrome page replay to auto-confirm")
	}
}

func TestCanAutoConfirmBrowserReplayRejectsExplorer(t *testing.T) {
	step := visual_learning.LearningReplayStep{
		WindowProcess: "C:\\Windows\\explorer.exe",
		WindowTitle:   "native-window",
	}
	relocated := visual_learning.AnchorRelocationResult{Confidence: 0.9}
	if canAutoConfirmBrowserReplay(step, relocated) {
		t.Fatal("expected explorer replay to require confirmation")
	}
}

func TestCanAutoConfirmBrowserReplayRejectsDangerousBrowserText(t *testing.T) {
	step := visual_learning.LearningReplayStep{
		WindowProcess: "chrome.exe",
		WindowTitle:   "Checkout payment - Google Chrome",
	}
	relocated := visual_learning.AnchorRelocationResult{Confidence: 0.9}
	if canAutoConfirmBrowserReplay(step, relocated) {
		t.Fatal("expected payment page replay to require confirmation")
	}
}

func TestCanAutoConfirmLearningReplayAllowsDesktopAppOpen(t *testing.T) {
	step := visual_learning.LearningReplayStep{
		Action:        "click",
		Label:         "Claude",
		WindowProcess: "Claude",
		WindowTitle:   "Claude",
	}
	relocated := visual_learning.AnchorRelocationResult{Confidence: 0.78, Reason: "OpenCV fallback matched a candidate"}
	if !canAutoConfirmLearningReplay(step, relocated) {
		t.Fatal("expected simple desktop app open/switch replay to auto-confirm")
	}
}

func TestCanAutoConfirmLearningReplayAllowsDockOwnedAppOpen(t *testing.T) {
	step := visual_learning.LearningReplayStep{
		Action:        "click",
		Label:         "Cursor",
		WindowProcess: "Window Server",
		WindowTitle:   "Cursor",
	}
	relocated := visual_learning.AnchorRelocationResult{Confidence: 0.78, Reason: "OpenCV fallback matched a candidate"}
	if !canAutoConfirmLearningReplay(step, relocated) {
		t.Fatal("expected Dock/app icon replay to auto-confirm")
	}
}

func TestCanAutoConfirmLearningReplayRejectsDangerousDesktopText(t *testing.T) {
	step := visual_learning.LearningReplayStep{
		Action:        "click",
		Label:         "Delete",
		WindowProcess: "Claude",
		WindowTitle:   "Claude",
	}
	relocated := visual_learning.AnchorRelocationResult{Confidence: 0.9}
	if canAutoConfirmLearningReplay(step, relocated) {
		t.Fatal("expected dangerous desktop click to require confirmation")
	}
}

func TestCanAutoConfirmLearningReplayRejectsLowConfidenceDesktopAppOpen(t *testing.T) {
	step := visual_learning.LearningReplayStep{
		Action:        "click",
		Label:         "Claude",
		WindowProcess: "Claude",
		WindowTitle:   "Claude",
	}
	relocated := visual_learning.AnchorRelocationResult{Confidence: 0.62}
	if canAutoConfirmLearningReplay(step, relocated) {
		t.Fatal("expected low-confidence desktop app replay to require confirmation")
	}
}

func TestFallbackScaledWindowAnchorRelocationUsesCurrentWindowSize(t *testing.T) {
	step := visual_learning.LearningReplayStep{
		X: 900,
		Y: 480,
		WindowsAnchor: &visual_learning.WindowsClickAnchorResult{
			OK:              true,
			DetectorBackend: "directml",
			ExecutionPoint:  visual_learning.PixelPoint{X: 800, Y: 400},
			AnchorBBox:      visual_learning.PixelBBox{X: 760, Y: 380, W: 80, H: 40},
		},
	}
	capture := visual_learning.WindowCapture{
		Width:      500,
		Height:     250,
		WindowRect: visual_learning.PixelBBox{X: 100, Y: 50, W: 500, H: 250},
	}
	relocated, ok := fallbackScaledWindowAnchorRelocation(step, capture, 1000, 500, "no candidates")
	if !ok {
		t.Fatal("expected scaled fallback relocation")
	}
	if relocated.Method != "scaled_anchor_relocation" {
		t.Fatalf("method = %q", relocated.Method)
	}
	if relocated.ExecutionPoint.X != 500 || relocated.ExecutionPoint.Y != 250 {
		t.Fatalf("execution point = %+v, want absolute (500,250)", relocated.ExecutionPoint)
	}
	if relocated.OriginalPoint.X != 900 || relocated.OriginalPoint.Y != 480 {
		t.Fatalf("original point = %+v", relocated.OriginalPoint)
	}
	if relocated.AnchorBBox.X != 380 || relocated.AnchorBBox.Y != 190 || relocated.AnchorBBox.W != 40 || relocated.AnchorBBox.H != 20 {
		t.Fatalf("anchor bbox = %+v", relocated.AnchorBBox)
	}
}

func TestFallbackScaledWindowAnchorRelocationSkipsRecordedScreenAnchor(t *testing.T) {
	step := visual_learning.LearningReplayStep{
		WindowsAnchor: &visual_learning.WindowsClickAnchorResult{
			OK:              true,
			DetectorBackend: "recorded",
			ExecutionPoint:  visual_learning.PixelPoint{X: 800, Y: 400},
		},
	}
	capture := visual_learning.WindowCapture{Width: 500, Height: 250}
	if _, ok := fallbackScaledWindowAnchorRelocation(step, capture, 1000, 500, ""); ok {
		t.Fatal("expected recorded screen-coordinate anchor to be skipped")
	}
}
