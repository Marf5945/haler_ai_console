// app_commemorative_photo.go —「galgame 紀念照」機制的 app 層整合。
//
// 機制總覽（沿用既有「LLM 只提議、App 才執行」的 action 協定）：
//  1. 聊一段時間後，人格在 context 看到提示，可主動輸出
//     紀念照ㄌ<場景描述>ㄌ確認 → maybeCommemorativePhoto 攔截，回一張「待確認卡」
//     （NeedsUser=true），不直接產圖。
//  2. 使用者在前端點「拍下這一刻」→ 呼叫 ConfirmCommemorativePhoto(scene, digest)，
//     App 用啟用人格 + 場景組提示詞，呼叫本機 ComfyUI 產圖，落地相冊，
//     並寫一筆記憶錨點，讓之後的對話「記得這張照片的產生背景」。
//  3. 相冊以 ListAlbumPhotos / AlbumPhotoImage / SetAlbumPhotoCaption /
//     DeleteAlbumPhoto 等綁定供前端展示與管理。
//
// egress：ComfyUI 走 urlsafe.PolicyLocalLLM（只放行 loopback、擋 LAN/metadata）。
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ui_console/adapter/imagegen"
	"ui_console/data/album"
	"ui_console/data/memory"
	"ui_console/data/storage"
	"ui_console/domain/keepsake_recall"
	"ui_console/internal/urlsafe"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/actionchain"
	"ui_console/shared/settings"
)

const (
	commemorativePhotoActionLabel = "紀念照"
	keepsakeProjectID             = "default"
	keepsakeGenTimeout            = 4 * time.Minute
)

// ── 設定（存在相冊目錄旁，env 可覆寫）──

// KeepsakeConfig 是紀念照功能的可調設定。
type KeepsakeConfig struct {
	Mode string `json:"mode"` // "comfyui"（本機）或 "cloud"（雲端）

	// 本機 ComfyUI
	ComfyUIURL string `json:"comfyui_url"` // 例 http://127.0.0.1:8188
	Checkpoint string `json:"checkpoint"`  // ckpt_name

	// 雲端（OpenAI images 相容）
	CloudEndpoint string `json:"cloud_endpoint"` // 完整 URL，例 https://api.openai.com/v1/images/generations
	CloudAPIKey   string `json:"cloud_api_key"`
	CloudModel    string `json:"cloud_model"` // 例 dall-e-3 / 服務商模型名

	// 共用產圖參數
	StylePreset  string `json:"style_preset"`  // 全域畫風前綴，可空（用預設動漫風）
	Negative     string `json:"negative"`      // 負向詞，可空
	Width        int    `json:"width"`         // 0 用預設
	Height       int    `json:"height"`        // 0 用預設
	Steps        int    `json:"steps"`         // 0 用預設（僅 ComfyUI 有效）
	SuggestEvery int    `json:"suggest_every"` // 每聊幾輪可提議一次（前端/prompt 節流參考）
}

const (
	keepsakeModeComfyUI = "comfyui"
	keepsakeModeCloud   = "cloud"
)

func defaultKeepsakeConfig() KeepsakeConfig {
	return KeepsakeConfig{
		Mode:         keepsakeModeComfyUI,
		ComfyUIURL:   "http://127.0.0.1:8188",
		Checkpoint:   "",
		StylePreset:  "",
		Negative:     "",
		Width:        512,
		Height:       768,
		Steps:        24,
		SuggestEvery: 12,
	}
}

func keepsakeConfigPath() string {
	root := storage.ProjectRoot(appDataRoot(), keepsakeProjectID)
	return filepath.Join(root, "album", "keepsake_config.json")
}

