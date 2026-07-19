//go:build windows

package voice

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// Windows OS-native synthesizer backed by System.Speech (SAPI) through
// PowerShell. OS-provided voices only; nothing is downloaded or bundled.
// PowerShell keeps DLL handling inside the OS runtime, matching the §14.6
// graceful-degradation rule; pitch offsets are not supported by plain SAPI
// and are intentionally ignored here (rate + voice-gender hints only).

type windowsSynth struct{}

func newPlatformSynthesizer() Synthesizer { return &windowsSynth{} }

func (w *windowsSynth) Name() string { return "os_native_windows_sapi" }

func (w *windowsSynth) Available() bool {
	_, err := exec.LookPath("powershell.exe")
	return err == nil
}

func (w *windowsSynth) Speak(ctx context.Context, req SynthRequest) error {
	if !w.Available() {
		return ErrTTSUnavailable
	}
	rate := req.Profile.SpeakingRate
	if rate <= 0 {
		rate = 1.0
	}
	// SAPI rate range is -10..10 around the platform default.
	sapiRate := int((rate - 1.0) * 10)
	if sapiRate > 10 {
		sapiRate = 10
	}
	if sapiRate < -10 {
		sapiRate = -10
	}
	genderHint := "Neutral"
	switch strings.ToLower(req.Profile.VoiceGender) {
	case "male":
		genderHint = "Male"
	case "female":
		genderHint = "Female"
	}
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Speech
$s = New-Object System.Speech.Synthesis.SpeechSynthesizer
$s.Rate = %d
try { $s.SelectVoiceByHints([System.Speech.Synthesis.VoiceGender]::%s) } catch {}
$text = [Console]::In.ReadToEnd()
$s.Speak($text)
`, sapiRate, genderHint)
	cmd := exec.CommandContext(ctx, "powershell.exe",
		"-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Stdin = strings.NewReader(req.Text)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("voice tts: sapi failed: %w", err)
	}
	return nil
}
