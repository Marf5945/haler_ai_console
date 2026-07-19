//go:build !darwin && !windows

package voice

import "context"

// Platforms without an OS-native TTS integration degrade gracefully: voice
// output reports unavailable and the UI falls back to caption text with a
// notification sound (spec §34.5).

type unavailableSynth struct{}

func newPlatformSynthesizer() Synthesizer { return unavailableSynth{} }

func (unavailableSynth) Name() string    { return "os_native_unavailable" }
func (unavailableSynth) Available() bool { return false }

func (unavailableSynth) Speak(ctx context.Context, req SynthRequest) error {
	return ErrTTSUnavailable
}
