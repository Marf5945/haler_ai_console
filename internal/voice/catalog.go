package voice

// Voice Pack Catalog (spec §34.6 / §34.8).
//
// The catalog lists the five OS-native character voice profiles (no
// download required) plus optional third-party packs. Optional packs are
// display-only entries here: they are never auto-downloaded and never
// auto-enabled. Installation goes exclusively through the pinned
// InstallTTSPack flow (source allowlist + SHA256 + quarantine + explicit
// user enable), and stays "not_configured" until an artifact is pinned and
// a Developer Review Card exists (§22.4).

type VoiceCatalogEntry struct {
	VoiceID            string   `json:"voiceId"`
	PersonaBinding     string   `json:"personaBinding,omitempty"`
	DisplayName        string   `json:"displayName"`
	Engine             string   `json:"engine"`
	EngineVersion      string   `json:"engineVersion,omitempty"`
	ModelVersion       string   `json:"modelVersion,omitempty"`
	SourceURL          string   `json:"sourceUrl,omitempty"`
	License            string   `json:"license"`
	SHA256             string   `json:"sha256,omitempty"`
	ExpectedBytes      int64    `json:"expectedBytes,omitempty"`
	MaxBytes           int64    `json:"maxBytes,omitempty"`
	SupportedLanguages []string `json:"supportedLanguages,omitempty"`
	FallbackVoiceID    string   `json:"fallbackVoiceId,omitempty"`
	DefaultRate        float64  `json:"defaultRate,omitempty"`
	DefaultPitchOffset int      `json:"defaultPitchOffset,omitempty"`
	VoiceAgeStyle      string   `json:"voiceAgeStyle,omitempty"`
	VoiceGenderStyle   string   `json:"voiceGenderStyle,omitempty"`
	Optional           bool     `json:"optional"`
	Installed          bool     `json:"installed"`
	Status             string   `json:"status"`
	Reason             string   `json:"reason,omitempty"`
}

const (
	catalogStatusOSReady       = "os_native_ready"
	catalogStatusOSUnavailable = "os_native_unavailable"
)

// VoicePackCatalog returns the five character voice entries plus optional
// downloadable packs. osEngineAvailable reports whether the platform
// synthesizer is usable.
func (s *Service) VoicePackCatalog(osEngineAvailable bool) []VoiceCatalogEntry {
	entries := make([]VoiceCatalogEntry, 0, len(voiceProfiles)+1)
	status := catalogStatusOSReady
	if !osEngineAvailable {
		status = catalogStatusOSUnavailable
	}
	for _, p := range Profiles() {
		entries = append(entries, VoiceCatalogEntry{
			VoiceID:            p.VoiceID,
			PersonaBinding:     p.PersonaBinding,
			DisplayName:        p.DisplayName,
			Engine:             p.Engine,
			License:            "OS-provided voice; not redistributed",
			SupportedLanguages: p.SupportedLangs,
			FallbackVoiceID:    p.FallbackVoiceID,
			DefaultRate:        p.SpeakingRate,
			DefaultPitchOffset: p.PitchOffset,
			VoiceAgeStyle:      p.VoiceAgeStyle,
			VoiceGenderStyle:   p.VoiceGender,
			Optional:           false,
			Installed:          osEngineAvailable,
			Status:             status,
		})
	}

	pack := s.TTSPackStatus()
	entries = append(entries, VoiceCatalogEntry{
		VoiceID:       pack.PackID,
		DisplayName:   pack.Label,
		Engine:        pack.Engine,
		License:       pack.License,
		ExpectedBytes: pack.ExpectedBytes,
		Optional:      true,
		Installed:     pack.Available,
		Status:        pack.Status,
		Reason:        pack.Reason,
	})
	return entries
}
