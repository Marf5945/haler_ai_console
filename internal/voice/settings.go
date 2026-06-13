package voice

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"ui_console/data/storage"
	"ui_console/shared/executil"
)

const (
	LanguageFollowApp = "follow_app"
	LanguageAuto      = "auto"
	LanguageManual    = "manual"
	ManagedModelFile  = "ggml-base.bin"
	// BundledRunnerSHA256 是 macOS 內建 whisper-cli(whisper.cpp,MIT 授權)的雜湊。
	BundledRunnerSHA256 = "c01e6dea1165e459d73193a341db9f31e83e4dc01b8b17e169a00b2d396246fa"
	// BundledRunnerSHA256Windows:Windows 版 whisper-cli.exe 的雜湊。
	// whisper.cpp 官方提供 Windows 建置;打包進 resources 後把雜湊填入此處即可啟用。
	// 空字串表示尚未打包,語音轉文字在 Windows 會明確回報暫不支援。
	BundledRunnerSHA256Windows = ""
)

// ManagedRunnerFile 是內建語音執行檔的檔名,依平台加上副檔名。
var ManagedRunnerFile = managedRunnerFileName()

func managedRunnerFileName() string {
	if runtime.GOOS == "windows" {
		return "whisper-cli.exe"
	}
	return "whisper-cli"
}

// bundledRunnerSHA256 回傳目前平台內建 runner 的預期雜湊;空字串表示此平台尚未內建。
func bundledRunnerSHA256() string {
	if runtime.GOOS == "windows" {
		return BundledRunnerSHA256Windows
	}
	return BundledRunnerSHA256
}

type Settings struct {
	DebugMode      bool   `json:"debugMode"`
	LanguageMode   string `json:"languageMode"`
	ManualLanguage string `json:"manualLanguage"`
	CommandMode    bool   `json:"commandMode"`
	WhisperBinPath string `json:"whisperBinPath"`
	ModelPath      string `json:"modelPath"`
}

type State struct {
	Settings         Settings `json:"settings"`
	WhisperBinPath   string   `json:"whisperBinPath"`
	ModelPath        string   `json:"modelPath"`
	ManagedModelPath string   `json:"managedModelPath"`
	WhisperAvailable bool     `json:"whisperAvailable"`
	ModelAvailable   bool     `json:"modelAvailable"`
	Language         string   `json:"language"`
	Status           string   `json:"status"`
}

type Service struct {
	mu        sync.Mutex
	root      string
	cwd       string
	program   string
	resources string
	store     *storage.JSONStore[Settings]
	current   Settings
}

func NewService(root, cwd string, program string, resources string) *Service {
	svc := &Service{
		root:      root,
		cwd:       cwd,
		program:   program,
		resources: resources,
		store:     storage.NewJSONStore[Settings](filepath.Join(root, "data", "preferences", "voice_settings.json")),
		current:   defaultSettings(),
	}
	if loaded, err := svc.store.Load(); err == nil && !isZeroSettings(loaded) {
		svc.current = normalizeSettings(loaded)
	}
	return svc
}

func defaultSettings() Settings {
	return Settings{
		LanguageMode: LanguageAuto,
	}
}

func isZeroSettings(s Settings) bool {
	return !s.DebugMode &&
		s.LanguageMode == "" &&
		s.ManualLanguage == "" &&
		!s.CommandMode &&
		s.WhisperBinPath == "" &&
		s.ModelPath == ""
}

func normalizeSettings(s Settings) Settings {
	if s.LanguageMode == "" {
		s.LanguageMode = LanguageAuto
	}
	switch s.LanguageMode {
	case LanguageFollowApp, LanguageAuto, LanguageManual:
	default:
		s.LanguageMode = LanguageAuto
	}
	s.ManualLanguage = normalizeWhisperLanguage(s.ManualLanguage)
	return s
}

func normalizeWhisperLanguage(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	switch lang {
	case "zh-tw", "zh-hant", "zh-cn", "zh-hans":
		return "zh"
	case "en-us", "en-gb":
		return "en"
	case "ja-jp":
		return "ja"
	case "ko-kr":
		return "ko"
	case "":
		return ""
	default:
		return lang
	}
}

func (s *Service) Get(panelLanguage string) State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stateLocked(panelLanguage)
}

func (s *Service) Save(next Settings, panelLanguage string) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = normalizeSettings(next)
	if err := s.store.Save(s.current); err != nil {
		return s.stateLocked(panelLanguage), err
	}
	return s.stateLocked(panelLanguage), nil
}

