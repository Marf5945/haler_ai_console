//go:build darwin

package voice

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// macOS OS-native synthesizer backed by /usr/bin/say (Apple Speech
// Synthesis). OS-provided voices only; nothing is downloaded or bundled.
//
// Voice matching note: `say -v ?` reports localized display names, e.g.
// "Eddy (中文（台灣）)" on a Chinese-language system but
// "Eddy (Chinese (Taiwan))" on an English one. Profiles therefore list
// locale-neutral base names ("Eddy", "Meijia", …) and matching combines the
// base name with a preferred-locale order (zh_TW → zh_CN → zh_HK).
//
// Pitch note: [[pbas]] embedded commands only work with legacy synthesizer
// voices; the modern neural voices ignore them. We still emit the command as
// best-effort, but character differentiation must come from actual voice
// selection, which is why male personas prefer Eddy / Reed / Rocko /
// Grandpa instead of a pitched-down female voice.

const (
	sayBinary             = "/usr/bin/say"
	sayBaseWordsPerMinute = 180
	sayMinWordsPerMinute  = 90
	sayMaxWordsPerMinute  = 320
)

var sayLocalePreference = []string{"zh_TW", "zh_CN", "zh_HK"}

type sayVoice struct {
	fullName string // exact name to pass to `say -v`
	base     string // lowercased name before the localized locale suffix
	locale   string // e.g. zh_TW
}

type darwinSynth struct {
	once   sync.Once
	voices []sayVoice
}

func newPlatformSynthesizer() Synthesizer { return &darwinSynth{} }

func (d *darwinSynth) Name() string { return "os_native_darwin_say" }

func (d *darwinSynth) Available() bool {
	info, err := os.Stat(sayBinary)
	return err == nil && !info.IsDir()
}

func (d *darwinSynth) loadVoices() {
	d.once.Do(func() {
		out, err := exec.Command(sayBinary, "-v", "?").Output()
		if err != nil {
			return
		}
		d.voices = parseSayVoices(string(out))
	})
}

// parseSayVoices parses `say -v ?` output lines of the form
// "<display name>  <locale>  # <sample text>".
func parseSayVoices(out string) []sayVoice {
	var voices []sayVoice
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " ")
		if strings.TrimSpace(line) == "" {
			continue
		}
		sep := strings.Index(line, "  ")
		if sep <= 0 {
			continue
		}
		fullName := strings.TrimSpace(line[:sep])
		rest := strings.TrimSpace(line[sep:])
		locale := rest
		if idx := strings.IndexAny(rest, " \t#"); idx > 0 {
			locale = rest[:idx]
		}
		if fullName == "" || locale == "" {
			continue
		}
		voices = append(voices, sayVoice{
			fullName: fullName,
			base:     sayVoiceBaseName(fullName),
			locale:   locale,
		})
	}
	return voices
}

// sayVoiceBaseName strips localized locale/quality suffixes:
// "Eddy (中文（台灣）)" → "eddy", "月（高音質）" → "月", "Meijia" → "meijia".
func sayVoiceBaseName(fullName string) string {
	name := fullName
	if idx := strings.Index(name, " ("); idx > 0 {
		name = name[:idx]
	}
	if idx := strings.Index(name, "（"); idx > 0 {
		name = name[:idx]
	}
	return strings.ToLower(strings.TrimSpace(name))
}

// voiceQualityScore ranks downloadable voice tiers so the best installed
// variant of the same base name wins: 高音質/Premium > 加強/進階/Enhanced >
// plain.
func voiceQualityScore(fullName string) int {
	lower := strings.ToLower(fullName)
	switch {
	case strings.Contains(fullName, "高音質") || strings.Contains(lower, "premium"):
		return 3
	case strings.Contains(fullName, "加強") || strings.Contains(fullName, "進階") ||
		strings.Contains(lower, "enhanced"):
		return 2
	default:
		return 1
	}
}

// pickVoice returns the exact `say -v` name for the profile's first
// installed preferred voice, honouring the locale preference order and
// picking the highest-quality installed variant of that name. Empty string
// means: use the OS default voice.
func (d *darwinSynth) pickVoice(profile VoiceProfile) string {
	d.loadVoices()
	for _, candidate := range profile.PreferredVoices {
		base := strings.ToLower(strings.TrimSpace(candidate))
		if base == "" {
			continue
		}
		for _, locale := range sayLocalePreference {
			if name := d.bestVariant(base, locale); name != "" {
				return name
			}
		}
		// Fall back to a base-name match in any locale (covers voices whose
		// only variant is not in the preference list).
		if name := d.bestVariant(base, ""); name != "" {
			return name
		}
	}
	return ""
}

// bestVariant returns the highest-quality installed voice with the given
// base name; locale == "" matches any locale.
func (d *darwinSynth) bestVariant(base, locale string) string {
	bestName := ""
	bestScore := 0
	for _, v := range d.voices {
		if v.base != base {
			continue
		}
		if locale != "" && v.locale != locale {
			continue
		}
		if score := voiceQualityScore(v.fullName); score > bestScore {
			bestScore = score
			bestName = v.fullName
		}
	}
	return bestName
}

func (d *darwinSynth) Speak(ctx context.Context, req SynthRequest) error {
	if !d.Available() {
		return ErrTTSUnavailable
	}
	args := []string{}
	if voiceName := d.pickVoice(req.Profile); voiceName != "" {
		args = append(args, "-v", voiceName)
	}
	rate := req.Profile.SpeakingRate
	if rate <= 0 {
		rate = 1.0
	}
	wpm := int(sayBaseWordsPerMinute * rate)
	if wpm < sayMinWordsPerMinute {
		wpm = sayMinWordsPerMinute
	}
	if wpm > sayMaxWordsPerMinute {
		wpm = sayMaxWordsPerMinute
	}
	args = append(args, "-r", strconv.Itoa(wpm))

	// Text is passed via stdin (say reads standard input when no text
	// argument is given), which avoids argv quoting issues entirely.
	payload := req.Text
	if req.Profile.PitchOffset != 0 {
		// Best-effort legacy pitch command; modern neural voices ignore it.
		payload = fmt.Sprintf("[[pbas %+d]] %s", req.Profile.PitchOffset, payload)
	}
	cmd := exec.CommandContext(ctx, sayBinary, args...)
	cmd.Stdin = strings.NewReader(payload)
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("voice tts: say failed: %w", err)
	}
	return nil
}
