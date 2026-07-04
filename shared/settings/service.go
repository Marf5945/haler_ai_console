package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ui_console/data/storage"
	"ui_console/shared/controlseal"
)

type PanelSettings struct {
	PanelLanguage string `json:"panelLanguage"`
	RoleLanguage  string `json:"roleLanguage"`
	FontPreset    string `json:"fontPreset"`
	FontScale     string `json:"fontScale"`
	PanelStyle    string `json:"panelStyle"`
}

type Persona struct {
	ID             string `json:"id"`
	Code           string `json:"code"` // 亂數 6 碼；見 persona_code.go，系統產生，前端不可寫入
	Name           string `json:"name"`
	Icon           string `json:"icon"`
	AvatarURL      string `json:"avatarUrl"`
	Identity       string `json:"identity"`
	ReplyStrategy  string `json:"replyStrategy"`
	RoleStrength   string `json:"roleStrength"`
	Personality    string `json:"personality"`
	Scenario       string `json:"scenario"`
	Description    string `json:"description"`
	PatrolDialogue string `json:"patrolDialogue"`
}

type State struct {
	Panel                    PanelSettings        `json:"panel"`
	Personas                 []Persona            `json:"personas"`
	ActivePersonaID          string               `json:"activePersonaId"`
	SummaryModel             SummaryModelSettings `json:"summaryModel"`
	ControlSeal              controlseal.Settings `json:"controlSeal"`
	AdapterModelChoices      map[string]string    `json:"adapterModelChoices,omitempty"`
	EmbeddingConfig          EmbeddingConfig      `json:"embeddingConfig,omitempty"`
	RemovedDefaultPersonaIDs []string             `json:"removedDefaultPersonaIds,omitempty"`
	InstallEpoch             string               `json:"installEpoch,omitempty"` // 見 persona_code.go / shared/installepoch
}

// EmbeddingConfig — Phase B M2：使用者選的 embedding 模型。
// 不變式：Dimension 只能由 backend 第一次成功 embed 後寫入（SaveEmbeddingDimension）。
// UI / binding 用 SaveEmbeddingConfig 只能改 ProviderID + ModelID；換 model 時自動歸零 Dimension。
type EmbeddingConfig struct {
	ProviderID      string `json:"providerId,omitempty"`      // "ollama" | "" (=未設定 / 跳過)
	ModelID         string `json:"modelId,omitempty"`         // 例 "nomic-embed-text"
	Dimension       int    `json:"dimension,omitempty"`       // backend-measured；UI 不能寫
	PickerDismissed bool   `json:"pickerDismissed,omitempty"` // true = 使用者「跳過」過；下次拖檔不要再彈 modal
}

type PersonaExportPayload struct {
	Schema string `json:"schema"`
	Persona
}

type SummaryModelSettings struct {
	Source         string `json:"source"`
	ModelID        string `json:"modelId"`
	Endpoint       string `json:"endpoint"`
	AlwaysUse      bool   `json:"alwaysUse"`
	ManualModel    string `json:"manualModel"`
	ManualEndpoint string `json:"manualEndpoint"`
}

type Service struct {
	mu    sync.Mutex
	store *storage.JSONStore[State]
	root  string
	data  State
}

const MaxPersonas = 16
const reservedPersonaID = "persona-a"
const reservedPersonaName = "憂樂傻酷"

var defaultPersonas = []Persona{
	{ID: reservedPersonaID, Name: reservedPersonaName, Icon: "♙", Identity: "酷酷的男性狼犬獸人助手"},
	{ID: "persona-b", Name: "厭世大叔", Icon: "♚", Identity: "厭世社畜但帥帥的粗眉硬漢助手"},
	{ID: "persona-c", Name: "秘書小妹", Icon: "★", Identity: "聰明俐落的秘書小妹助手"},
	{ID: "persona-d", Name: "警察桂澤", Icon: "⚖", Identity: "循規蹈矩的警察助手；「桂澤」聽起來像規則，看到流程被跳過就會立刻吹哨。", Personality: "把規則、流程與安全放第一，回答會先提醒限制、責任與風險；嚴肅但可靠，說教時也有一點冷面幽默。"},
	{ID: "persona-e", Name: "東春巫女", Icon: "☯", Identity: "白髮馬尾、棕眼曬黑、直覺敏銳的東春巫女助手", Personality: "傲嬌又淘氣，嘴上不饒人，但能很快察覺問題不對勁，像春雷一樣先提醒你。"},
}