func (s *Service) stateLocked(panelLanguage string) State {
	binPath := firstExistingExecutable(s.resolveBinCandidates())
	modelPath := firstExistingFile(s.resolveModelCandidates())
	modelInvalid := false
	if modelPath != "" {
		if err := verifyManagedModelFile(modelPath); err != nil {
			modelInvalid = true
			modelPath = ""
		}
	}
	lang := s.resolveLanguageLocked(panelLanguage)
	state := State{
		Settings:         s.current,
		WhisperBinPath:   binPath,
		ModelPath:        modelPath,
		ManagedModelPath: s.ManagedModelPath(),
		WhisperAvailable: binPath != "",
		ModelAvailable:   modelPath != "",
		Language:         lang,
	}
	switch {
	case modelInvalid:
		state.Status = "invalid_model"
	case !state.WhisperAvailable && !state.ModelAvailable:
		state.Status = "missing_binary_and_model"
	case !state.WhisperAvailable:
		state.Status = "missing_binary"
	case !state.ModelAvailable:
		state.Status = "missing_model"
	default:
		state.Status = "ready"
	}
	return state
}

func (s *Service) resolveLanguageLocked(panelLanguage string) string {
	switch s.current.LanguageMode {
	case LanguageAuto:
		return "auto"
	case LanguageManual:
		if s.current.ManualLanguage != "" {
			return s.current.ManualLanguage
		}
		return "auto"
	default:
		return WhisperLanguageFromPanel(panelLanguage)
	}
}

func WhisperLanguageFromPanel(panelLanguage string) string {
	value := strings.ToLower(strings.TrimSpace(panelLanguage))
	switch {
	case strings.Contains(value, "英文") || strings.Contains(value, "english") || strings.HasPrefix(value, "en"):
		return "en"
	case strings.Contains(value, "日文") || strings.Contains(value, "日本") || strings.HasPrefix(value, "ja"):
		return "ja"
	case strings.Contains(value, "韓") || strings.HasPrefix(value, "ko"):
		return "ko"
	case strings.Contains(value, "繁") || strings.Contains(value, "簡") || strings.Contains(value, "中文") || strings.HasPrefix(value, "zh"):
		return "zh"
	default:
		return "auto"
	}
}

func (s *Service) resolveBinCandidates() []string {
	path, err := s.ensureManagedRunner()
	if err != nil {
		return nil
	}
	return []string{path}
}

func (s *Service) resolveModelCandidates() []string {
	return []string{s.ManagedModelPath()}
}

func (s *Service) ManagedModelPath() string {
	return filepath.Join(s.managedBase(), "voice", "models", ManagedModelFile)
}

func (s *Service) ManagedRunnerPath() string {
	return filepath.Join(s.managedBase(), "voice", ManagedRunnerFile)
}

func (s *Service) managedBase() string {
	base := s.program
	if strings.TrimSpace(base) == "" {
		base = s.cwd
	}
	if filepath.Ext(base) == ".app" {
		base = filepath.Dir(base)
	}
	return base
}

func (s *Service) bundledRunnerPath() string {
	if s.resources == "" {
		return ""
	}
	return filepath.Join(s.resources, "voice", ManagedRunnerFile)
}

func (s *Service) ensureManagedRunner() (string, error) {
	expectedSHA := bundledRunnerSHA256()
	if expectedSHA == "" {
		return "", fmt.Errorf("voice: 此平台(%s)尚未內建語音執行檔,語音轉文字暫不支援", runtime.GOOS)
	}
	source := s.bundledRunnerPath()
	if source == "" {
		return "", fmt.Errorf("voice: bundled runner missing")
	}
	if err := verifyFileSHA256(source, expectedSHA); err != nil {
		return "", fmt.Errorf("voice: bundled runner checksum: %w", err)
	}
	target := s.ManagedRunnerPath()
	if filepath.Clean(source) == filepath.Clean(target) {
		return source, nil
	}
	if err := verifyFileSHA256(target, expectedSHA); err == nil {
		_ = os.Chmod(target, 0o700)
		return target, nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return "", fmt.Errorf("voice: create runner dir: %w", err)
	}
	tmp := target + ".download"
	_ = os.Remove(tmp)
	if err := copyFile(source, tmp, 0o700); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := verifyFileSHA256(tmp, expectedSHA); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("voice: copied runner checksum: %w", err)
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("voice: install runner: %w", err)
	}
	return target, nil
}

func copyFile(source, target string, perm os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("voice: open runner source: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("voice: create runner copy: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("voice: copy runner: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("voice: close runner copy: %w", err)
	}
	return os.Chmod(target, perm)
}

func verifyFileSHA256(path string, want string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}
	if got := hex.EncodeToString(hasher.Sum(nil)); got != want {
		return fmt.Errorf("expected %s, got %s", want, got)
	}
	return nil
}

func firstExistingExecutable(paths []string) string {
	for _, path := range paths {
		if path == "" {
			continue
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() && executil.IsExecutable(path, info) {
			return path
		}
	}
	return ""
}

func firstExistingFile(paths []string) string {
	for _, path := range paths {
		if path == "" {
			continue
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}