// loadKeepsakeConfig 讀設定並套用 env 覆寫與預設。
func loadKeepsakeConfig() KeepsakeConfig {
	cfg := defaultKeepsakeConfig()
	store := storage.NewJSONStore[KeepsakeConfig](keepsakeConfigPath())
	if loaded, err := store.Load(); err == nil {
		if m := strings.TrimSpace(loaded.Mode); m == keepsakeModeComfyUI || m == keepsakeModeCloud {
			cfg.Mode = m
		}
		if strings.TrimSpace(loaded.ComfyUIURL) != "" {
			cfg.ComfyUIURL = loaded.ComfyUIURL
		}
		if strings.TrimSpace(loaded.Checkpoint) != "" {
			cfg.Checkpoint = loaded.Checkpoint
		}
		cfg.CloudEndpoint = loaded.CloudEndpoint
		cfg.CloudAPIKey = loaded.CloudAPIKey
		cfg.CloudModel = loaded.CloudModel
		cfg.StylePreset = loaded.StylePreset
		cfg.Negative = loaded.Negative
		if loaded.Width > 0 {
			cfg.Width = loaded.Width
		}
		if loaded.Height > 0 {
			cfg.Height = loaded.Height
		}
		if loaded.Steps > 0 {
			cfg.Steps = loaded.Steps
		}
		if loaded.SuggestEvery > 0 {
			cfg.SuggestEvery = loaded.SuggestEvery
		}
	}
	if v := strings.TrimSpace(os.Getenv("AI_CONSOLE_COMFYUI_URL")); v != "" {
		cfg.ComfyUIURL = v
	}
	if v := strings.TrimSpace(os.Getenv("AI_CONSOLE_COMFYUI_CKPT")); v != "" {
		cfg.Checkpoint = v
	}
	return cfg
}

// GetKeepsakeConfig 綁定：前端讀目前設定。
func (a *App) GetKeepsakeConfig() KeepsakeConfig { return loadKeepsakeConfig() }

// SaveKeepsakeConfig 綁定：前端存設定。
func (a *App) SaveKeepsakeConfig(cfg KeepsakeConfig) error {
	store := storage.NewJSONStore[KeepsakeConfig](keepsakeConfigPath())
	return store.Save(cfg)
}

// keepsakeReady 判斷目前設定是否足以產圖（決定是否讓人格主動提議）。
func keepsakeReady(cfg KeepsakeConfig) bool {
	if cfg.Mode == keepsakeModeCloud {
		return strings.TrimSpace(cfg.CloudEndpoint) != "" &&
			strings.TrimSpace(cfg.CloudAPIKey) != "" &&
			strings.TrimSpace(cfg.CloudModel) != ""
	}
	return strings.TrimSpace(cfg.Checkpoint) != ""
}

// buildKeepsakeAdapter 依 mode 建立對應的產圖 adapter，並回傳 model 提示字串。
func buildKeepsakeAdapter(cfg KeepsakeConfig) (imagegen.Adapter, string, error) {
	if cfg.Mode == keepsakeModeCloud {
		if !keepsakeReady(cfg) {
			return nil, "", fmt.Errorf("雲端產圖未設定完整（需端點 / 金鑰 / 模型）")
		}
		client := urlsafe.NewSafeClient(urlsafe.PolicyCloudAPI, "keepsake_imagegen_cloud", keepsakeGenTimeout)
		return imagegen.NewCloudAdapter(cfg.CloudEndpoint, cfg.CloudAPIKey, cfg.CloudModel, client), cfg.CloudModel, nil
	}
	// 預設 ComfyUI（本機）
	if strings.TrimSpace(cfg.Checkpoint) == "" {
		return nil, "", fmt.Errorf("尚未設定 ComfyUI checkpoint（請在紀念照設定填入模型名稱）")
	}
	client := urlsafe.NewSafeClient(urlsafe.PolicyLocalLLM, "keepsake_imagegen", keepsakeGenTimeout)
	return imagegen.NewComfyUIAdapter(cfg.ComfyUIURL, client, cfg.Checkpoint, "ai-console-keepsake"), cfg.Checkpoint, nil
}

// ── 動作攔截：紀念照ㄌ<場景>ㄌ確認 ──

// AlbumPhotoView 是回給前端的相冊一筆（不含圖位元組，圖另以 AlbumPhotoImage 取）。
type AlbumPhotoView struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	CreatedAt   string `json:"createdAt"`
	PersonaName string `json:"personaName"`
	Scene       string `json:"scene"`
	Caption     string `json:"caption"`
	MemoryTag   string `json:"memoryTag"`
	Seed        int64  `json:"seed"`
	Model       string `json:"model"`
}