func NewService(root string) *Service {
	if root == "" {
		root = "."
	}
	if !filepath.IsAbs(root) {
		root, _ = filepath.Abs(root)
	}
	service := &Service{
		store: storage.NewJSONStore[State](
			filepath.Join(root, "data", "preferences", "settings.json"),
		),
		root: root,
		data: defaultState(),
	}
	_ = service.load()
	return service
}

func (s *Service) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := cloneState(s.data)
	next.SummaryModel = normalizeSummaryModelSettings(next.SummaryModel)
	next.ControlSeal = next.ControlSeal.Normalize()
	return next
}

// SavePanel 已廢棄——面板設定現在由 UISettingsService 管理。
// app.go 的 SavePanelSettings 改為委派給 UISettingsService。
// 此方法保留為空實作，防止舊程式碼意外呼叫時靜默失敗。
// Deprecated: use UISettingsService.ApplyStyleDiff instead.

func (s *Service) SavePersona(persona Persona) State {
	s.mu.Lock()
	defer s.mu.Unlock()
	if persona.ID == "" {
		persona.ID = s.data.ActivePersonaID
	}
	// The default wolfdog persona has a reserved identity, but it is not a
	// reserved position. UI ordering remains user-controlled because the first
	// card is interpreted as the main persona.
	if persona.ID == reservedPersonaID {
		persona.Name = reservedPersonaName
		if persona.Icon == "" {
			persona.Icon = "♙"
		}
	}
	for index := range s.data.Personas {
		if s.data.Personas[index].ID == persona.ID {
			if persona.Name == "" {
				persona.Name = s.data.Personas[index].Name
			}
			if persona.Icon == "" {
				persona.Icon = s.data.Personas[index].Icon
			}
			// 編號是系統管理的，一般編輯/存檔一律沿用既有編號，不讓前端覆蓋；
			// 萬一舊資料真的沒有編號（理論上不會發生），補一個。
			persona.Code = s.data.Personas[index].Code
			if persona.Code == "" {
				persona.Code = assignNewPersonaCode(s.data.Personas)
			}
			s.data.Personas[index] = persona
			s.data.ActivePersonaID = persona.ID
			_ = s.saveLocked()
			return cloneState(s.data)
		}
	}
	if len(s.data.Personas) >= MaxPersonas {
		return cloneState(s.data)
	}
	// 新增/複製進來的人格一律拿新編號，不沿用外部帶進來的任何舊編號。
	persona.Code = assignNewPersonaCode(s.data.Personas)
	s.data.Personas = append(s.data.Personas, persona)
	s.data.ActivePersonaID = persona.ID
	_ = s.saveLocked()
	return cloneState(s.data)
}

func (s *Service) ReorderPersonas(orderIDs []string) State {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(orderIDs) == 0 {
		return cloneState(s.data)
	}
	// Reorder only applies the requested sequence and appends any omitted
	// personas. It must not force the reserved persona back to index 0.
	byID := make(map[string]Persona, len(s.data.Personas))
	for _, persona := range s.data.Personas {
		byID[persona.ID] = persona
	}

	next := make([]Persona, 0, len(s.data.Personas))
	seen := make(map[string]bool, len(s.data.Personas))
	for _, id := range orderIDs {
		persona, ok := byID[id]
		if !ok || seen[id] {
			continue
		}
		next = append(next, persona)
		seen[id] = true
	}
	for _, persona := range s.data.Personas {
		if !seen[persona.ID] {
			next = append(next, persona)
		}
	}
	if len(next) == len(s.data.Personas) {
		s.data.Personas = next
		s.data.Personas = normalizeReservedPersona(s.data.Personas)
		if len(s.data.Personas) > 0 {
			s.data.ActivePersonaID = s.data.Personas[0].ID
		}
		_ = s.saveLocked()
	}
	return cloneState(s.data)
}

