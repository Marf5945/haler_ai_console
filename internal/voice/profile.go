package voice

import "strings"

// Voice profiles are presentation-layer only (spec §34.6): they may change
// voice_id, speaking rate, pitch/style parameters, and phrasing cadence.
// They must never change permissions, risk policy, tool availability,
// routing, memory behavior, or task authority.
//
// voice_age_style / voice_gender_style describe the synthetic voice style
// only. They must never be used to infer a real person's age, disability,
// health status, gender, or identity (spec §34.4 slow-speaker protection).

const (
	VoiceLivelyMale         = "lively_male"
	VoiceLowUncleMale       = "low_uncle_male"
	VoiceBrightGirl         = "bright_girl"
	VoiceProfessionalFemale = "professional_female"
	VoiceExcitedClearMale   = "excited_clear_male"
	VoiceOSDefault          = "os_default"
)

type VoiceProfile struct {
	VoiceID        string  `json:"voiceId"`
	PersonaBinding string  `json:"personaBinding,omitempty"`
	DisplayName    string  `json:"displayName"`
	Engine         string  `json:"engine"`
	StyleHint      string  `json:"styleHint"`
	SampleLine     string  `json:"sampleLine"`
	SafetyNote     string  `json:"safetyNote,omitempty"`
	VoiceAgeStyle  string  `json:"voiceAgeStyle"`
	VoiceGender    string  `json:"voiceGenderStyle"`
	// SpeakingRate is a multiplier over the platform default speaking rate.
	SpeakingRate float64 `json:"speakingRate"`
	// PitchOffset shifts the synthesized pitch baseline in roughly
	// semitone-sized steps. Negative values lower the voice.
	PitchOffset int `json:"pitchOffset"`
	// PreferredVoices lists OS voice names in preference order. The first
	// installed voice wins; otherwise the OS default voice is used with the
	// rate/pitch parameters of this profile.
	PreferredVoices []string `json:"preferredVoices,omitempty"`
	FallbackVoiceID string   `json:"fallbackVoiceId"`
	SupportedLangs  []string `json:"supportedLanguages"`
}

const engineOSNative = "os_native"

// Per-character OS voice preference lists (locale-neutral base names; the
// darwin synthesizer resolves them against localized `say -v ?` display
// names with a zh_TW → zh_CN → zh_HK locale preference). Modern macOS ships
// male Chinese-capable voices (Eddy / Reed / Rocko / Grandpa), so character
// differentiation comes from real voice selection; pitch offsets are only a
// best-effort fallback for legacy voices.
// 2026-07-19 使用者定案選角。設定面板顯示中文名，但 `say` 使用英文名＋
// (Enhanced)/(Premium) 後綴（實機確認）：月=Yue、翰(瀚)=Han、波波=Bobo、
// 黎瀲=Lilian。品質評分自動挑最高音質變體。警察首選聲音三（Eloquence，
// 需另行下載），未安裝時退到 Reed。
var (
	livelyMaleVoices   = []string{"Han", "翰", "Eddy", "Reed", "Rocko", "Meijia"}
	lowUncleVoices     = []string{"Bobo", "波波", "Rocko", "Grandpa", "Reed", "Eddy", "Meijia"}
	professionalVoices = []string{"Lilian", "黎瀲", "Meijia", "Mei-Jia", "Tingting", "Shelley", "Sandy"}
	// 警察定案：彬彬（聲音三為 Siri 專屬，say 不可用）。
	excitedMaleVoices  = []string{"Binbin", "彬彬", "Lisheng", "Yun", "Reed", "Eddy", "Rocko", "Meijia"}
	brightGirlVoices   = []string{"Yue", "月", "Flo", "Shelley", "Sandy", "Meijia"}
)

var zhLangs = []string{"zh", "en"}