func toAlbumView(p album.Photo) AlbumPhotoView {
	return AlbumPhotoView{
		ID:          p.ID,
		Code:        p.Code,
		CreatedAt:   p.CreatedAt,
		PersonaName: p.PersonaName,
		Scene:       p.Scene,
		Caption:     p.Caption,
		MemoryTag:   p.MemoryTag,
		Seed:        p.Seed,
		Model:       p.Model,
	}
}

// maybeCommemorativePhoto 攔截 紀念照 動作；非此動作回 handled=false。
// 注意：這裡「不」直接產圖，而是回一張待確認卡（NeedsUser=true），
// 交由使用者在前端點「拍下這一刻」後呼叫 ConfirmCommemorativePhoto。
func (a *App) maybeCommemorativePhoto(action, target, traceID string) (bool, skill_step.CLIResponse) {
	if strings.TrimSpace(action) != commemorativePhotoActionLabel {
		return false, skill_step.CLIResponse{}
	}
	scene := strings.TrimSpace(target)
	persona := a.getActivePersona()
	name := persona.Name
	if name == "" {
		name = "我"
	}
	text := fmt.Sprintf("要不要把這一刻拍下來當紀念？%s想和你留一張：「%s」。", name, scene)
	if scene == "" {
		text = fmt.Sprintf("要不要把這一刻拍下來當紀念？%s想和你留一張合照。", name)
	}
	return true, skill_step.CLIResponse{
		Action:    commemorativePhotoActionLabel,
		Target:    scene,
		Next:      "確認",
		Text:      text,
		NeedsUser: true,
	}
}

// ── 確認後實際產圖 ──

// ConfirmCommemorativePhoto 綁定：使用者確認後產圖、落地相冊、寫記憶錨點。
// scene 為場景描述（可沿用提議時的 target）；contextDigest 為產生背景（對話摘要快照，可空）。
func (a *App) ConfirmCommemorativePhoto(scene, contextDigest string) (AlbumPhotoView, error) {
	scene = strings.TrimSpace(scene)
	cfg := loadKeepsakeConfig()

	adapter, modelHint, err := buildKeepsakeAdapter(cfg)
	if err != nil {
		return AlbumPhotoView{}, err
	}

	persona := a.getActivePersona()
	positive := imagegen.BuildScenePrompt(imagegen.SceneInput{
		PersonaIdentity:    personaAppearance(persona),
		PersonaPersonality: persona.Personality,
		Scene:              scene,
		StylePreset:        cfg.StylePreset,
	})

	ctx, cancel := context.WithTimeout(context.Background(), keepsakeGenTimeout)
	defer cancel()

	result, err := adapter.Generate(ctx, imagegen.Request{
		Positive: positive,
		Negative: cfg.Negative,
		Width:    cfg.Width,
		Height:   cfg.Height,
		Steps:    cfg.Steps,
		Seed:     -1,
		Model:    modelHint,
	})
	if err != nil {
		return AlbumPhotoView{}, fmt.Errorf("產圖失敗: %w", err)
	}

	memTag := a.anchorKeepsakeMemory(persona, scene, contextDigest)

	store := album.NewStoreForProject(appDataRoot(), keepsakeProjectID)
	saved, err := store.Add(album.Photo{
		PersonaID:     persona.ID,
		PersonaName:   persona.Name,
		Scene:         scene,
		ContextDigest: strings.TrimSpace(contextDigest),
		MemoryTag:     memTag,
		Prompt:        positive,
		Negative:      firstNonEmptyStr(cfg.Negative, imagegen.DefaultNegative),
		Seed:          result.Seed,
		Model:         result.Model,
		Width:         cfg.Width,
		Height:        cfg.Height,
	}, result.PNG)
	if err != nil {
		return AlbumPhotoView{}, err
	}
	return toAlbumView(saved), nil
}