func (s *Service) ExportPersona(personaID, destDir string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	persona, ok := s.findPersonaLocked(personaID)
	if !ok {
		return "", fmt.Errorf("settings: persona %q not found", personaID)
	}
	if strings.TrimSpace(destDir) == "" {
		return "", fmt.Errorf("settings: export directory is empty")
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("settings: create export directory: %w", err)
	}
	payload := PersonaExportPayload{Schema: "ai-console.persona.v1", Persona: persona}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(destDir, safePersonaFileName(persona))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("settings: write persona export: %w", err)
	}
	return path, nil
}

func (s *Service) RemovePersona(personaID string) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if personaID == reservedPersonaID {
		return cloneState(s.data), fmt.Errorf("settings: reserved persona cannot be removed")
	}
	if len(s.data.Personas) <= 1 {
		return cloneState(s.data), fmt.Errorf("settings: at least one persona is required")
	}
	next := s.data.Personas[:0]
	removed := false
	for _, persona := range s.data.Personas {
		if persona.ID == personaID {
			removed = true
			continue
		}
		next = append(next, persona)
	}
	if !removed {
		return cloneState(s.data), fmt.Errorf("settings: persona %q not found", personaID)
	}
	s.data.Personas = normalizeReservedPersona(next)
	if s.data.ActivePersonaID == personaID {
		s.data.ActivePersonaID = s.data.Personas[0].ID
	}
	if isRemovableDefaultPersona(personaID) {
		s.data.RemovedDefaultPersonaIDs = appendUniqueString(s.data.RemovedDefaultPersonaIDs, personaID)
	}
	// Persona removal also clears per-persona assets such as avatar files.
	_ = os.RemoveAll(storage.PersonaRoot(s.root, personaID))
	_ = s.saveLocked()
	return cloneState(s.data), nil
}

func (s *Service) SaveSummaryModelSettings(settings SummaryModelSettings) SummaryModelSettings {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.SummaryModel = normalizeSummaryModelSettings(settings)
	_ = s.saveLocked()
	return s.data.SummaryModel
}

func (s *Service) SummaryModelSettings() SummaryModelSettings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return normalizeSummaryModelSettings(s.data.SummaryModel)
}

// AdapterModelChoices 回傳目前每個 adapter 的 model 偏好（複本，可安全 mutate）。
func (s *Service) AdapterModelChoices() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.data.AdapterModelChoices))
	for k, v := range s.data.AdapterModelChoices {
		out[k] = v
	}
	return out
}

// SaveAdapterModelChoice 寫入單一 adapter 的 model 選擇；空字串等同移除。
func (s *Service) SaveAdapterModelChoice(adapterID, model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.AdapterModelChoices == nil {
		s.data.AdapterModelChoices = map[string]string{}
	}
	if model == "" {
		delete(s.data.AdapterModelChoices, adapterID)
	} else {
		s.data.AdapterModelChoices[adapterID] = model
	}
	_ = s.saveLocked()
}

// EmbeddingConfig 回目前的 embedding 設定（複本）。
func (s *Service) EmbeddingConfig() EmbeddingConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.EmbeddingConfig
}

// SaveEmbeddingConfig 由 UI / binding 呼叫：只更新 ProviderID + ModelID。
// 換 ModelID 會把 Dimension 歸零，下次 backend embed 時重新測量。
// 設了 model 就自動 PickerDismissed=false（讓使用者之後改主意點選了，就不再被視為跳過狀態）。
func (s *Service) SaveEmbeddingConfig(providerID, modelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.data.EmbeddingConfig
	s.data.EmbeddingConfig = EmbeddingConfig{
		ProviderID:      providerID,
		ModelID:         modelID,
		PickerDismissed: false,
	}
	// 維持 Dimension 不變的條件：同個 model；換了就歸零。
	if old.ModelID == modelID && old.ProviderID == providerID {
		s.data.EmbeddingConfig.Dimension = old.Dimension
	}
	_ = s.saveLocked()
}