var voiceProfiles = []VoiceProfile{
	{
		VoiceID:         VoiceLivelyMale,
		PersonaBinding:  "persona-a",
		DisplayName:     "憂樂傻酷・活潑男聲",
		Engine:          engineOSNative,
		StyleHint:       "bright, quick, friendly; short sentences",
		SampleLine:      "嗨嗨，我來啦，交給我吧！",
		VoiceAgeStyle:   "young_adult",
		VoiceGender:     "male",
		SpeakingRate:    1.08,
		PitchOffset:     -2,
		PreferredVoices: livelyMaleVoices,
		FallbackVoiceID: VoiceOSDefault,
		SupportedLangs:  zhLangs,
	},
	{
		VoiceID:         VoiceLowUncleMale,
		PersonaBinding:  "persona-b",
		DisplayName:     "厭世大叔・低沉男聲",
		Engine:          engineOSNative,
		StyleHint:       "low, dry, mature but reliable; slower pace",
		SampleLine:      "唉，好啦好啦，我看看。",
		VoiceAgeStyle:   "mature_adult",
		VoiceGender:     "male",
		SpeakingRate:    0.85,
		PitchOffset:     -6,
		PreferredVoices: lowUncleVoices,
		FallbackVoiceID: VoiceOSDefault,
		SupportedLangs:  zhLangs,
	},
	{
		VoiceID:         VoiceProfessionalFemale,
		PersonaBinding:  "persona-c",
		DisplayName:     "秘書・專業女聲",
		Engine:          engineOSNative,
		StyleHint:       "clear, stable, information-dense but not cold",
		SampleLine:      "午安，這裡是秘書，我幫您整理好了。",
		VoiceAgeStyle:   "adult",
		VoiceGender:     "female",
		SpeakingRate:    1.0,
		PitchOffset:     0,
		PreferredVoices: professionalVoices,
		FallbackVoiceID: VoiceOSDefault,
		SupportedLangs:  zhLangs,
	},
	{
		VoiceID:        VoiceExcitedClearMale,
		PersonaBinding: "persona-d",
		DisplayName:    "警察・激動但清晰的男聲",
		Engine:         engineOSNative,
		StyleHint:      "firm, energetic, intelligible; clear reminders",
		SampleLine:      "請注意！這個操作有風險，先確認再執行。",
		SafetyNote: "must not escalate risk tone into intimidation of, or " +
			"commands directed at, the user",
		VoiceAgeStyle:   "adult",
		VoiceGender:     "male",
		SpeakingRate:    1.12,
		PitchOffset:     6,
		PreferredVoices: excitedMaleVoices,
		FallbackVoiceID: VoiceOSDefault,
		SupportedLangs:  zhLangs,
	},
	{
		VoiceID:        VoiceBrightGirl,
		PersonaBinding: "persona-e",
		DisplayName:    "東春・熱情明亮聲線",
		Engine:         engineOSNative,
		StyleHint:      "warm, bright, curious; light cadence",
		SampleLine:      "找到問題了喔！我們快點修好它吧！",
		SafetyNote: "synthetic bright style only; must not use real minor " +
			"voice recordings or identifiable child voices",
		VoiceAgeStyle:   "bright_youthful_synthetic",
		VoiceGender:     "female",
		SpeakingRate:    1.05,
		PitchOffset:     4,
		PreferredVoices: brightGirlVoices,
		FallbackVoiceID: VoiceOSDefault,
		SupportedLangs:  zhLangs,
	},
}

var defaultVoiceProfile = VoiceProfile{
	VoiceID:         VoiceOSDefault,
	DisplayName:     "系統預設聲音",
	Engine:          engineOSNative,
	StyleHint:       "neutral platform default",
	SampleLine:      "這是系統預設聲音。",
	VoiceAgeStyle:   "adult",
	VoiceGender:     "neutral",
	SpeakingRate:    1.0,
	PitchOffset:     0,
	FallbackVoiceID: VoiceOSDefault,
	SupportedLangs:  zhLangs,
}

// personaNameBindings lets renamed or re-created personas keep their voice
// when the persona ID changed but the display name is one of the five
// default characters.
var personaNameBindings = map[string]string{
	"憂樂傻酷": VoiceLivelyMale,
	"厭世大叔": VoiceLowUncleMale,
	"秘書小妹": VoiceProfessionalFemale,
	"警察桂澤": VoiceExcitedClearMale,
	"東春巫女": VoiceBrightGirl,
}

// Profiles returns a copy of the five character voice profiles.
func Profiles() []VoiceProfile {
	out := make([]VoiceProfile, len(voiceProfiles))
	copy(out, voiceProfiles)
	return out
}

// DefaultVoiceProfile returns the neutral OS-default profile used for
// personas without a bound character voice.
func DefaultVoiceProfile() VoiceProfile {
	return defaultVoiceProfile
}

// ProfileByVoiceID resolves a profile by voice_id.
func ProfileByVoiceID(voiceID string) (VoiceProfile, bool) {
	for _, p := range voiceProfiles {
		if p.VoiceID == voiceID {
			return p, true
		}
	}
	if voiceID == VoiceOSDefault {
		return defaultVoiceProfile, true
	}
	return VoiceProfile{}, false
}

// ProfileForPersona maps a persona to its default voice profile. Unknown or
// custom personas fall back to the neutral OS-default profile; the persona
// name is used as a secondary match so renamed default personas keep their
// character voice.
func ProfileForPersona(personaID, personaName string) VoiceProfile {
	personaID = strings.TrimSpace(personaID)
	for _, p := range voiceProfiles {
		if p.PersonaBinding != "" && p.PersonaBinding == personaID {
			return p
		}
	}
	name := strings.TrimSpace(personaName)
	if voiceID, ok := personaNameBindings[name]; ok {
		if p, found := ProfileByVoiceID(voiceID); found {
			return p
		}
	}
	return defaultVoiceProfile
}