// recallKeepsakePhoto 讓上方互動彈窗回話前，看看使用者這句話有沒有講到這個
// 人格已保留的某張紀念照；有的話回傳（可能要附的圖片, 要塞進 prompt 的描述）。
// 只有 adapter/model 通過 brainIsMultimodal 判斷可以讀圖時才會真的附圖，
// 否則 imgs 為空、note 仍然會給文字描述，讓非 vision 模型也能「靠描述回想」。
// 這段回憶是永久的：只要照片沒被使用者刪掉（DeleteAlbumPhoto），關程式、
// 切人格都不會讓它消失——跟上方互動另一套「30 句、關彈窗即清空」的短期
// 記憶（shared/inspectormemory）是完全不同的兩種保留策略。
func (a *App) recallKeepsakePhoto(personaID, userText, adapterID, model string) ([]composerImage, string) {
	personaID = strings.TrimSpace(personaID)
	userText = strings.TrimSpace(userText)
	if personaID == "" || userText == "" {
		return nil, ""
	}
	store := album.NewStoreForProject(appDataRoot(), keepsakeProjectID)
	photos, err := store.List()
	if err != nil || len(photos) == 0 {
		return nil, ""
	}
	match := keepsake_recall.Find(photos, personaID, userText)
	if match == nil {
		return nil, ""
	}

	hasImage := brainIsMultimodal(adapterID, model)
	var imgs []composerImage
	if hasImage {
		if raw, readErr := os.ReadFile(store.ImagePath(*match)); readErr == nil && len(raw) > 0 {
			imgs = []composerImage{{MIME: "image/png", DataB64: base64.StdEncoding.EncodeToString(raw)}}
		} else {
			hasImage = false // 讀檔失敗就退回純文字描述，不假裝有附圖
		}
	}
	return imgs, keepsake_recall.DescriptionNote(*match, hasImage)
}

// anchorKeepsakeMemory 把「產生了一張紀念照」寫進記憶管線，讓之後對話記得這段背景。
// 回傳寫入後可被 展開 撈回的記憶錨點標籤（盡力而為；失敗回空字串、不擋產圖）。
func (a *App) anchorKeepsakeMemory(persona settings.Persona, scene, digest string) string {
	root := storage.ProjectRoot(appDataRoot(), keepsakeProjectID)
	pipeline := memory.NewPipeline(root)

	var b strings.Builder
	b.WriteString("[紀念照] 與「")
	b.WriteString(persona.Name)
	b.WriteString("」留下一張紀念照。")
	if scene != "" {
		b.WriteString("場景：")
		b.WriteString(scene)
		b.WriteString("。")
	}
	if d := strings.TrimSpace(digest); d != "" {
		b.WriteString("當時的對話背景：")
		b.WriteString(d)
	}
	if _, err := pipeline.AppendTalkEntry("紀念照", b.String()); err != nil {
		return ""
	}
	// 記憶標籤需由摘要管線指派，這裡先不強取；相冊已留 ContextDigest 作為背景。
	return ""
}

// ── 相冊綁定 ──

// ListAlbumPhotos 綁定：回所有紀念照（最新在前）。
func (a *App) ListAlbumPhotos() ([]AlbumPhotoView, error) {
	store := album.NewStoreForProject(appDataRoot(), keepsakeProjectID)
	photos, err := store.List()
	if err != nil {
		return nil, err
	}
	views := make([]AlbumPhotoView, 0, len(photos))
	for _, p := range photos {
		views = append(views, toAlbumView(p))
	}
	return views, nil
}