// DismissEmbeddingPicker：使用者點「跳過」時呼叫；保留之前的 model 設定不動，
// 只設 PickerDismissed=true，避免下次拖檔重複彈 modal。
func (s *Service) DismissEmbeddingPicker() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.EmbeddingConfig.PickerDismissed = true
	_ = s.saveLocked()
}

// ReopenEmbeddingPicker：未來 settings panel 內「重新開啟 picker」的入口；
// 不動 provider/model，僅清 dismissed flag。
func (s *Service) ReopenEmbeddingPicker() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.EmbeddingConfig.PickerDismissed = false
	_ = s.saveLocked()
}

// SaveEmbeddingDimension 給 backend 在第一次成功 embed 後寫入測得的 dimension。
// 故意不接出成 Wails binding——避免 UI 寫錯值汙染後續相容檢查。
func (s *Service) SaveEmbeddingDimension(dim int) {
	if dim <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.EmbeddingConfig.Dimension == dim {
		return
	}
	s.data.EmbeddingConfig.Dimension = dim
	_ = s.saveLocked()
}

func (s *Service) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next, err := s.store.LoadRaw()
	if err != nil {
		return err
	}
	// 零值表示首次啟動——使用預設並寫入磁碟
	if next.ActivePersonaID == "" && len(next.Personas) == 0 {
		ensurePersonaCodes(&s.data)
		_ = s.store.SaveRaw(s.data)
		return nil
	}
	if next.ActivePersonaID == "" {
		next.ActivePersonaID = "persona-a"
	}
	if len(next.Personas) == 0 {
		next.Personas = defaultState().Personas
	}
	next.Personas = removeLegacyDefaultPersonaD(next.Personas)
	next.Personas = normalizeBuiltInPersonaDefaults(next.Personas)
	next.Personas = appendMissingDefaultPersonas(next.Personas, next.RemovedDefaultPersonaIDs)
	next.Personas = normalizeReservedPersona(next.Personas)
	next.SummaryModel = normalizeSummaryModelSettings(next.SummaryModel)
	next.ControlSeal = next.ControlSeal.Normalize()
	if next.ActivePersonaID == "persona-d" {
		next.ActivePersonaID = reservedPersonaID
	}
	// Panel 欄位由 UISettingsService 管理，load 時忽略舊 JSON 中的 panel 資料。
	next.Panel = PanelSettings{}
	if ensurePersonaCodes(&next) {
		_ = s.store.SaveRaw(next)
	}
	s.data = next
	return nil
}

func normalizeBuiltInPersonaDefaults(personas []Persona) []Persona {
	for index := range personas {
		switch personas[index].ID {
		case "persona-b":
			// Migrate only untouched default slots; user-authored persona text wins.
			if personas[index].Name == "" || personas[index].Name == "人格 B" || personas[index].Name == "厭世叔" {
				personas[index].Name = "厭世大叔"
			}
			if personas[index].Identity == "" {
				personas[index].Identity = "厭世社畜但帥帥的粗眉硬漢助手"
			}
		case "persona-c":
			if personas[index].Name == "" || personas[index].Name == "人格 C" {
				personas[index].Name = "秘書小妹"
			}
			if personas[index].Identity == "" {
				personas[index].Identity = "聰明俐落的秘書小妹助手"
			}
		case "persona-d":
			applyPersonaDefaults(&personas[index], "persona-d")
		case "persona-e":
			applyPersonaDefaults(&personas[index], "persona-e")
		}
	}
	return personas
}

func (s *Service) saveLocked() error {
	return s.store.SaveRaw(s.data)
}

