package visual_learning

import "time"

// NativeClickEvent is recorded by the platform recorder when the user clicks
// outside the Wails WebView. Coordinates are OS screen coordinates.
type NativeClickEvent struct {
	Timestamp     time.Time `json:"timestamp"`
	X             int       `json:"x"`
	Y             int       `json:"y"`
	Button        string    `json:"button"`
	WindowTitle   string    `json:"window_title,omitempty"`
	WindowProcess string    `json:"window_process,omitempty"`
	WindowHandle  uintptr   `json:"window_handle,omitempty"`
	ClickCount    int       `json:"click_count,omitempty"`
	ScreenX       int       `json:"screen_x,omitempty"`
	ScreenY       int       `json:"screen_y,omitempty"`
	ScreenWidth   int       `json:"screen_width,omitempty"`
	ScreenHeight  int       `json:"screen_height,omitempty"`
	WindowRect    PixelBBox `json:"window_rect,omitempty"`
}

// NativeKeyboardEvent is recorded by the platform recorder when the user types
// outside the Wails WebView. Text is committed text only; shortcuts are stored
// as key/modifier pairs.
type NativeKeyboardEvent struct {
	Timestamp     time.Time `json:"timestamp"`
	Action        string    `json:"action"`
	Text          string    `json:"text,omitempty"`
	Key           string    `json:"key,omitempty"`
	KeyCode       int       `json:"key_code,omitempty"`
	Modifiers     []string  `json:"modifiers,omitempty"`
	WindowTitle   string    `json:"window_title,omitempty"`
	WindowProcess string    `json:"window_process,omitempty"`
	WindowHandle  uintptr   `json:"window_handle,omitempty"`
	ScreenWidth   int       `json:"screen_width,omitempty"`
	ScreenHeight  int       `json:"screen_height,omitempty"`
	WindowRect    PixelBBox `json:"window_rect,omitempty"`
}

type NativePermissionStatus struct {
	Accessibility   bool     `json:"accessibility"`
	InputMonitoring bool     `json:"input_monitoring"`
	ScreenRecording bool     `json:"screen_recording"`
	Requested       bool     `json:"requested,omitempty"`
	NeedsRestart    bool     `json:"needs_restart,omitempty"`
	Missing         []string `json:"missing,omitempty"`
	MissingKeys     []string `json:"missing_keys,omitempty"`
	Platform        string   `json:"platform,omitempty"`
	Message         string   `json:"message,omitempty"`
}

type WindowCapture struct {
	ImageData     []byte    `json:"image_data,omitempty"`
	Width         int       `json:"width"`
	Height        int       `json:"height"`
	WindowRect    PixelBBox `json:"window_rect"`
	WindowTitle   string    `json:"window_title,omitempty"`
	WindowProcess string    `json:"window_process,omitempty"`
	// Scale is the pixels-per-point ratio of the captured image relative to
	// WindowRect (e.g. 2.0 on a macOS Retina display). 0 must be treated as 1.
	Scale float64 `json:"scale,omitempty"`
}

// PixelScale returns the capture's pixels-per-point ratio, defaulting to 1.
func (c WindowCapture) PixelScale() float64 {
	if c.Scale > 0 {
		return c.Scale
	}
	return 1
}

// ResolvedWindow describes the current OS window matched to a recorded step.
// When the recorded native handle is stale (window closed/reopened), the
// platform implementation may re-find the window by process name and title;
// Refound is true in that case.
type ResolvedWindow struct {
	Handle  uintptr   `json:"handle"`
	PID     int       `json:"pid,omitempty"`
	Title   string    `json:"title,omitempty"`
	Process string    `json:"process,omitempty"`
	Rect    PixelBBox `json:"rect"`
	Refound bool      `json:"refound,omitempty"`
}

// NativeReplayResult is returned for one OS-level replay step.
type NativeReplayResult struct {
	OK                   bool     `json:"ok"`
	Skipped              bool     `json:"skipped,omitempty"`
	NeedsConfirmation    bool     `json:"needs_confirmation,omitempty"`
	Method               string   `json:"method"`
	Index                int      `json:"index,omitempty"`
	Label                string   `json:"label,omitempty"`
	Selector             string   `json:"selector,omitempty"`
	X                    int      `json:"x"`
	Y                    int      `json:"y"`
	OriginalX            int      `json:"original_x,omitempty"`
	OriginalY            int      `json:"original_y,omitempty"`
	Error                string   `json:"error,omitempty"`
	Warning              string   `json:"warning,omitempty"`
	WindowTitle          string   `json:"window_title,omitempty"`
	WindowProcess        string   `json:"window_process,omitempty"`
	ForegroundOK         bool     `json:"foreground_ok,omitempty"`
	ForegroundTitle      string   `json:"foreground_title,omitempty"`
	ForegroundProcess    string   `json:"foreground_process,omitempty"`
	Relocated            bool     `json:"relocated,omitempty"`
	RelocationMethod     string   `json:"relocation_method,omitempty"`
	RelocationConfidence float64  `json:"relocation_confidence,omitempty"`
	RelocationReason     string   `json:"relocation_reason,omitempty"`
	DebugImagePath       string   `json:"debug_image_path,omitempty"`
	DebugInfoPath        string   `json:"debug_info_path,omitempty"`
	Text                 string   `json:"text,omitempty"`
	Key                  string   `json:"key,omitempty"`
	Modifiers            []string `json:"modifiers,omitempty"`
	Sensitive            bool     `json:"sensitive,omitempty"`
}