// AlbumPhotoImage 綁定：回某張紀念照的 data URL（base64），供前端 <img src> 直接顯示。
func (a *App) AlbumPhotoImage(id string) (string, error) {
	store := album.NewStoreForProject(appDataRoot(), keepsakeProjectID)
	p, ok, err := store.Get(id)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("找不到紀念照 %s", id)
	}
	data, err := os.ReadFile(store.ImagePath(p))
	if err != nil {
		return "", fmt.Errorf("讀取紀念照圖檔失敗: %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data), nil
}

// SetAlbumPhotoCaption 綁定：更新某張的使用者說明。
func (a *App) SetAlbumPhotoCaption(id, caption string) (AlbumPhotoView, error) {
	store := album.NewStoreForProject(appDataRoot(), keepsakeProjectID)
	p, err := store.SetCaption(id, caption)
	if err != nil {
		return AlbumPhotoView{}, err
	}
	return toAlbumView(p), nil
}

// DeleteAlbumPhoto 綁定：刪除一張紀念照。
func (a *App) DeleteAlbumPhoto(id string) error {
	store := album.NewStoreForProject(appDataRoot(), keepsakeProjectID)
	return store.Delete(id)
}

// ── 人格頭像：共用同一套產圖介面 ──

// GenerateAvatarViaImageGen 綁定：用「同一套產圖設定」(KeepsakeConfig 的 ComfyUI/雲端)
// 生成人格頭像，存成該人格的靜態頭像。讓使用者只需設定**一個**產圖接口，
// 紀念照與頭像共用，不必各自設定 API。
//
// state 為頭像表情狀態（idle/happy/thinking/working/warning/blocked/sleepy），可空。
func (a *App) GenerateAvatarViaImageGen(personaID, state string) error {
	cfg := loadKeepsakeConfig()
	adapter, modelHint, err := buildKeepsakeAdapter(cfg)
	if err != nil {
		return err
	}
	persona := a.personaByID(personaID)

	scene := "portrait avatar, head and shoulders, centered, simple clean background, icon"
	if expr := avatarStateExpression(state); expr != "" {
		scene += ", " + expr
	}
	positive := imagegen.BuildScenePrompt(imagegen.SceneInput{
		PersonaIdentity:    personaAppearance(persona),
		PersonaPersonality: persona.Personality,
		Scene:              scene,
		StylePreset:        cfg.StylePreset,
	})

	ctx, cancel := context.WithTimeout(context.Background(), keepsakeGenTimeout)
	defer cancel()
	res, err := adapter.Generate(ctx, imagegen.Request{
		Positive: positive,
		Negative: cfg.Negative,
		Width:    512,
		Height:   512,
		Steps:    cfg.Steps,
		Seed:     -1,
		Model:    modelHint,
	})
	if err != nil {
		return fmt.Errorf("頭像產圖失敗: %w", err)
	}
	// 存成靜態頭像（SaveStaticAvatar 會縮到 256 並把 provider 設為 static_image）。
	if err := a.avatarService.SaveStaticAvatar(personaID, "image/png", res.PNG, nil, 256); err != nil {
		return fmt.Errorf("儲存頭像失敗: %w", err)
	}
	return nil
}

// personaByID 依 ID 取人格；找不到回退啟用人格。
func (a *App) personaByID(personaID string) settings.Persona {
	personaID = strings.TrimSpace(personaID)
	if personaID != "" && a.settingsService != nil {
		for _, p := range a.settingsService.State().Personas {
			if p.ID == personaID {
				return p
			}
		}
	}
	return a.getActivePersona()
}

// avatarStateExpression 把頭像狀態觸發器轉成一句表情描述（給產圖用）。
func avatarStateExpression(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "happy":
		return "smiling warmly"
	case "thinking":
		return "looking thoughtful"
	case "working":
		return "focused expression"
	case "warning":
		return "slightly serious expression"
	case "blocked":
		return "confused but polite"
	case "sleepy":
		return "drowsy, half-closed eyes"
	case "idle", "":
		return "neutral calm expression"
	default:
		return "neutral calm expression"
	}
}

// ── 小工具 ──

// personaAppearance 從人格欄位萃取「外觀/身分」描述當作提示詞主體。
func personaAppearance(p settings.Persona) string {
	if s := strings.TrimSpace(p.Identity); s != "" {
		return s
	}
	if s := strings.TrimSpace(p.Description); s != "" {
		return s
	}
	return strings.TrimSpace(p.Name)
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// keepsakeComposerHint 只有在已設定 ComfyUI checkpoint 時才回提示字串，
// 避免人格提議了卻因未設定產圖器而失敗。供 buildMainComposerPrompt 注入。
func (a *App) keepsakeComposerHint() string {
	if !keepsakeReady(loadKeepsakeConfig()) {
		return ""
	}
	return keepsakeSuggestionHint()
}

// keepsakeSuggestionHint 是注入路由/composer prompt 的提示：聊到適當時機可提議紀念照。
// 由 prompt 組裝端取用（見設計文件「接線」段）。
func keepsakeSuggestionHint() string {
	return fmt.Sprintf(
		"當對話累積到一個有情感意義的時刻（例如聊了一段時間、完成一件事、氣氛溫馨或告一段落），"+
			"你可以『主動』提議拍一張紀念照：輸出 %s%s<一句話場景描述>%s%s。"+
			"場景描述要具體（地點/氛圍/兩人在做什麼），不要每輪都提議；使用者沒答應就別重複。",
		commemorativePhotoActionLabel, actionchain.Separator, actionchain.Separator, actionchain.StandbyNext,
	)
}