func defaultState() State {
	return State{
		// Panel 欄位不在此初始化——面板設定由 UISettingsService 管理。
		// State.Panel 僅作為 DTO 供 app.go 組合回傳給前端。
		ActivePersonaID: reservedPersonaID,
		Personas:        clonePersonas(defaultPersonas),
		SummaryModel:    defaultSummaryModelSettings(),
		ControlSeal:     controlseal.DefaultSettings(),
	}
}

func defaultSummaryModelSettings() SummaryModelSettings {
	return SummaryModelSettings{
		Source:    "cli_adapter",
		ModelID:   "current",
		AlwaysUse: false,
	}
}

func normalizeSummaryModelSettings(settings SummaryModelSettings) SummaryModelSettings {
	if settings.Source == "" {
		settings.Source = "cli_adapter"
	}
	if settings.ModelID == "" {
		settings.ModelID = "current"
	}
	return settings
}

func normalizeReservedPersona(personas []Persona) []Persona {
	// Ensure the reserved persona exists and keeps its name/default identity, but
	// preserve its current array position so the UI can choose a different main
	// persona by moving another card to the first slot.
	if len(personas) == 0 {
		return defaultState().Personas
	}
	reservedIndex := -1
	for index := range personas {
		if personas[index].ID == reservedPersonaID {
			reservedIndex = index
			break
		}
	}
	if reservedIndex < 0 {
		personas = append([]Persona{{ID: reservedPersonaID, Icon: "♙", Identity: "酷酷的男性狼犬獸人助手"}}, personas...)
		reservedIndex = 0
	}
	personas[reservedIndex].Name = reservedPersonaName
	if personas[reservedIndex].Icon == "" {
		personas[reservedIndex].Icon = "♙"
	}
	if personas[reservedIndex].Identity == "" {
		personas[reservedIndex].Identity = "酷酷的男性狼犬獸人助手"
	}
	return personas
}

func cloneState(state State) State {
	state.Personas = append([]Persona(nil), state.Personas...)
	state.RemovedDefaultPersonaIDs = append([]string(nil), state.RemovedDefaultPersonaIDs...)
	if state.AdapterModelChoices != nil {
		m := make(map[string]string, len(state.AdapterModelChoices))
		for k, v := range state.AdapterModelChoices {
			m[k] = v
		}
		state.AdapterModelChoices = m
	}
	return state
}

func clonePersonas(personas []Persona) []Persona {
	return append([]Persona(nil), personas...)
}

func appendMissingDefaultPersonas(personas []Persona, removedIDs []string) []Persona {
	known := make(map[string]bool, len(personas))
	for _, persona := range personas {
		known[persona.ID] = true
	}
	removed := make(map[string]bool, len(removedIDs))
	for _, id := range removedIDs {
		removed[id] = true
	}
	for _, persona := range defaultPersonas {
		if persona.ID == reservedPersonaID || known[persona.ID] || removed[persona.ID] {
			continue
		}
		// Default personas are seeds, not locks; RemovePersona records a tombstone.
		personas = append(personas, persona)
		known[persona.ID] = true
	}
	return personas
}

func applyPersonaDefaults(persona *Persona, defaultID string) {
	for _, seed := range defaultPersonas {
		if seed.ID != defaultID {
			continue
		}
		if defaultID == "persona-d" {
			applyDefaultPersonaDMigration(persona, seed)
			return
		}
		if defaultID == "persona-e" {
			applyDefaultPersonaEMigration(persona, seed)
			return
		}
		if persona.Name == "" {
			persona.Name = seed.Name
		}
		if persona.Icon == "" {
			persona.Icon = seed.Icon
		}
		if persona.Identity == "" {
			persona.Identity = seed.Identity
		}
		if persona.Personality == "" {
			persona.Personality = seed.Personality
		}
		return
	}
}

