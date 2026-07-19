package main

import (
	"fmt"
	"strings"

	"ui_console/internal/voice"
)

// Voice output bindings (spec §34.5 / §34.6).
//
// Voice profiles are presentation-layer only: switching persona changes the
// voice of future Voice Jobs and nothing else. Queued or playing jobs keep
// the voice_id snapshotted at creation time.

// VoiceProfiles returns the five character voice profiles.
func (a *App) VoiceProfiles() []voice.VoiceProfile {
	return voice.Profiles()
}

// VoicePackCatalog lists OS-native character voices plus optional
// downloadable packs. Optional packs are never auto-installed or
// auto-enabled from here.
func (a *App) VoicePackCatalog() []voice.VoiceCatalogEntry {
	available := a.voiceService.OutputEngine().Status().Available
	return a.voiceService.VoicePackCatalog(available)
}

// VoiceOutputStatus reports engine availability, the enabled flag, and the
// current queue state.
func (a *App) VoiceOutputStatus() voice.EngineStatus {
	return a.voiceService.OutputEngine().Status()
}

// SetVoiceOutputEnabled persists the TTS output toggle. Voice output stays
// off until the user explicitly enables it (spec §34.5).
func (a *App) SetVoiceOutputEnabled(enabled bool) (voice.State, error) {
	state := a.voiceService.Get(a.currentPanelLanguage())
	next := state.Settings
	next.TTSEnabled = enabled
	return a.voiceService.Save(next, a.currentPanelLanguage())
}

// SpeakVoiceLine enqueues a Voice Job for the active persona. kind is one of
// "probe", "readout", "safety_notice"; empty defaults to "readout". The job
// snapshots the active persona's voice_id at creation time.
func (a *App) SpeakVoiceLine(text string, kind string) (voice.VoiceJob, error) {
	jobKind := voice.VoiceJobKind(strings.TrimSpace(kind))
	switch jobKind {
	case voice.JobKindProbe, voice.JobKindReadout, voice.JobKindSafetyNotice:
	case "":
		jobKind = voice.JobKindReadout
	default:
		return voice.VoiceJob{}, fmt.Errorf("voice: unknown job kind %q", kind)
	}
	personaID, personaName, voiceID := a.activePersonaForVoice()
	profile := voice.ProfileForPersona(personaID, personaName)
	if selected, ok := voice.ProfileByVoiceID(voiceID); ok {
		profile = selected
	}
	return a.voiceService.OutputEngine().EnqueueFor(personaID, text, jobKind, profile)
}

// CancelVoiceProbes cancels queued and playing probe jobs. Call when the
// user resumes speaking: spoken probes must never overlap user speech.
func (a *App) CancelVoiceProbes() voice.EngineStatus {
	return a.voiceService.OutputEngine().CancelProbes()
}

// StopVoiceOutput is the universal-stop hook for voice output: it cancels
// the playing job and clears the queue.
func (a *App) StopVoiceOutput() voice.EngineStatus {
	return a.voiceService.OutputEngine().StopAll()
}

func (a *App) activePersonaForVoice() (string, string, string) {
	state := a.settingsService.State()
	personaID := state.ActivePersonaID
	for _, persona := range state.Personas {
		if persona.ID == personaID {
			return personaID, persona.Name, persona.VoiceID
		}
	}
	return personaID, "", ""
}

// PreviewVoiceProfile speaks the sample line of the given voice profile.
// User-initiated preview bypasses the TTS toggle (explicit consent).
func (a *App) PreviewVoiceProfile(voiceID string) (voice.VoiceJob, error) {
	profile, ok := voice.ProfileByVoiceID(voiceID)
	if !ok {
		return voice.VoiceJob{}, fmt.Errorf("voice: unknown voice id %q", voiceID)
	}
	return a.voiceService.OutputEngine().EnqueuePreview(profile.SampleLine, profile)
}

// PreviewVoiceProfileText speaks caller-provided localized preview text with
// the selected profile. User-initiated preview bypasses the TTS toggle.
func (a *App) PreviewVoiceProfileText(voiceID string, text string) (voice.VoiceJob, error) {
	profile, ok := voice.ProfileByVoiceID(voiceID)
	if !ok {
		return voice.VoiceJob{}, fmt.Errorf("voice: unknown voice id %q", voiceID)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		text = profile.SampleLine
	}
	return a.voiceService.OutputEngine().EnqueuePreview(text, profile)
}