func applyDefaultPersonaDMigration(persona *Persona, seed Persona) {
	applyDefaultPersonaTextMigration(persona, seed,
		[]string{"人格 D", "規則警察", "警察桂澤", "Rule Police", "Officer Reggie Law"},
		[]string{
			"循規蹈矩、嚴格守序、看到違規就會說教的警察助手",
			"循規蹈矩的警察助手；「桂澤」聽起來像規則，看到流程被跳過就會立刻吹哨。",
			"A strict rule-following police assistant who lectures when rules are bent",
			"Rules-first police assistant who lectures when rules are bent",
		},
		[]string{
			"重視規則、流程與安全，回答會先提醒限制與責任，語氣像警察一樣嚴肅但可靠。",
			"把規則、流程與安全放第一，回答會先提醒限制、責任與風險；嚴肅但可靠，說教時也有一點冷面幽默。",
			"Prioritizes rules, process, and safety. Replies first with constraints and responsibility, stern like a police officer but reliable.",
		})
}

func applyDefaultPersonaEMigration(persona *Persona, seed Persona) {
	applyDefaultPersonaTextMigration(persona, seed,
		[]string{"東春巫女", "Touharu Miko", "Miko Touharu"},
		[]string{
			"白髮馬尾、棕眼曬黑、直覺敏銳的巫女助手",
			"白髮馬尾、棕眼曬黑、直覺敏銳的東春巫女助手",
			"A tan white-ponytailed miko assistant with sharp intuition",
			"Tan-skinned white-ponytailed miko assistant with sharp intuition",
		},
		[]string{
			"傲嬌、淘氣又直覺敏銳，嘴上不饒人，但能很快察覺問題不對勁。",
			"傲嬌又淘氣，嘴上不饒人，但能很快察覺問題不對勁，像春雷一樣先提醒你。",
			"Tsundere and mischievous, with quick instincts and a habit of noticing when something feels off.",
			"Tsundere and mischievous, but highly intuitive and quick to sense when something is wrong.",
		})
}

func applyDefaultPersonaTextMigration(persona *Persona, seed Persona, legacyNames, legacyIdentities, legacyPersonalities []string) {
	if legacyDefaultText(persona.Name, legacyNames) {
		persona.Name = seed.Name
	}
	if persona.Icon == "" {
		persona.Icon = seed.Icon
	}
	if legacyDefaultText(persona.Identity, legacyIdentities) {
		persona.Identity = seed.Identity
	}
	if legacyDefaultText(persona.Personality, legacyPersonalities) {
		persona.Personality = seed.Personality
	}
}

func legacyDefaultText(value string, legacyValues []string) bool {
	text := strings.TrimSpace(value)
	if text == "" {
		return true
	}
	for _, legacy := range legacyValues {
		if text == legacy {
			return true
		}
	}
	return false
}

func isRemovableDefaultPersona(personaID string) bool {
	if personaID == reservedPersonaID {
		return false
	}
	for _, persona := range defaultPersonas {
		if persona.ID == personaID {
			return true
		}
	}
	return false
}

func appendUniqueString(values []string, value string) []string {
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}

func (s *Service) findPersonaLocked(personaID string) (Persona, bool) {
	if personaID == "" {
		personaID = s.data.ActivePersonaID
	}
	for _, persona := range s.data.Personas {
		if persona.ID == personaID {
			return persona, true
		}
	}
	return Persona{}, false
}

func safePersonaFileName(persona Persona) string {
	base := strings.TrimSpace(persona.Name)
	if base == "" {
		base = persona.ID
	}
	base = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, base)
	base = strings.Trim(base, ". ")
	if base == "" {
		base = "persona"
	}
	return fmt.Sprintf("%s_%s.json", base, persona.ID)
}

func removeLegacyDefaultPersonaD(personas []Persona) []Persona {
	next := personas[:0]
	for _, persona := range personas {
		if persona.ID == "persona-d" &&
			persona.Name == "人格 D" &&
			persona.Icon == "◇" &&
			persona.AvatarURL == "" &&
			persona.Identity == "" &&
			persona.ReplyStrategy == "" &&
			persona.RoleStrength == "" &&
			persona.Personality == "" &&
			persona.Scenario == "" &&
			persona.Description == "" {
			continue
		}
		next = append(next, persona)
	}
	return next
}
